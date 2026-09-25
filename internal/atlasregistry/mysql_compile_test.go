package atlasregistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMySQLRegistryCompileLifecycle(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	unique := time.Now().UnixNano()
	runID := fmt.Sprintf("compile-run-%d", unique)
	policy := validFactPolicy()
	policy.SourceID = fmt.Sprintf("compile-source-%d", unique)
	policy.FactGroups = []DataGroupKind{GroupIdentity, GroupVenue}
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	identity := validIdentityBatch()
	identity.SourceID = policy.SourceID
	identity.Teams[0].TeamID = fmt.Sprintf("nba-compile-team-%d", unique)
	identity.Teams[0].OfficialName = fmt.Sprintf("Compile Celtics %d", unique)
	identity.SeasonEvidence = &OfficialSeasonEvidence{League: LeagueNBA, Season: identity.Season,
		EffectiveAt:       time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		RosterPublishedAt: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.nba.com/teams",
		Groups: []OfficialGroupEvidence{{Name: "Eastern Conference", Divisions: []string{"Atlantic Division"}},
			{Name: "Western Conference", Divisions: []string{"Pacific Division"}}}}
	venue := VenueBatch{FactMetadata: identity.FactMetadata, Venues: []VenueFact{{
		TeamID: identity.Teams[0].TeamID, VenueID: "td-garden", OfficialName: "TD Garden",
		City: "Boston", Region: "Massachusetts", CountryCode: "US",
		Latitude: 42.366303, Longitude: -71.062228, IsPrimary: true, RegularGameCapacity: 19156}}}
	venue.ContentHash = strings.Repeat("b", 64)
	for _, batch := range []FactBatch{identity, venue} {
		if _, err := registry.StageFacts(ctx, runID, batch); err != nil {
			t.Fatal(err)
		}
	}
	mediaPolicy := validFactPolicy()
	mediaPolicy.SourceID = fmt.Sprintf("compile-media-%d", unique)
	mediaPolicy.Kind, mediaPolicy.League, mediaPolicy.FactGroups = CapabilityMedia, "", nil
	mediaPolicy.MediaKinds = []MediaKind{MediaLogo}
	if err := registry.InstallPolicy(ctx, mediaPolicy); err != nil {
		t.Fatal(err)
	}
	logo := MediaAssetBatch{SourceID: mediaPolicy.SourceID, CapabilityKey: mediaPolicy.CapabilityKey,
		Kind: MediaLogo, AssetID: "logo-1", EntityID: identity.Teams[0].TeamID,
		FileURL: "https://example.org/logo.png", SourcePageURL: "https://example.org/logo",
		ContentHash: strings.Repeat("d", 64), FetchedAt: time.Now().UTC(),
		RightsOptions: []MediaRightsOption{{Kind: RightsPublicDomain,
			SourceURL: "https://example.org/logo", PublicBasis: "reviewed public-domain evidence",
			RetrievedAt: time.Now().UTC()}}}
	if _, err := registry.StageMedia(ctx, runID, logo); err != nil {
		t.Fatal(err)
	}
	request := CompileRequest{RunID: runID, Profile: ProfilePreview,
		ConfigurationFingerprint: strings.Repeat("c", 64),
		RequestedAt:              time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}
	first, err := registry.CompilePublicationContent(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.SchemaVersion != 2 || !validHash(first.NextBaselineToken) || len(first.Content.Teams) != 1 ||
		first.Content.Teams[0].Venue.Value == nil || first.Content.Teams[0].Identity.Value == nil ||
		first.Content.Teams[0].Visual.Kind != TeamVisualAbbreviation ||
		first.Content.Teams[0].Visual.Abbreviation != "BOS" {
		t.Fatalf("compiled content = %#v", first)
	}
	replayed, err := registry.CompilePublicationContent(ctx, request)
	if err != nil || replayed.NextBaselineToken != first.NextBaselineToken {
		t.Fatalf("replay = %#v, %v", replayed, err)
	}
	changedRequest := request
	changedRequest.RequestedAt = changedRequest.RequestedAt.Add(time.Second)
	if _, err := registry.CompilePublicationContent(ctx, changedRequest); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("same key with changed requestedAt = %v", err)
	}
	if _, err := registry.StageFacts(ctx, runID, venue); !errors.Is(err, ErrRunSealed) {
		t.Fatalf("stage after compile = %v", err)
	}
	decision := MediaRightsDecision{SourceID: logo.SourceID, CapabilityKey: logo.CapabilityKey,
		AssetID: logo.AssetID, ContentHash: logo.ContentHash, Kind: logo.Kind,
		EntityID: logo.EntityID, FileURL: logo.FileURL, SourcePageURL: logo.SourcePageURL,
		SelectedRights: logo.RightsOptions[0], ReviewedBy: "trusted-test-curator",
		ReviewedAt: time.Now().UTC(), Active: true}
	if err := registry.InstallMediaRightsDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	rebase := request
	rebase.BaselineToken = first.NextBaselineToken
	rebase.ConfigurationFingerprint = strings.Repeat("d", 64)
	second, err := registry.CompilePublicationContent(ctx, rebase)
	if err != nil || second.NextBaselineToken == first.NextBaselineToken ||
		len(second.Content.Teams) != 1 || second.Content.Teams[0].Visual.Kind != TeamVisualMedia {
		t.Fatalf("rebase = %#v, %v", second, err)
	}
	decision.Active = false
	decision.ReviewedAt = time.Now().UTC()
	if err := registry.InstallMediaRightsDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	rebase.BaselineToken = second.NextBaselineToken
	rebase.ConfigurationFingerprint = strings.Repeat("6", 64)
	withoutDecision, err := registry.CompilePublicationContent(ctx, rebase)
	if err != nil || withoutDecision.Content.Teams[0].Visual.Kind != TeamVisualAbbreviation {
		t.Fatalf("revoked per-file rights = %#v, %v", withoutDecision.Content.Teams[0].Visual, err)
	}
	mediaPolicy.Active = false
	if err := registry.InstallPolicy(ctx, mediaPolicy); err != nil {
		t.Fatal(err)
	}
	rebase.BaselineToken = withoutDecision.NextBaselineToken
	rebase.ConfigurationFingerprint = strings.Repeat("e", 64)
	withoutLogo, err := registry.CompilePublicationContent(ctx, rebase)
	if err != nil || withoutLogo.Content.Teams[0].Visual.Kind != TeamVisualAbbreviation {
		t.Fatalf("revoked media policy = %#v, %v", withoutLogo.Content.Teams[0].Visual, err)
	}
	renameRun := fmt.Sprintf("rename-run-%d", unique)
	renamed := identity
	renamed.Teams = append([]TeamIdentityFact(nil), identity.Teams...)
	renamed.Teams[0].OfficialName = fmt.Sprintf("Renamed Celtics %d", unique)
	renamed.ContentHash = strings.Repeat("1", 64)
	renamed.FetchedAt = identity.FetchedAt.Add(time.Hour)
	renamed.SeasonEvidence = nil
	if _, err := registry.StageFacts(ctx, renameRun, renamed); err != nil {
		t.Fatal(err)
	}
	renameRequest := request
	renameRequest.RunID = renameRun
	renameRequest.BaselineToken = withoutLogo.NextBaselineToken
	renameRequest.ConfigurationFingerprint = strings.Repeat("2", 64)
	renamedResult, err := registry.CompilePublicationContent(ctx, renameRequest)
	if err != nil {
		t.Fatal(err)
	}
	renamedTeam := renamedResult.Content.Teams[0]
	if renamedTeam.Slug == first.Content.Teams[0].Slug || len(renamedTeam.SlugHistory) != 1 ||
		renamedTeam.SlugHistory[0] != first.Content.Teams[0].Slug ||
		len(renamedTeam.Aliases) != 1 || renamedTeam.Aliases[0] != identity.Teams[0].OfficialName {
		t.Fatalf("slug history = %#v", renamedTeam)
	}
	contenderRun := fmt.Sprintf("slug-contender-%d", unique)
	contender := identity
	contender.Teams = append([]TeamIdentityFact(nil), identity.Teams...)
	contender.Teams[0].TeamID = fmt.Sprintf("nba-contender-%d", unique)
	contender.ContentHash = strings.Repeat("3", 64)
	if _, err := registry.StageFacts(ctx, contenderRun, contender); err != nil {
		t.Fatal(err)
	}
	contenderRequest := request
	contenderRequest.RunID = contenderRun
	contenderRequest.ConfigurationFingerprint = strings.Repeat("4", 64)
	if _, err := registry.CompilePublicationContent(ctx, contenderRequest); !errors.Is(err, ErrSlugConflict) {
		t.Fatalf("reusing old slug = %v", err)
	}
	policy.Active = false
	if err := registry.InstallPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	revokedRequest := request
	revokedRequest.RunID = fmt.Sprintf("revoked-facts-run-%d", unique)
	revokedRequest.BaselineToken = renamedResult.NextBaselineToken
	revokedRequest.ConfigurationFingerprint = strings.Repeat("5", 64)
	if _, err := registry.CompilePublicationContent(ctx, revokedRequest); !errors.Is(err, ErrCompilationRejected) {
		t.Fatalf("revoked identity source remained publishable = %v", err)
	}
	if err := registry.AbandonRun(ctx, runID); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.CompilePublicationContent(ctx, request); !errors.Is(err, ErrRunAbandoned) {
		t.Fatalf("compile abandoned run = %v", err)
	}
}

func TestMySQLRegistryCompileRejectsMissingBaselineAndSealsRun(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	registry := NewMySQLRegistry(db)
	runID := fmt.Sprintf("missing-baseline-%d", time.Now().UnixNano())
	request := CompileRequest{RunID: runID, BaselineToken: strings.Repeat("e", 64),
		Profile: ProfilePreview, ConfigurationFingerprint: strings.Repeat("f", 64),
		RequestedAt: time.Now().UTC()}
	if _, err := registry.CompilePublicationContent(ctx, request); !errors.Is(err, ErrBaselineMissing) {
		t.Fatalf("missing baseline = %v", err)
	}
	if _, err := registry.StageFacts(ctx, runID, validIdentityBatch()); !errors.Is(err, ErrRunSealed) {
		t.Fatalf("run not sealed on first compile attempt: %v", err)
	}
}
