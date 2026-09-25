package atlasregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidMediaRightsDecision = errors.New("invalid registry media rights decision")

// MediaRightsDecision is trusted local curation for one exact file revision.
// A source observation cannot approve its own asserted rights.
type MediaRightsDecision struct {
	SourceID       string
	CapabilityKey  string
	AssetID        string
	ContentHash    string
	Kind           MediaKind
	EntityID       string
	FileURL        string
	SourcePageURL  string
	SelectedRights MediaRightsOption
	ReviewedBy     string
	ReviewedAt     time.Time
	Active         bool
}

func ValidateMediaRightsDecision(decision MediaRightsDecision) error {
	batch := MediaAssetBatch{SourceID: decision.SourceID, CapabilityKey: decision.CapabilityKey,
		AssetID: decision.AssetID, ContentHash: decision.ContentHash, Kind: decision.Kind,
		EntityID: decision.EntityID, FileURL: decision.FileURL,
		SourcePageURL: decision.SourcePageURL, FetchedAt: decision.ReviewedAt,
		RightsOptions: []MediaRightsOption{decision.SelectedRights}}
	if ValidateMediaAssetBatch("rights-review", batch) != nil ||
		!validText(decision.ReviewedBy, 160) || decision.SelectedRights.SourceURL != decision.SourcePageURL ||
		decision.SelectedRights.RetrievedAt.After(decision.ReviewedAt) {
		return ErrInvalidMediaRightsDecision
	}
	return nil
}

// InstallMediaRightsDecision is a trusted curation operation, deliberately
// outside the refresh-facing Registry interface.
func (registry *MySQLRegistry) InstallMediaRightsDecision(ctx context.Context, decision MediaRightsDecision) error {
	if registry == nil || registry.db == nil || ValidateMediaRightsDecision(decision) != nil {
		return ErrInvalidMediaRightsDecision
	}
	tx, err := registry.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	policy, err := loadUsagePolicy(ctx, tx, decision.SourceID, decision.CapabilityKey)
	if err != nil {
		return err
	}
	if policy.Kind != CapabilityMedia {
		return ErrPolicyNotAllowed
	}
	permitted := false
	for _, kind := range policy.MediaKinds {
		if kind == decision.Kind {
			permitted = true
			break
		}
	}
	if !permitted {
		return ErrPolicyNotAllowed
	}
	rights, err := json.Marshal(decision.SelectedRights)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO registry_media_rights_decisions
		(source_id, capability_key, asset_id, content_hash, media_kind, entity_id,
		file_url, source_page_url, selected_rights, reviewed_by, reviewed_at, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE media_kind = VALUES(media_kind), entity_id = VALUES(entity_id),
		file_url = VALUES(file_url), source_page_url = VALUES(source_page_url),
		selected_rights = VALUES(selected_rights), reviewed_by = VALUES(reviewed_by),
		reviewed_at = VALUES(reviewed_at), active = VALUES(active)`,
		decision.SourceID, decision.CapabilityKey, decision.AssetID, decision.ContentHash,
		decision.Kind, decision.EntityID, decision.FileURL, decision.SourcePageURL,
		rights, decision.ReviewedBy, decision.ReviewedAt.UTC(), decision.Active)
	if err != nil {
		return fmt.Errorf("save media rights decision: %w", err)
	}
	return tx.Commit()
}

func loadApprovedMediaRights(ctx context.Context, tx *sql.Tx, batch MediaAssetBatch) (*MediaRightsOption, error) {
	var kind MediaKind
	var entityID, fileURL, sourcePageURL string
	var raw []byte
	var active bool
	err := tx.QueryRowContext(ctx, `SELECT media_kind, entity_id, file_url, source_page_url,
		selected_rights, active FROM registry_media_rights_decisions
		WHERE source_id = ? AND capability_key = ? AND asset_id = ? AND content_hash = ? FOR UPDATE`,
		batch.SourceID, batch.CapabilityKey, batch.AssetID, batch.ContentHash).
		Scan(&kind, &entityID, &fileURL, &sourcePageURL, &raw, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !active || kind != batch.Kind || entityID != batch.EntityID ||
		fileURL != batch.FileURL || sourcePageURL != batch.SourcePageURL {
		return nil, nil
	}
	var approved MediaRightsOption
	if err := json.Unmarshal(raw, &approved); err != nil {
		return nil, fmt.Errorf("decode approved media rights: %w", err)
	}
	approvedJSON, err := json.Marshal(approved)
	if err != nil {
		return nil, err
	}
	for _, option := range batch.RightsOptions {
		candidateJSON, err := json.Marshal(option)
		if err != nil {
			return nil, err
		}
		if string(candidateJSON) == string(approvedJSON) {
			return &approved, nil
		}
	}
	return nil, nil
}
