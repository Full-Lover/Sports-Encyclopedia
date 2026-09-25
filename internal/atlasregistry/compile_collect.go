package atlasregistry

import (
	"encoding/json"
	"sort"
)

func collectGroupedFacts(facts []stagedFact, selected map[LeagueCode]LeagueSeason) groupedFacts {
	result := groupedFacts{
		identity: make(map[string][]factCandidate[TeamIdentityFact]),
		venue:    make(map[string][]factCandidate[VenueFact]),
		leader:   make(map[string][]factCandidate[LeaderFact]),
		roster:   make(map[string][]factCandidate[TeamRosterFact]),
	}
	for _, observation := range facts {
		batch, policy := observation.Batch, observation.Policy
		metadata := batch.metadata()
		season, exists := selected[metadata.League]
		if !exists || season.Season != metadata.Season || !policy.AllowWebsite ||
			!policyAllowsFact(policy, batch) {
			continue
		}
		citation := SourceCitation{SourceID: metadata.SourceID, CapabilityKey: metadata.CapabilityKey,
			SourceURL: metadata.SourceURL, FetchedAt: metadata.FetchedAt}
		switch typed := batch.(type) {
		case TeamIdentityBatch:
			for _, item := range typed.Teams {
				result.identity[item.TeamID] = append(result.identity[item.TeamID],
					factCandidate[TeamIdentityFact]{Value: item, Policy: policy, Citation: citation})
			}
		case VenueBatch:
			for _, item := range typed.Venues {
				if item.IsPrimary {
					result.venue[item.TeamID] = append(result.venue[item.TeamID],
						factCandidate[VenueFact]{Value: item, Policy: policy, Citation: citation})
				}
			}
		case LeaderBatch:
			for _, item := range typed.Leaders {
				result.leader[item.TeamID] = append(result.leader[item.TeamID],
					factCandidate[LeaderFact]{Value: item, Policy: policy, Citation: citation})
			}
		case RosterBatch:
			for _, item := range typed.Rosters {
				copy := item
				copy.Entries = append([]RosterEntryFact(nil), item.Entries...)
				sort.Slice(copy.Entries, func(i, j int) bool {
					return copy.Entries[i].PersonID < copy.Entries[j].PersonID
				})
				result.roster[item.TeamID] = append(result.roster[item.TeamID],
					factCandidate[TeamRosterFact]{Value: copy, Policy: policy, Citation: citation})
			}
		}
	}
	return result
}

func selectCurrentTeams(profile PublicationProfile, previous map[string]CompiledTeam,
	facts []stagedFact, selected map[LeagueCode]LeagueSeason) ([]string, error) {
	current := make(map[string]LeagueCode)
	for id, team := range previous {
		if _, exists := selected[team.League]; exists {
			current[id] = team.League
		}
	}
	type completeList struct {
		batch TeamIdentityBatch
	}
	complete := make(map[LeagueCode]completeList)
	completeIDs := make(map[LeagueCode]map[string]struct{})
	for _, observation := range facts {
		batch, valid := observation.Batch.(TeamIdentityBatch)
		if !valid || !observation.Policy.Active || !observation.Policy.AllowWebsite ||
			!policyAllowsFact(observation.Policy, batch) {
			continue
		}
		season, exists := selected[batch.League]
		if !exists || batch.Season != season.Season || !batch.CompletePagination ||
			!observation.Policy.OfficialAuthority {
			continue
		}
		old, found := complete[batch.League]
		if !found || batch.FetchedAt.After(old.batch.FetchedAt) {
			complete[batch.League] = completeList{batch: batch}
		} else if batch.FetchedAt.Equal(old.batch.FetchedAt) {
			left, _ := json.Marshal(old.batch.Teams)
			right, _ := json.Marshal(batch.Teams)
			if string(left) != string(right) {
				return nil, ErrCompilationRejected
			}
		}
	}
	for league, season := range selected {
		if full, exists := complete[league]; exists {
			completeIDs[league] = make(map[string]struct{}, len(full.batch.Teams))
			for id, oldLeague := range current {
				if oldLeague == league {
					delete(current, id)
				}
			}
			for _, team := range full.batch.Teams {
				completeIDs[league][team.TeamID] = struct{}{}
				if oldLeague, duplicate := current[team.TeamID]; duplicate && oldLeague != league {
					return nil, ErrCompilationRejected
				}
				current[team.TeamID] = league
			}
		} else {
			for _, team := range previous {
				if team.League == league && team.Season != season.Season {
					return nil, ErrCompilationRejected
				}
			}
		}
	}
	for _, observation := range facts {
		batch, valid := observation.Batch.(TeamIdentityBatch)
		if !valid || !observation.Policy.AllowWebsite || !policyAllowsFact(observation.Policy, batch) {
			continue
		}
		season, exists := selected[batch.League]
		if !exists || batch.Season != season.Season {
			continue
		}
		if profile == ProfileV1 && !observation.Policy.OfficialAuthority {
			continue
		}
		for _, team := range batch.Teams {
			if members, hasCompleteList := completeIDs[batch.League]; hasCompleteList {
				if _, currentMember := members[team.TeamID]; !currentMember {
					continue
				}
			}
			if other, duplicate := current[team.TeamID]; duplicate && other != batch.League {
				return nil, ErrCompilationRejected
			}
			current[team.TeamID] = batch.League
		}
	}
	if profile == ProfileV1 {
		for _, league := range []LeagueCode{LeagueNBA, LeagueNFL, LeagueMLB, LeagueNHL} {
			if _, exists := complete[league]; !exists {
				baselineCovered := false
				for _, prior := range previous {
					if prior.League == league && prior.Season == selected[league].Season {
						baselineCovered = true
						break
					}
				}
				if !baselineCovered {
					return nil, ErrCompilationRejected
				}
			}
		}
	}
	ids := make([]string, 0, len(current))
	for id := range current {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func identityLeague(facts []stagedFact, teamID string, selected map[LeagueCode]LeagueSeason) LeagueCode {
	for _, observation := range facts {
		batch, valid := observation.Batch.(TeamIdentityBatch)
		if !valid || !observation.Policy.AllowWebsite || !policyAllowsFact(observation.Policy, batch) ||
			selected[batch.League].Season != batch.Season {
			continue
		}
		for _, team := range batch.Teams {
			if team.TeamID == teamID {
				return batch.League
			}
		}
	}
	return ""
}
