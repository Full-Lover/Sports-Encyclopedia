package atlasregistry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMediaRightsDecisionBindsExactFileRevisionAndOption(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	policy := validFactPolicy()
	policy.SourceID = fmt.Sprintf("rights-source-%d", time.Now().UnixNano())
	policy.Kind, policy.League, policy.FactGroups = CapabilityMedia, "", nil
	policy.MediaKinds = []MediaKind{MediaLogo}
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	file := MediaAssetBatch{SourceID: policy.SourceID, CapabilityKey: policy.CapabilityKey,
		AssetID: "logo-file", ContentHash: strings.Repeat("a", 64), Kind: MediaLogo,
		EntityID: "team-1", FileURL: "https://example.org/logo.png",
		SourcePageURL: "https://example.org/logo", FetchedAt: time.Now().UTC()}
	first := MediaRightsOption{Kind: RightsPublicDomain, SourceURL: file.SourcePageURL,
		PublicBasis: "old unsupported claim", RetrievedAt: time.Now().UTC()}
	second := MediaRightsOption{Kind: RightsOpenLicense, SourceURL: file.SourcePageURL,
		Author: "Example Photographer", LicenseName: "CC BY-SA", LicenseVersion: "4.0",
		LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0/", RetrievedAt: time.Now().UTC()}
	file.RightsOptions = []MediaRightsOption{first, second}
	decision := MediaRightsDecision{SourceID: file.SourceID, CapabilityKey: file.CapabilityKey,
		AssetID: file.AssetID, ContentHash: file.ContentHash, Kind: file.Kind,
		EntityID: file.EntityID, FileURL: file.FileURL, SourcePageURL: file.SourcePageURL,
		SelectedRights: second, ReviewedBy: "rights-curator", ReviewedAt: time.Now().UTC(), Active: true}
	if err := registry.InstallMediaRightsDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	check := func(batch MediaAssetBatch) *MediaRightsOption {
		t.Helper()
		tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		rights, err := loadApprovedMediaRights(ctx, tx, batch)
		if err != nil {
			t.Fatal(err)
		}
		return rights
	}
	if approved := check(file); approved == nil || approved.Kind != RightsOpenLicense {
		t.Fatalf("reviewed second option not selected: %#v", approved)
	}
	changedURL := file
	changedURL.FileURL = "https://example.org/other.png"
	if approved := check(changedURL); approved != nil {
		t.Fatalf("changed file URL inherited approval: %#v", approved)
	}
	changedRights := file
	changedRights.RightsOptions = []MediaRightsOption{first}
	if approved := check(changedRights); approved != nil {
		t.Fatalf("unreviewed option inherited approval: %#v", approved)
	}
	decision.Active = false
	decision.ReviewedAt = time.Now().UTC()
	if err := registry.InstallMediaRightsDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	if approved := check(file); approved != nil {
		t.Fatalf("revoked approval remained active: %#v", approved)
	}
}
