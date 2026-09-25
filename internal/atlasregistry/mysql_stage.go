package atlasregistry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidRunID     = errors.New("invalid registry run id")
	ErrRunSealed        = errors.New("registry run sealed")
	ErrRunAbandoned     = errors.New("registry run abandoned")
	ErrRevisionConflict = errors.New("registry revision key reused with different payload")
)

type ReconcileReceipt struct {
	RunID         string
	Kind          CapabilityKind
	DataGroup     DataGroupKind
	SourceID      string
	CapabilityKey string
	ContentHash   string
	FactCount     int
	Replayed      bool
}

// StageFacts atomically checks the run and its currently active source policy
// before an immutable batch becomes eligible for a later compilation.
func (registry *MySQLRegistry) StageFacts(ctx context.Context, runID string, batch FactBatch) (ReconcileReceipt, error) {
	if registry == nil || registry.db == nil || !validToken(runID, 128) || batch == nil {
		return ReconcileReceipt{}, ErrInvalidRunID
	}
	switch batch.(type) {
	case TeamIdentityBatch, VenueBatch, LeaderBatch, RosterBatch:
	default:
		return ReconcileReceipt{}, ErrInvalidTeamIdentityBatch
	}
	if err := batch.validate(runID); err != nil {
		return ReconcileReceipt{}, err
	}
	metadata := batch.metadata()
	payload, payloadHash, err := encodeBatch(batch)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	receipt := ReconcileReceipt{RunID: runID, Kind: CapabilityFacts, DataGroup: batch.factGroup(),
		SourceID: metadata.SourceID, CapabilityKey: metadata.CapabilityKey, ContentHash: metadata.ContentHash,
		FactCount: factCount(batch)}
	return registry.withOpenRun(ctx, runID, func(tx *sql.Tx) (ReconcileReceipt, error) {
		policy, err := loadUsagePolicy(ctx, tx, metadata.SourceID, metadata.CapabilityKey)
		if err != nil {
			return ReconcileReceipt{}, err
		}
		if !policyAllowsFact(policy, batch) {
			return ReconcileReceipt{}, ErrPolicyNotAllowed
		}
		var storedHash string
		err = tx.QueryRowContext(ctx, `SELECT payload_hash FROM registry_fact_batches
			WHERE run_id = ? AND source_id = ? AND capability_key = ? AND group_kind = ?
			AND league = ? AND season = ? AND content_hash = ?`, runID, metadata.SourceID,
			metadata.CapabilityKey, batch.factGroup(), metadata.League, metadata.Season,
			metadata.ContentHash).Scan(&storedHash)
		if err == nil {
			if storedHash != payloadHash {
				return ReconcileReceipt{}, ErrRevisionConflict
			}
			receipt.Replayed = true
			return receipt, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ReconcileReceipt{}, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO registry_fact_batches
			(run_id, source_id, capability_key, group_kind, league, season, source_url,
			fetched_at, content_hash, payload_hash, complete_pagination, payload)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, runID, metadata.SourceID,
			metadata.CapabilityKey, batch.factGroup(), metadata.League, metadata.Season,
			metadata.SourceURL, metadata.FetchedAt.UTC(), metadata.ContentHash, payloadHash,
			metadata.CompletePagination, payload)
		if err != nil {
			return ReconcileReceipt{}, fmt.Errorf("stage registry fact batch: %w", err)
		}
		return receipt, nil
	})
}

func (registry *MySQLRegistry) StageMedia(ctx context.Context, runID string, batch MediaAssetBatch) (ReconcileReceipt, error) {
	if registry == nil || registry.db == nil || !validToken(runID, 128) {
		return ReconcileReceipt{}, ErrInvalidRunID
	}
	if err := ValidateMediaAssetBatch(runID, batch); err != nil {
		return ReconcileReceipt{}, err
	}
	payload, payloadHash, err := encodeBatch(batch)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	receipt := ReconcileReceipt{RunID: runID, Kind: CapabilityMedia, SourceID: batch.SourceID,
		CapabilityKey: batch.CapabilityKey, ContentHash: batch.ContentHash, FactCount: 1}
	return registry.withOpenRun(ctx, runID, func(tx *sql.Tx) (ReconcileReceipt, error) {
		policy, err := loadUsagePolicy(ctx, tx, batch.SourceID, batch.CapabilityKey)
		if err != nil {
			return ReconcileReceipt{}, err
		}
		if !policyAllowsMedia(policy, batch) {
			return ReconcileReceipt{}, ErrPolicyNotAllowed
		}
		var storedHash string
		err = tx.QueryRowContext(ctx, `SELECT payload_hash FROM registry_media_batches
			WHERE run_id = ? AND source_id = ? AND capability_key = ? AND asset_id = ?
			AND content_hash = ?`, runID, batch.SourceID, batch.CapabilityKey,
			batch.AssetID, batch.ContentHash).Scan(&storedHash)
		if err == nil {
			if storedHash != payloadHash {
				return ReconcileReceipt{}, ErrRevisionConflict
			}
			receipt.Replayed = true
			return receipt, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ReconcileReceipt{}, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO registry_media_batches
			(run_id, source_id, capability_key, asset_id, media_kind, entity_id,
			content_hash, payload_hash, payload) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, runID, batch.SourceID,
			batch.CapabilityKey, batch.AssetID, batch.Kind, batch.EntityID,
			batch.ContentHash, payloadHash, payload)
		if err != nil {
			return ReconcileReceipt{}, fmt.Errorf("stage registry media batch: %w", err)
		}
		return receipt, nil
	})
}

func (registry *MySQLRegistry) withOpenRun(ctx context.Context, runID string, work func(*sql.Tx) (ReconcileReceipt, error)) (ReconcileReceipt, error) {
	tx, err := registry.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return ReconcileReceipt{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO registry_runs (run_id) VALUES (?)`, runID); err != nil {
		return ReconcileReceipt{}, err
	}
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM registry_runs WHERE run_id = ? FOR UPDATE`, runID).Scan(&state); err != nil {
		return ReconcileReceipt{}, err
	}
	if state == "SEALED" {
		return ReconcileReceipt{}, ErrRunSealed
	}
	if state == "ABANDONED" {
		return ReconcileReceipt{}, ErrRunAbandoned
	}
	receipt, err := work(tx)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReconcileReceipt{}, err
	}
	return receipt, nil
}

func (registry *MySQLRegistry) AbandonRun(ctx context.Context, runID string) error {
	if registry == nil || registry.db == nil || !validToken(runID, 128) {
		return ErrInvalidRunID
	}
	tx, err := registry.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO registry_runs (run_id) VALUES (?)`, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE registry_runs SET state = 'ABANDONED',
		abandoned_at = COALESCE(abandoned_at, UTC_TIMESTAMP(6)) WHERE run_id = ?`, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func encodeBatch(batch any) ([]byte, string, error) {
	payload, err := json.Marshal(batch)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(payload)
	return payload, hex.EncodeToString(sum[:]), nil
}

func factCount(batch FactBatch) int {
	switch value := batch.(type) {
	case TeamIdentityBatch:
		return len(value.Teams)
	case VenueBatch:
		return len(value.Venues)
	case LeaderBatch:
		return len(value.Leaders)
	case RosterBatch:
		return len(value.Rosters)
	default:
		return 0
	}
}
