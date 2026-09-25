package atlasregistry

import (
	"strings"
	"testing"
	"time"
)

func TestMediaRequiresRightsAndWebsitePolicy(t *testing.T) {
	content := CompiledRegistryContent{Teams: []CompiledTeam{{TeamID: "team-1"}}}
	batch := MediaAssetBatch{SourceID: "commons", CapabilityKey: "logo", Kind: MediaLogo,
		AssetID: "file-1", EntityID: "team-1", FileURL: "https://example.org/file.png",
		SourcePageURL: "https://example.org/page", ContentHash: strings.Repeat("a", 64),
		FetchedAt: time.Now().UTC()}
	policy := UsagePolicy{SourceID: batch.SourceID, CapabilityKey: batch.CapabilityKey,
		Kind: CapabilityMedia, MediaKinds: []MediaKind{MediaLogo}, Active: true, AllowWebsite: true}
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy}})
	if content.Teams[0].Logo != nil {
		t.Fatal("unlicensed media became displayable")
	}
	batch.RightsOptions = []MediaRightsOption{{Kind: RightsPublicDomain,
		SourceURL: batch.SourcePageURL, PublicBasis: "documented public domain",
		RetrievedAt: time.Now().UTC()}}
	policy.AllowWebsite = false
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy, ApprovedRights: &batch.RightsOptions[0]}})
	if content.Teams[0].Logo != nil {
		t.Fatal("policy-denied media became displayable")
	}
	policy.AllowWebsite = true
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy}})
	if content.Teams[0].Logo != nil {
		t.Fatal("source rights claim without a review became displayable")
	}
	attachMedia(&content, []stagedMedia{{Batch: batch, Policy: policy, ApprovedRights: &batch.RightsOptions[0]}})
	if content.Teams[0].Logo == nil || content.Teams[0].Logo.Rights.Kind != RightsPublicDomain {
		t.Fatalf("licensed media was not selected: %#v", content.Teams[0].Logo)
	}
}
