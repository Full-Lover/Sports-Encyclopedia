package atlasregistry

import (
	"encoding/json"
	"sort"
	"time"
)

type factCandidate[T any] struct {
	Value    T
	Policy   UsagePolicy
	Citation SourceCitation
}

func resolveGroup[T any](candidates []factCandidate[T], previous CompiledGroup[T],
	failed *FailedGroup, at time.Time) (CompiledGroup[T], error) {
	latest := make(map[[2]string]factCandidate[T], len(candidates))
	for _, candidate := range candidates {
		key := [2]string{candidate.Citation.SourceID, candidate.Citation.CapabilityKey}
		prior, found := latest[key]
		if found && candidate.Citation.FetchedAt.Equal(prior.Citation.FetchedAt) {
			left, leftErr := json.Marshal(prior.Value)
			right, rightErr := json.Marshal(candidate.Value)
			if leftErr != nil || rightErr != nil || string(left) != string(right) {
				return CompiledGroup[T]{}, ErrRevisionConflict
			}
		}
		if !found || candidate.Citation.FetchedAt.After(prior.Citation.FetchedAt) {
			latest[key] = candidate
		}
	}
	chosen := make([]factCandidate[T], 0, len(latest))
	for _, candidate := range latest {
		chosen = append(chosen, candidate)
	}
	sort.Slice(chosen, func(i, j int) bool {
		if chosen[i].Citation.SourceID != chosen[j].Citation.SourceID {
			return chosen[i].Citation.SourceID < chosen[j].Citation.SourceID
		}
		return chosen[i].Citation.CapabilityKey < chosen[j].Citation.CapabilityKey
	})
	if len(chosen) == 0 {
		if failed == nil {
			if previous.State == "" {
				return CompiledGroup[T]{State: StateUnavailable}, nil
			}
			return previous, nil
		}
		retainedValue := previous.Value
		if retainedValue == nil {
			retainedValue = previous.LastVerifiedValue
		}
		if retainedValue == nil {
			return CompiledGroup[T]{State: StateUnavailable}, nil
		}
		lastSuccess := previous.SyncedAt
		if lastSuccess.IsZero() {
			lastSuccess = previous.LastSuccessfulSyncAt
		}
		return CompiledGroup[T]{State: StateRetainedAfterFailure, Value: retainedValue,
			LastSuccessfulSyncAt: lastSuccess, FailedAt: failed.FailedAt,
			PossiblyStale: failed.PossiblyStale, Verification: previous.Verification,
			Sources: previous.Sources}, nil
	}
	official := make([]factCandidate[T], 0, len(chosen))
	for _, candidate := range chosen {
		if candidate.Policy.OfficialAuthority {
			official = append(official, candidate)
		}
	}
	considered := chosen
	rule := RuleTwoSources
	if len(official) != 0 {
		considered = official
		rule = RuleOfficial
	}
	firstValue, err := json.Marshal(considered[0].Value)
	if err != nil {
		return CompiledGroup[T]{}, err
	}
	conflict := false
	independent := make(map[string]struct{}, len(considered))
	for _, candidate := range considered {
		value, err := json.Marshal(candidate.Value)
		if err != nil {
			return CompiledGroup[T]{}, err
		}
		if string(value) != string(firstValue) {
			conflict = true
		}
		independent[candidate.Policy.IndependenceKey] = struct{}{}
	}
	sources := make([]SourceCitation, 0, len(considered))
	var syncedAt time.Time
	for _, candidate := range considered {
		sources = append(sources, candidate.Citation)
		if candidate.Citation.FetchedAt.After(syncedAt) {
			syncedAt = candidate.Citation.FetchedAt
		}
	}
	if conflict {
		result := CompiledGroup[T]{State: StateConflict, ConflictDetectedAt: at, Sources: sources}
		if previous.Verification != nil {
			result.LastVerifiedValue = previous.Value
		}
		if result.LastVerifiedValue == nil {
			result.LastVerifiedValue = previous.LastVerifiedValue
		}
		return result, nil
	}
	value := considered[0].Value
	result := CompiledGroup[T]{State: StateCurrent, Value: &value, SyncedAt: syncedAt, Sources: sources}
	if rule == RuleOfficial || len(independent) >= 2 {
		result.Verification = &Verification{VerifiedAt: at, Rule: rule}
	}
	return result, nil
}
