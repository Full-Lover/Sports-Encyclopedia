package atlasregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type storedBatch struct {
	group, sourceID, capabilityKey, hash string
	payload                              []byte
}

func loadStagedFacts(ctx context.Context, tx *sql.Tx, runID string) ([]stagedFact, error) {
	rows, err := tx.QueryContext(ctx, `SELECT group_kind, source_id, capability_key, payload_hash, payload
		FROM registry_fact_batches WHERE run_id = ? ORDER BY batch_id`, runID)
	if err != nil {
		return nil, err
	}
	var stored []storedBatch
	for rows.Next() {
		var item storedBatch
		if err := rows.Scan(&item.group, &item.sourceID, &item.capabilityKey, &item.hash, &item.payload); err != nil {
			rows.Close()
			return nil, err
		}
		stored = append(stored, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	result := make([]stagedFact, 0, len(stored))
	for _, item := range stored {
		var batch FactBatch
		switch DataGroupKind(item.group) {
		case GroupIdentity:
			var value TeamIdentityBatch
			err = json.Unmarshal(item.payload, &value)
			batch = value
		case GroupVenue:
			var value VenueBatch
			err = json.Unmarshal(item.payload, &value)
			batch = value
		case GroupLeader:
			var value LeaderBatch
			err = json.Unmarshal(item.payload, &value)
			batch = value
		case GroupRoster:
			var value RosterBatch
			err = json.Unmarshal(item.payload, &value)
			batch = value
		default:
			return nil, fmt.Errorf("unknown staged registry fact group %q", item.group)
		}
		if err != nil {
			return nil, fmt.Errorf("decode staged registry facts: %w", err)
		}
		if err := batch.validate(runID); err != nil {
			return nil, err
		}
		_, digest, err := encodeBatch(batch)
		if err != nil {
			return nil, err
		}
		metadata := batch.metadata()
		if digest != item.hash || metadata.SourceID != item.sourceID || metadata.CapabilityKey != item.capabilityKey || string(batch.factGroup()) != item.group {
			return nil, ErrRevisionConflict
		}
		policy, err := loadUsagePolicy(ctx, tx, item.sourceID, item.capabilityKey)
		if errors.Is(err, ErrPolicyNotAllowed) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, stagedFact{Batch: batch, Policy: policy})
	}
	return result, nil
}

func loadStagedMedia(ctx context.Context, tx *sql.Tx, runID string) ([]stagedMedia, error) {
	rows, err := tx.QueryContext(ctx, `SELECT source_id, capability_key, payload_hash, payload
		FROM registry_media_batches WHERE run_id = ? ORDER BY batch_id`, runID)
	if err != nil {
		return nil, err
	}
	var stored []storedBatch
	for rows.Next() {
		var item storedBatch
		if err := rows.Scan(&item.sourceID, &item.capabilityKey, &item.hash, &item.payload); err != nil {
			rows.Close()
			return nil, err
		}
		stored = append(stored, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	result := make([]stagedMedia, 0, len(stored))
	for _, item := range stored {
		var batch MediaAssetBatch
		if err := json.Unmarshal(item.payload, &batch); err != nil {
			return nil, err
		}
		if err := ValidateMediaAssetBatch(runID, batch); err != nil {
			return nil, err
		}
		_, digest, err := encodeBatch(batch)
		if err != nil {
			return nil, err
		}
		if digest != item.hash || batch.SourceID != item.sourceID || batch.CapabilityKey != item.capabilityKey {
			return nil, ErrRevisionConflict
		}
		policy, err := loadUsagePolicy(ctx, tx, item.sourceID, item.capabilityKey)
		if errors.Is(err, ErrPolicyNotAllowed) {
			continue
		}
		if err != nil {
			return nil, err
		}
		approved, err := loadApprovedMediaRights(ctx, tx, batch)
		if err != nil {
			return nil, err
		}
		result = append(result, stagedMedia{Batch: batch, Policy: policy, ApprovedRights: approved})
	}
	return result, nil
}

func loadInheritedMedia(ctx context.Context, tx *sql.Tx, baseline CompiledRegistryContent) ([]stagedMedia, error) {
	seen := make(map[[2]string]struct{})
	var assets []SelectedMedia
	for _, team := range baseline.Teams {
		if team.Visual.Media != nil {
			assets = append(assets, *team.Visual.Media)
		}
		if team.VenuePhoto.Media != nil {
			assets = append(assets, *team.VenuePhoto.Media)
		}
		for _, photo := range team.PlayerPhotos {
			if photo.Photo.Media != nil {
				assets = append(assets, *photo.Photo.Media)
			}
		}
	}
	var result []stagedMedia
	for _, asset := range assets {
		key := [2]string{asset.SourceID, asset.AssetID}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		batch := MediaAssetBatch{SourceID: asset.SourceID, CapabilityKey: asset.CapabilityKey,
			Kind: asset.Kind, AssetID: asset.AssetID, EntityID: asset.EntityID,
			FileURL: asset.FileURL, SourcePageURL: asset.SourcePageURL,
			ContentHash: asset.ContentHash, FetchedAt: asset.FetchedAt,
			RightsOptions: []MediaRightsOption{asset.Rights}}
		if err := ValidateMediaAssetBatch("inherited", batch); err != nil {
			return nil, fmt.Errorf("invalid inherited media: %w", err)
		}
		policy, err := loadUsagePolicy(ctx, tx, batch.SourceID, batch.CapabilityKey)
		if errors.Is(err, ErrPolicyNotAllowed) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if policy.AllowWebsite && policyAllowsMedia(policy, batch) {
			approved, err := loadApprovedMediaRights(ctx, tx, batch)
			if err != nil {
				return nil, err
			}
			result = append(result, stagedMedia{Batch: batch, Policy: policy, ApprovedRights: approved})
		}
	}
	return result, nil
}
