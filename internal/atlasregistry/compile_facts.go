package atlasregistry

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

var ErrCompilationRejected = errors.New("registry compilation rejected")

type stagedFact struct {
	Batch  FactBatch
	Policy UsagePolicy
}

type stagedMedia struct {
	Batch          MediaAssetBatch
	Policy         UsagePolicy
	ApprovedRights *MediaRightsOption
}

type groupedFacts struct {
	identity map[string][]factCandidate[TeamIdentityFact]
	venue    map[string][]factCandidate[VenueFact]
	leader   map[string][]factCandidate[LeaderFact]
	roster   map[string][]factCandidate[TeamRosterFact]
}

func compileRegistryContent(request CompileRequest, baseline CompiledRegistryContent,
	facts []stagedFact, media []stagedMedia) (CompiledRegistryContent, error) {
	evidence, err := mergeSeasonEvidence(baseline.SeasonEvidence, facts)
	if err != nil {
		return CompiledRegistryContent{}, err
	}
	previous := make(map[string]CompiledTeam, len(baseline.Teams))
	for _, team := range baseline.Teams {
		previous[team.TeamID] = team
	}
	selected := make(map[LeagueCode]LeagueSeason)
	for _, league := range []LeagueCode{LeagueNBA, LeagueNFL, LeagueMLB, LeagueNHL} {
		var leagueEvidence []OfficialSeasonEvidence
		for _, item := range evidence {
			if item.League == league {
				leagueEvidence = append(leagueEvidence, item)
			}
		}
		if len(leagueEvidence) == 0 {
			if request.Profile == ProfileV1 {
				return CompiledRegistryContent{}, ErrCompilationRejected
			}
			continue
		}
		policy, _ := PolicyForLeague(league)
		season, err := policy.SelectPublishedSeason(request.RequestedAt, leagueEvidence)
		if err != nil {
			return CompiledRegistryContent{}, err
		}
		selected[league] = season
	}
	inputs := collectGroupedFacts(facts, selected)
	teamIDs, err := selectCurrentTeams(request.Profile, previous, facts, selected)
	if err != nil {
		return CompiledRegistryContent{}, err
	}
	result := CompiledRegistryContent{SeasonEvidence: evidence}
	for _, teamID := range teamIDs {
		prior := previous[teamID]
		league := prior.League
		if league == "" {
			league = identityLeague(facts, teamID, selected)
		}
		season, found := selected[league]
		if !found {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		policy, _ := PolicyForLeague(league)
		team := CompiledTeam{TeamID: teamID, League: league, Season: season.Season, Offseason: season.Offseason}
		team.Identity, err = resolveGroup(inputs.identity[teamID], prior.Identity,
			failedScope(request.FailedGroups, league, teamID, GroupIdentity), request.RequestedAt)
		if err != nil {
			return CompiledRegistryContent{}, err
		}
		identity := displayValue(team.Identity)
		if identity == nil || !validTeamAbbreviation(identity.OfficialAbbreviation) {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		var currentEvidence *OfficialSeasonEvidence
		for i := range evidence {
			if evidence[i].League == league && evidence[i].Season == season.Season {
				currentEvidence = &evidence[i]
				break
			}
		}
		if currentEvidence == nil || policy.ValidateAlignment(AlignmentDraft{
			Group: identity.OfficialGroup, Division: identity.Division}, *currentEvidence) != nil {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		team.Slug = teamSlug(identity.OfficialName)
		if team.Slug == "" {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		aliases := make(map[string]struct{}, len(prior.Aliases)+1)
		for _, name := range prior.Aliases {
			if name != identity.OfficialName {
				aliases[name] = struct{}{}
			}
		}
		if old := displayValue(prior.Identity); old != nil && old.OfficialName != identity.OfficialName {
			aliases[old.OfficialName] = struct{}{}
		}
		for name := range aliases {
			team.Aliases = append(team.Aliases, name)
		}
		sort.Strings(team.Aliases)
		team.Venue, err = resolveGroup(inputs.venue[teamID], prior.Venue,
			failedScope(request.FailedGroups, league, teamID, GroupVenue), request.RequestedAt)
		if err != nil {
			return CompiledRegistryContent{}, err
		}
		oldLeader, oldRoster := prior.Leader, prior.Roster
		if prior.Season != "" && prior.Season != season.Season {
			oldLeader, oldRoster = CompiledGroup[LeaderFact]{}, CompiledGroup[TeamRosterFact]{}
		}
		team.Leader, err = resolveGroup(inputs.leader[teamID], oldLeader,
			failedScope(request.FailedGroups, league, teamID, GroupLeader), request.RequestedAt)
		if err != nil {
			return CompiledRegistryContent{}, err
		}
		if leader := displayValue(team.Leader); leader != nil && leader.Role != policy.LeaderRole() {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		team.Roster, err = resolveGroup(inputs.roster[teamID], oldRoster,
			failedScope(request.FailedGroups, league, teamID, GroupRoster), request.RequestedAt)
		if err != nil {
			return CompiledRegistryContent{}, err
		}
		if roster := team.Roster.Value; roster != nil {
			groups := policy.RosterPresentation(roster.Entries)
			ordered := make([]RosterEntryFact, 0, len(roster.Entries))
			for _, group := range groups {
				ordered = append(ordered, group.Entries...)
			}
			roster.Entries = ordered
		}
		if request.Profile == ProfileV1 && (displayValue(team.Venue) == nil ||
			len(team.Venue.Sources) == 0 || len(team.Identity.Sources) == 0 ||
			identity.OfficialWebsiteURL == "") {
			return CompiledRegistryContent{}, ErrCompilationRejected
		}
		result.Teams = append(result.Teams, team)
	}
	if len(result.Teams) == 0 || !consistentVenueCoordinates(result.Teams) {
		return CompiledRegistryContent{}, ErrCompilationRejected
	}
	attachMedia(&result, media)
	return result, nil
}

func displayValue[T any](group CompiledGroup[T]) *T {
	if group.Value != nil {
		return group.Value
	}
	return group.LastVerifiedValue
}

func failedScope(failures []FailedGroup, league LeagueCode, teamID string, group DataGroupKind) *FailedGroup {
	for i := range failures {
		if failures[i].League == league && failures[i].TeamID == teamID && failures[i].Group == group {
			return &failures[i]
		}
	}
	return nil
}

func mergeSeasonEvidence(previous []OfficialSeasonEvidence, facts []stagedFact) ([]OfficialSeasonEvidence, error) {
	byKey := make(map[[2]string]OfficialSeasonEvidence)
	for _, item := range previous {
		byKey[[2]string{string(item.League), item.Season}] = item
	}
	for _, observation := range facts {
		batch, isIdentity := observation.Batch.(TeamIdentityBatch)
		if !isIdentity || batch.SeasonEvidence == nil || !observation.Policy.Active ||
			!observation.Policy.AllowWebsite || !observation.Policy.OfficialAuthority ||
			!policyAllowsFact(observation.Policy, batch) {
			continue
		}
		item := *batch.SeasonEvidence
		item.AuthoritySourceID = batch.SourceID
		item.AuthorityCapabilityKey = batch.CapabilityKey
		key := [2]string{string(item.League), item.Season}
		if old, exists := byKey[key]; exists {
			old.AuthoritySourceID, old.AuthorityCapabilityKey = "", ""
			comparable := item
			comparable.AuthoritySourceID, comparable.AuthorityCapabilityKey = "", ""
			left, _ := json.Marshal(old)
			right, _ := json.Marshal(comparable)
			if string(left) != string(right) {
				return nil, ErrCompilationRejected
			}
		}
		byKey[key] = item
	}
	result := make([]OfficialSeasonEvidence, 0, len(byKey))
	for _, item := range byKey {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].League != result[j].League {
			return result[i].League < result[j].League
		}
		return result[i].EffectiveAt.Before(result[j].EffectiveAt)
	})
	return result, nil
}

func teamSlug(name string) string {
	var builder strings.Builder
	separator := false
	for _, character := range strings.ToLower(name) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			if separator && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			separator = false
			builder.WriteRune(character)
		} else {
			separator = true
		}
	}
	if builder.Len() > 160 {
		return ""
	}
	return builder.String()
}

func consistentVenueCoordinates(teams []CompiledTeam) bool {
	type location struct {
		latitude, longitude   float64
		city, region, country string
	}
	venues := make(map[string]location)
	for _, team := range teams {
		venue := displayValue(team.Venue)
		if venue == nil {
			continue
		}
		current := location{venue.Latitude, venue.Longitude, venue.City, venue.Region, venue.CountryCode}
		if previous, exists := venues[venue.VenueID]; exists && previous != current {
			return false
		}
		venues[venue.VenueID] = current
	}
	return true
}
