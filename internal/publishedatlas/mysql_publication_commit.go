package publishedatlas

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func (store *MySQLPublicationStore) StageCandidate(ctx context.Context, candidate PublicationCandidate, hash string) error {
	if store == nil || store.db == nil || len(hash) != 64 {
		return ErrCandidateInvalid
	}
	if err := validatePublicationCandidate(candidate, candidate.RunID, PublicationBaseline{
		SnapshotID: candidate.BaseSnapshotID, RegistryBaselineToken: candidate.BaseRegistryBaselineToken,
	}); err != nil {
		return err
	}
	computedHash, err := candidateHash(candidate)
	if err != nil || computedHash != hash {
		return ErrCandidateInvalid
	}
	mapJSON, err := json.Marshal(candidate.Map)
	if err != nil {
		return err
	}
	_, err = store.db.ExecContext(ctx, `INSERT INTO publication_snapshots
		(snapshot_id, run_id, candidate_hash, base_snapshot_id, base_registry_token,
		next_registry_token, profile, schema_version, map_document, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'CANDIDATE')
		ON DUPLICATE KEY UPDATE snapshot_id = snapshot_id`, candidate.Map.SnapshotID, candidate.RunID,
		hash, nullablePublicationString(string(candidate.BaseSnapshotID)),
		nullablePublicationString(candidate.BaseRegistryBaselineToken), candidate.NextRegistryBaselineToken,
		candidate.Profile, candidate.SchemaVersion, mapJSON)
	if err != nil {
		return fmt.Errorf("stage publication candidate: %w", err)
	}
	var storedSnapshot, storedRun string
	if err := store.db.QueryRowContext(ctx, `SELECT snapshot_id, run_id FROM publication_snapshots
		WHERE candidate_hash = ?`, hash).Scan(&storedSnapshot, &storedRun); err != nil {
		return fmt.Errorf("read staged candidate: %w", err)
	}
	if storedSnapshot != string(candidate.Map.SnapshotID) || storedRun != candidate.RunID {
		return ErrCandidateInvalid
	}
	return nil
}

func (store *MySQLPublicationStore) CommitCandidate(ctx context.Context, runID, token string, candidate PublicationCandidate, hash string) (CommitOutcome, error) {
	computedHash, err := candidateHash(candidate)
	if err != nil || computedHash != hash || candidate.RunID != runID {
		return CommitOutcome{}, ErrCandidateInvalid
	}
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return CommitOutcome{}, err
	}
	defer tx.Rollback()
	var heldRun, heldToken sql.NullString
	var expiry sql.NullTime
	var now time.Time
	if err := tx.QueryRowContext(ctx, `SELECT run_id, token, expires_at, UTC_TIMESTAMP(6)
		FROM publication_lease WHERE singleton_id = 1 FOR UPDATE`).Scan(&heldRun, &heldToken, &expiry, &now); err != nil {
		return CommitOutcome{}, fmt.Errorf("lock publication lease: %w", err)
	}
	if heldRun.String != runID || heldToken.String != token || !expiry.Valid || !expiry.Time.After(now) {
		return CommitOutcome{}, ErrLeaseLost
	}
	var snapshot, registryToken, profile sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT snapshot_id, registry_token, profile
		FROM publication_active_pointer WHERE singleton_id = 1 FOR UPDATE`).Scan(&snapshot, &registryToken, &profile); err != nil {
		return CommitOutcome{}, fmt.Errorf("lock publication pointer: %w", err)
	}
	current := PublicationBaseline{
		SnapshotID: SnapshotID(snapshot.String), RegistryBaselineToken: registryToken.String,
		Profile: PublicationProfile(profile.String),
	}
	if current.SnapshotID != candidate.BaseSnapshotID || current.RegistryBaselineToken != candidate.BaseRegistryBaselineToken {
		return CommitOutcome{BaseChanged: true, CurrentBaseline: current}, nil
	}
	var allowedVersion int
	var paused bool
	if err := tx.QueryRowContext(ctx, `SELECT allowed_version, paused FROM publication_schema_gate
		WHERE singleton_id = 1 FOR UPDATE`).Scan(&allowedVersion, &paused); err != nil {
		return CommitOutcome{}, fmt.Errorf("lock publication schema gate: %w", err)
	}
	if paused {
		return CommitOutcome{}, ErrPublicationPaused
	}
	if candidate.SchemaVersion != allowedVersion {
		return CommitOutcome{}, ErrSchemaUnsupported
	}
	if current.Profile == ProfileV1 && candidate.Profile == ProfilePreview {
		return CommitOutcome{}, ErrProfileDowngrade
	}
	var stagedRun, stagedHash, status string
	if err := tx.QueryRowContext(ctx, `SELECT run_id, candidate_hash, status FROM publication_snapshots
		WHERE snapshot_id = ? FOR UPDATE`, candidate.Map.SnapshotID).Scan(&stagedRun, &stagedHash, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CommitOutcome{}, ErrCandidateInvalid
		}
		return CommitOutcome{}, fmt.Errorf("lock staged candidate: %w", err)
	}
	if stagedRun != runID || stagedHash != hash || status != "CANDIDATE" {
		return CommitOutcome{}, ErrCandidateInvalid
	}
	if _, err := tx.ExecContext(ctx, `UPDATE publication_snapshots SET status = 'PUBLISHED',
		published_at = UTC_TIMESTAMP(6) WHERE snapshot_id = ? AND status = 'CANDIDATE'`, candidate.Map.SnapshotID); err != nil {
		return CommitOutcome{}, fmt.Errorf("publish candidate document: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE publication_active_pointer SET snapshot_id = ?,
		registry_token = ?, profile = ?, schema_version = ? WHERE singleton_id = 1
		AND snapshot_id <=> ? AND registry_token <=> ?`, candidate.Map.SnapshotID,
		candidate.NextRegistryBaselineToken, candidate.Profile, candidate.SchemaVersion,
		nullablePublicationString(string(candidate.BaseSnapshotID)),
		nullablePublicationString(candidate.BaseRegistryBaselineToken))
	if err != nil {
		return CommitOutcome{}, fmt.Errorf("switch active publication pointer: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return CommitOutcome{}, err
	}
	if affected != 1 {
		return CommitOutcome{}, ErrBaseChanged
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO publication_receipts
		(run_id, snapshot_id, candidate_hash) VALUES (?, ?, ?)`, runID, candidate.Map.SnapshotID, hash); err != nil {
		return CommitOutcome{}, fmt.Errorf("record publication receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return CommitOutcome{}, fmt.Errorf("commit publication: %w", err)
	}
	return CommitOutcome{Receipt: PublicationReceipt{
		RunID: runID, SnapshotID: candidate.Map.SnapshotID, CandidateHash: hash,
	}}, nil
}

func nullablePublicationString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
