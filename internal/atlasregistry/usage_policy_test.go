package atlasregistry

import (
	"errors"
	"testing"
	"time"
)

func validFactPolicy() UsagePolicy {
	return UsagePolicy{
		SourceID: "nba-official", CapabilityKey: "teams.identity", Kind: CapabilityFacts,
		League: LeagueNBA, FactGroups: []DataGroupKind{GroupIdentity},
		OfficialAuthority: true, IndependenceKey: "nba",
		AllowWebsite: true, EvidenceURL: "https://www.nba.com/teams",
		ReviewedBy: "repository-curation", ReviewedAt: time.Now().UTC(), Active: true,
	}
}

func TestValidateUsagePolicy(t *testing.T) {
	if err := ValidateUsagePolicy(validFactPolicy()); err != nil {
		t.Fatal(err)
	}
	media := validFactPolicy()
	media.Kind, media.League, media.FactGroups = CapabilityMedia, "", nil
	media.MediaKinds = []MediaKind{MediaLogo}
	if err := ValidateUsagePolicy(media); err != nil {
		t.Fatal(err)
	}
	media.MediaKinds = []MediaKind{MediaLogo, MediaLogo}
	if err := ValidateUsagePolicy(media); !errors.Is(err, ErrInvalidUsagePolicy) {
		t.Fatalf("duplicate media kind error = %v", err)
	}
	policy := validFactPolicy()
	policy.EvidenceURL = "http://example.com/policy"
	if err := ValidateUsagePolicy(policy); !errors.Is(err, ErrInvalidUsagePolicy) {
		t.Fatalf("unsafe evidence URL error = %v", err)
	}
}
