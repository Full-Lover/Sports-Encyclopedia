package atlasregistry

import (
	"context"
	"database/sql"
	"errors"
)

// A baseline is an immutable historical selection, not a permanent grant to
// redistribute its facts. A fresh compilation conservatively drops groups
// whose current source capabilities no longer support their prior decision.
func filterBaselineFacts(ctx context.Context, tx *sql.Tx, baseline *CompiledRegistryContent) error {
	retainedEvidence := baseline.SeasonEvidence[:0]
	for _, item := range baseline.SeasonEvidence {
		if item.AuthoritySourceID == "" || item.AuthorityCapabilityKey == "" {
			continue
		}
		policy, err := loadUsagePolicy(ctx, tx, item.AuthoritySourceID, item.AuthorityCapabilityKey)
		if errors.Is(err, ErrPolicyNotAllowed) {
			continue
		}
		if err != nil {
			return err
		}
		if !policy.Active || !policy.AllowWebsite || !policy.OfficialAuthority ||
			policy.Kind != CapabilityFacts || policy.League != item.League {
			continue
		}
		allowed := false
		for _, group := range policy.FactGroups {
			if group == GroupIdentity {
				allowed = true
				break
			}
		}
		if allowed {
			retainedEvidence = append(retainedEvidence, item)
		}
	}
	baseline.SeasonEvidence = retainedEvidence
	for i := range baseline.Teams {
		team := &baseline.Teams[i]
		allowed, err := baselineGroupAllowed(ctx, tx, team.League, GroupIdentity,
			team.Identity.Sources, team.Identity.Verification)
		if err != nil {
			return err
		}
		if !allowed {
			team.Identity = CompiledGroup[TeamIdentityFact]{State: StateUnavailable, UnavailableReason: UnavailableMissing}
		}
		allowed, err = baselineGroupAllowed(ctx, tx, team.League, GroupVenue,
			team.Venue.Sources, team.Venue.Verification)
		if err != nil {
			return err
		}
		if !allowed {
			team.Venue = CompiledGroup[VenueFact]{State: StateUnavailable, UnavailableReason: UnavailableMissing}
		}
		allowed, err = baselineGroupAllowed(ctx, tx, team.League, GroupLeader,
			team.Leader.Sources, team.Leader.Verification)
		if err != nil {
			return err
		}
		if !allowed {
			team.Leader = CompiledGroup[LeaderFact]{State: StateUnavailable, UnavailableReason: UnavailableMissing}
		}
		allowed, err = baselineGroupAllowed(ctx, tx, team.League, GroupRoster,
			team.Roster.Sources, team.Roster.Verification)
		if err != nil {
			return err
		}
		if !allowed {
			team.Roster = CompiledGroup[TeamRosterFact]{State: StateUnavailable, UnavailableReason: UnavailableMissing}
		}
	}
	return nil
}

func baselineGroupAllowed(ctx context.Context, tx *sql.Tx, league LeagueCode, group DataGroupKind,
	sources []SourceCitation, verification *Verification) (bool, error) {
	if len(sources) == 0 {
		return false, nil
	}
	independent := make(map[string]struct{}, len(sources))
	official := false
	for _, citation := range sources {
		policy, err := loadUsagePolicy(ctx, tx, citation.SourceID, citation.CapabilityKey)
		if errors.Is(err, ErrPolicyNotAllowed) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !policy.Active || !policy.AllowWebsite || policy.Kind != CapabilityFacts || policy.League != league {
			return false, nil
		}
		allowed := false
		for _, permitted := range policy.FactGroups {
			if permitted == group {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, nil
		}
		independent[policy.IndependenceKey] = struct{}{}
		official = official || policy.OfficialAuthority
	}
	if verification != nil {
		switch verification.Rule {
		case RuleOfficial:
			if !official {
				return false, nil
			}
		case RuleTwoSources:
			if len(independent) < 2 {
				return false, nil
			}
		default:
			return false, nil
		}
	}
	return true, nil
}
