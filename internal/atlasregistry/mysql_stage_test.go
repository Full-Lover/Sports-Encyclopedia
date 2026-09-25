package atlasregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMySQLRegistryStageFactsAndRunStates(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	batch := validIdentityBatch()
	runID := fmt.Sprintf("facts-run-%d", time.Now().UnixNano())
	if _, err := registry.StageFacts(ctx, runID, batch); !errors.Is(err, ErrPolicyNotAllowed) {
		t.Fatalf("unknown policy error = %v", err)
	}
	policy := validFactPolicy()
	policy.SourceID = fmt.Sprintf("facts-source-%d", time.Now().UnixNano())
	batch.SourceID = policy.SourceID
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	first, err := registry.StageFacts(ctx, runID, batch)
	if err != nil || first.Replayed || first.FactCount != 1 {
		t.Fatalf("first stage = %#v, %v", first, err)
	}
	replay, err := registry.StageFacts(ctx, runID, batch)
	if err != nil || !replay.Replayed {
		t.Fatalf("idempotent replay = %#v, %v", replay, err)
	}
	conflicting := batch
	conflicting.Teams = append([]TeamIdentityFact(nil), batch.Teams...)
	conflicting.Teams[0].OfficialName = "Different Team"
	if _, err := registry.StageFacts(ctx, runID, conflicting); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("same revision key with different payload = %v", err)
	}
	policy.Active = false
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	batch.ContentHash = strings.Repeat("b", 64)
	if _, err := registry.StageFacts(ctx, runID, batch); !errors.Is(err, ErrPolicyNotAllowed) {
		t.Fatalf("revoked capability error = %v", err)
	}
	if err := registry.AbandonRun(ctx, runID); err != nil {
		t.Fatal(err)
	}
	if err := registry.AbandonRun(ctx, runID); err != nil {
		t.Fatalf("abandon replay: %v", err)
	}
	if _, err := registry.StageFacts(ctx, runID, batch); !errors.Is(err, ErrRunAbandoned) {
		t.Fatalf("stage after abandonment = %v", err)
	}
}

func TestMySQLRegistryStageMediaAndSeal(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	policy := validFactPolicy()
	policy.SourceID = fmt.Sprintf("media-source-%d", time.Now().UnixNano())
	policy.Kind, policy.League, policy.FactGroups = CapabilityMedia, "", nil
	policy.MediaKinds = []MediaKind{MediaLogo}
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	batch := MediaAssetBatch{
		SourceID: policy.SourceID, CapabilityKey: policy.CapabilityKey,
		Kind: MediaLogo, AssetID: "example-logo", EntityID: "nba-boston-celtics",
		FileURL:       "https://upload.wikimedia.org/example.png",
		SourcePageURL: "https://commons.wikimedia.org/wiki/File:Example.png",
		ContentHash:   strings.Repeat("c", 64), FetchedAt: time.Now().UTC(),
	}
	runID := fmt.Sprintf("media-run-%d", time.Now().UnixNano())
	first, err := registry.StageMedia(ctx, runID, batch)
	if err != nil || first.Replayed {
		t.Fatalf("first media stage = %#v, %v", first, err)
	}
	replay, err := registry.StageMedia(ctx, runID, batch)
	if err != nil || !replay.Replayed {
		t.Fatalf("media replay = %#v, %v", replay, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE registry_runs SET state = 'SEALED' WHERE run_id = ?`, runID); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.StageMedia(ctx, runID, batch); !errors.Is(err, ErrRunSealed) {
		t.Fatalf("stage after seal = %v", err)
	}
}
