package atlasregistry

import (
	"errors"
	"testing"
	"time"
)

func TestResolveGroupOfficialTwoSourceConflictAndFailure(t *testing.T) {
	at := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	value := TeamIdentityFact{TeamID: "team-1", OfficialName: "Team One"}
	makeCandidate := func(source, organization string, official bool, item TeamIdentityFact) factCandidate[TeamIdentityFact] {
		return factCandidate[TeamIdentityFact]{Value: item,
			Policy: UsagePolicy{OfficialAuthority: official, IndependenceKey: organization},
			Citation: SourceCitation{SourceID: source, CapabilityKey: "identity",
				SourceURL: "https://example.org/team", FetchedAt: at.Add(-time.Hour)}}
	}
	first := makeCandidate("a", "org-a", false, value)
	result, err := resolveGroup([]factCandidate[TeamIdentityFact]{first}, CompiledGroup[TeamIdentityFact]{}, nil, at)
	if err != nil || result.State != StateCurrent || result.Verification != nil {
		t.Fatalf("single community source = %#v, %v", result, err)
	}
	second := makeCandidate("b", "org-b", false, value)
	result, err = resolveGroup([]factCandidate[TeamIdentityFact]{first, second}, result, nil, at)
	if err != nil || result.Verification == nil || result.Verification.Rule != RuleTwoSources {
		t.Fatalf("independent agreement = %#v, %v", result, err)
	}
	disagrees := makeCandidate("c", "org-c", false, TeamIdentityFact{TeamID: "team-1", OfficialName: "Other"})
	conflict, err := resolveGroup([]factCandidate[TeamIdentityFact]{first, disagrees}, result, nil, at)
	if err != nil || conflict.State != StateConflict || conflict.Verification != nil ||
		conflict.LastVerifiedValue == nil || conflict.LastVerifiedValue.OfficialName != "Team One" {
		t.Fatalf("conflict with old verified value = %#v, %v", conflict, err)
	}
	official := makeCandidate("official", "league", true, value)
	resolved, err := resolveGroup([]factCandidate[TeamIdentityFact]{official, disagrees}, conflict, nil, at)
	if err != nil || resolved.State != StateCurrent || resolved.Verification == nil ||
		resolved.Verification.Rule != RuleOfficial {
		t.Fatalf("official precedence = %#v, %v", resolved, err)
	}
	failed := FailedGroup{FailedAt: at.Add(time.Hour), PossiblyStale: true}
	retained, err := resolveGroup([]factCandidate[TeamIdentityFact]{}, resolved, &failed, failed.FailedAt)
	if err != nil || retained.State != StateRetainedAfterFailure || !retained.PossiblyStale ||
		retained.Value == nil || retained.Value.OfficialName != "Team One" {
		t.Fatalf("failed refresh retention = %#v, %v", retained, err)
	}
	retainedConflict, err := resolveGroup([]factCandidate[TeamIdentityFact]{}, conflict, &failed, failed.FailedAt)
	if err != nil || retainedConflict.Value == nil || retainedConflict.Value.OfficialName != "Team One" {
		t.Fatalf("failure after conflict lost last verified value = %#v, %v", retainedConflict, err)
	}
	duplicateTimestamp := makeCandidate("a", "org-a", false, disagrees.Value)
	if _, err := resolveGroup([]factCandidate[TeamIdentityFact]{first, duplicateTimestamp},
		CompiledGroup[TeamIdentityFact]{}, nil, at); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("same-source timestamp conflict = %v", err)
	}
}

func TestResolveGroupDoesNotRegressToOlderOrNonOfficialObservation(t *testing.T) {
	at := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	old := TeamIdentityFact{TeamID: "team-1", OfficialName: "Old Name"}
	current := TeamIdentityFact{TeamID: "team-1", OfficialName: "Current Name"}
	prior := CompiledGroup[TeamIdentityFact]{State: StateCurrent, Value: &current,
		SyncedAt: at, Verification: &Verification{VerifiedAt: at, Rule: RuleOfficial},
		Sources: []SourceCitation{{SourceID: "official", CapabilityKey: "identity", FetchedAt: at}}}
	candidate := factCandidate[TeamIdentityFact]{Value: old,
		Policy:   UsagePolicy{OfficialAuthority: true, IndependenceKey: "league"},
		Citation: SourceCitation{SourceID: "official", CapabilityKey: "identity", FetchedAt: at.Add(-time.Hour)}}
	result, err := resolveGroup([]factCandidate[TeamIdentityFact]{candidate}, prior, nil, at.Add(time.Hour))
	if err != nil || result.Value == nil || result.Value.OfficialName != current.OfficialName {
		t.Fatalf("older official observation replaced baseline: %#v, %v", result, err)
	}
	candidate.Citation.FetchedAt = at
	if _, err := resolveGroup([]factCandidate[TeamIdentityFact]{candidate}, prior, nil, at.Add(time.Hour)); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("same-source timestamp reused for different value = %v", err)
	}
	candidate.Policy.OfficialAuthority = false
	candidate.Citation.SourceID = "community"
	candidate.Citation.FetchedAt = at.Add(time.Hour)
	result, err = resolveGroup([]factCandidate[TeamIdentityFact]{candidate}, prior, nil, at.Add(2*time.Hour))
	if err != nil || result.Value == nil || result.Value.OfficialName != current.OfficialName ||
		result.Verification == nil || result.Verification.Rule != RuleOfficial {
		t.Fatalf("community observation displaced official baseline: %#v, %v", result, err)
	}
	candidate.Policy.OfficialAuthority = true
	candidate.Citation.SourceID = "official"
	result, err = resolveGroup([]factCandidate[TeamIdentityFact]{candidate}, prior, nil, at.Add(2*time.Hour))
	if err != nil || result.Value == nil || result.Value.OfficialName != old.OfficialName {
		t.Fatalf("newer official correction not accepted: %#v, %v", result, err)
	}
}
