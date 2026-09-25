package atlasregistry

import (
	"strings"
	"testing"
	"time"
)

func TestMediaRequiresRightsAndWebsitePolicy(t *testing.T) {
	identity := TeamIdentityFact{TeamID: "team-1", OfficialAbbreviation: "ONE"}
	content := CompiledRegistryContent{Teams: []CompiledTeam{{TeamID: "team-1",
		Identity: CompiledGroup[TeamIdentityFact]{State: StateCurrent, Value: &identity}}}}
	batch := MediaAssetBatch{SourceID: "commons", CapabilityKey: "logo", Kind: MediaLogo,
		AssetID: "file-1", EntityID: "team-1", FileURL: "https://example.org/file.png",
		SourcePageURL: "https://example.org/page", ContentHash: strings.Repeat("a", 64),
		FetchedAt: time.Now().UTC()}
	policy := UsagePolicy{SourceID: batch.SourceID, CapabilityKey: batch.CapabilityKey,
		Kind: CapabilityMedia, MediaKinds: []MediaKind{MediaLogo}, Active: true, AllowWebsite: true}
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy}})
	if content.Teams[0].Visual.Kind != TeamVisualAbbreviation ||
		content.Teams[0].Visual.Abbreviation != "ONE" {
		t.Fatal("unlicensed media became displayable")
	}
	batch.RightsOptions = []MediaRightsOption{{Kind: RightsPublicDomain,
		SourceURL: batch.SourcePageURL, PublicBasis: "documented public domain",
		RetrievedAt: time.Now().UTC()}}
	policy.AllowWebsite = false
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy, ApprovedRights: &batch.RightsOptions[0]}})
	if content.Teams[0].Visual.Kind != TeamVisualAbbreviation {
		t.Fatal("policy-denied media became displayable")
	}
	policy.AllowWebsite = true
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy}})
	if content.Teams[0].Visual.Kind != TeamVisualAbbreviation {
		t.Fatal("source rights claim without a review became displayable")
	}
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy, ApprovedRights: &batch.RightsOptions[0]}})
	if content.Teams[0].Visual.Kind != TeamVisualMedia || content.Teams[0].Visual.Media == nil ||
		content.Teams[0].Visual.Media.Rights.Kind != RightsPublicDomain {
		t.Fatalf("licensed media was not selected: %#v", content.Teams[0].Visual)
	}
}

func TestMissingPhotosBecomeExplicitPlaceholders(t *testing.T) {
	identity := TeamIdentityFact{TeamID: "team-1", OfficialAbbreviation: "ONE"}
	venue := VenueFact{TeamID: "team-1", VenueID: "arena-1"}
	roster := TeamRosterFact{TeamID: "team-1", Entries: []RosterEntryFact{{PersonID: "p2"}, {PersonID: "p1"}}}
	content := CompiledRegistryContent{Teams: []CompiledTeam{{TeamID: "team-1",
		Identity: CompiledGroup[TeamIdentityFact]{State: StateCurrent, Value: &identity},
		Venue:    CompiledGroup[VenueFact]{State: StateCurrent, Value: &venue},
		Roster:   CompiledGroup[TeamRosterFact]{State: StateCurrent, Value: &roster}}}}
	attachMedia(&content, nil)
	team := content.Teams[0]
	if team.Visual.Kind != TeamVisualAbbreviation || team.VenuePhoto.Kind != PhotoPlaceholder ||
		len(team.PlayerPhotos) != 2 || team.PlayerPhotos[0].PersonID != "p1" ||
		team.PlayerPhotos[0].Photo.Kind != PhotoPlaceholder {
		t.Fatalf("explicit media fallbacks = %#v", team)
	}
}
