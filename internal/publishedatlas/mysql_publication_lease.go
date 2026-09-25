package publishedatlas

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type MySQLPublicationStore struct {
	db *sql.DB
}

func NewMySQLPublicationStore(db *sql.DB) *MySQLPublicationStore {
	return &MySQLPublicationStore{db: db}
}

func (store *MySQLPublicationStore) AcquireLease(ctx context.Context, runID, token string, ttl time.Duration) (LeaseAcquisition, error) {
	if store == nil || store.db == nil || len(runID) == 0 || len(runID) > 128 || len(token) != 64 || ttl < time.Second {
		return LeaseAcquisition{}, ErrCandidateInvalid
	}
	tx, err := store.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return LeaseAcquisition{}, err
	}
	defer tx.Rollback()
	var oldRun sql.NullString
	var oldExpiry sql.NullTime
	var now time.Time
	if err := tx.QueryRowContext(ctx, `SELECT run_id, expires_at, UTC_TIMESTAMP(6)
		FROM publication_lease WHERE singleton_id = 1 FOR UPDATE`).Scan(&oldRun, &oldExpiry, &now); err != nil {
		return LeaseAcquisition{}, fmt.Errorf("lock publication lease: %w", err)
	}
	if oldRun.Valid && oldExpiry.Valid && oldExpiry.Time.After(now) {
		return LeaseAcquisition{Kind: "IN_PROGRESS", RetryAfter: oldExpiry.Time.Sub(now)}, nil
	}
	if oldRun.Valid {
		receipt, found, err := publicationReceiptForRun(ctx, tx, oldRun.String)
		if err != nil {
			return LeaseAcquisition{}, fmt.Errorf("reconcile expired publication lease: %w", err)
		}
		if found {
			if _, err := tx.ExecContext(ctx, `UPDATE publication_lease SET run_id = NULL, token = NULL,
				expires_at = NULL WHERE singleton_id = 1`); err != nil {
				return LeaseAcquisition{}, err
			}
			if err := tx.Commit(); err != nil {
				return LeaseAcquisition{}, err
			}
			return LeaseAcquisition{Kind: "RECOVERED", Receipt: receipt}, nil
		}
	}
	if receipt, found, err := publicationReceiptForRun(ctx, tx, runID); err != nil {
		return LeaseAcquisition{}, err
	} else if found {
		return LeaseAcquisition{Kind: "RECOVERED", Receipt: receipt}, nil
	}
	var snapshot, registryToken, profile sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT snapshot_id, registry_token, profile
		FROM publication_active_pointer WHERE singleton_id = 1 FOR UPDATE`).Scan(&snapshot, &registryToken, &profile); err != nil {
		return LeaseAcquisition{}, fmt.Errorf("lock publication pointer: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE publication_lease SET run_id = ?, token = ?,
		expires_at = DATE_ADD(UTC_TIMESTAMP(6), INTERVAL ? MICROSECOND) WHERE singleton_id = 1`,
		runID, token, ttl.Microseconds()); err != nil {
		return LeaseAcquisition{}, fmt.Errorf("acquire publication lease: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return LeaseAcquisition{}, err
	}
	result := LeaseAcquisition{Kind: "ACQUIRED", Baseline: PublicationBaseline{
		SnapshotID: SnapshotID(snapshot.String), RegistryBaselineToken: registryToken.String,
		Profile: PublicationProfile(profile.String),
	}}
	if oldRun.Valid {
		result.SupersededRunID = oldRun.String
	}
	return result, nil
}

func publicationReceiptForRun(ctx context.Context, tx *sql.Tx, runID string) (PublicationReceipt, bool, error) {
	var receipt PublicationReceipt
	var snapshotID string
	err := tx.QueryRowContext(ctx, `SELECT run_id, snapshot_id, candidate_hash
		FROM publication_receipts WHERE run_id = ?`, runID).Scan(&receipt.RunID, &snapshotID, &receipt.CandidateHash)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicationReceipt{}, false, nil
	}
	if err != nil {
		return PublicationReceipt{}, false, err
	}
	receipt.SnapshotID = SnapshotID(snapshotID)
	return receipt, true, nil
}

func (store *MySQLPublicationStore) RenewLease(ctx context.Context, runID, token string, ttl time.Duration) (bool, error) {
	result, err := store.db.ExecContext(ctx, `UPDATE publication_lease SET
		expires_at = DATE_ADD(UTC_TIMESTAMP(6), INTERVAL ? MICROSECOND)
		WHERE singleton_id = 1 AND run_id = ? AND token = ? AND expires_at > UTC_TIMESTAMP(6)`,
		ttl.Microseconds(), runID, token)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (store *MySQLPublicationStore) ReleaseLease(ctx context.Context, runID, token string) error {
	_, err := store.db.ExecContext(ctx, `UPDATE publication_lease SET run_id = NULL, token = NULL,
		expires_at = NULL WHERE singleton_id = 1 AND run_id = ? AND token = ?`, runID, token)
	return err
}
