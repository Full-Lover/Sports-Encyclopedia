package atlasregistry

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCompileRegistryPreviewAndFailureRetention(t *testing.T) {
	at := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	identity := validIdentityBatch()
	identity.FetchedAt = at.Add(-time.Hour)
	identity.SeasonEvidence = &OfficialSeasonEvidence{
		League: LeagueNBA, Season: identity.Season,
		EffectiveAt: at.AddDate(-1, 0, 0), RosterPublishedAt: at.AddDate(-1, 0, 1),
		SourceURL: "https://www.nba.com/teams",
		Groups: []OfficialGroupEvidence{
			{Name: "Eastern Conference", Divisions: []string{"Atlantic Division"}},
			{Name: "Western Conference", Divisions: []string{"Pacific Division"}},
		},
	}
	policy := validFactPolicy()
	policy.FactGroups = []DataGroupKind{GroupIdentity, GroupVenue, GroupLeader, GroupRoster}
	venue := VenueBatch{FactMetadata: identity.FactMetadata, Venues: []VenueFact{{
		TeamID: "nba-boston-celtics", VenueID: "td-garden", OfficialName: "TD Garden",
		City: "Boston", Region: "Massachusetts", CountryCode: "US",
		Latitude: 42.366303, Longitude: -71.062228, IsPrimary: true,
		RegularGameCapacity: 19156,
	}}}
	venue.ContentHash = strings.Repeat("b", 64)
	leader := LeaderBatch{FactMetadata: identity.FactMetadata, Leaders: []LeaderFact{{
		TeamID: "nba-boston-celtics", PersonID: "coach-1", OfficialName: "Test Coach",
		Role: RoleHeadCoach,
	}}}
	leader.ContentHash = strings.Repeat("c", 64)
	three, one := "3", "1"
	roster := RosterBatch{FactMetadata: identity.FactMetadata, Rosters: []TeamRosterFact{{
		TeamID: "nba-boston-celtics", Entries: []RosterEntryFact{
			{PersonID: "player-3", OfficialName: "C", Number: &three, Position: "Guard", Status: "Active"},
			{PersonID: "player-1", OfficialName: "A", Number: &one, Position: "Guard", Status: "Active"},
		},
	}}}
	roster.ContentHash = strings.Repeat("d", 64)
	facts := []stagedFact{{identity, policy}, {venue, policy}, {leader, policy}, {roster, policy}}
	request := CompileRequest{RunID: "run-1", Profile: ProfilePreview, RequestedAt: at}
	content, err := compileRegistryContent(request, CompiledRegistryContent{}, facts, nil)
	if err != nil || len(content.Teams) != 1 {
		t.Fatalf("preview content = %#v, %v", content, err)
	}
	team := content.Teams[0]
	if team.Slug != "boston-celtics" || team.Venue.Value == nil || team.Leader.Value == nil ||
		team.Roster.Value == nil || team.Roster.Value.Entries[0].PersonID != "player-1" {
		t.Fatalf("compiled team = %#v", team)
	}
	if _, err := compileRegistryContent(CompileRequest{RunID: "run-v1", Profile: ProfileV1,
		RequestedAt: at}, CompiledRegistryContent{}, facts, nil); !errors.Is(err, ErrCompilationRejected) {
		t.Fatalf("incomplete four-league V1 coverage = %v", err)
	}
	failure := FailedGroup{League: LeagueNBA, TeamID: team.TeamID, Group: GroupVenue,
		FailedAt: at.Add(time.Hour), PossiblyStale: true}
	retained, err := compileRegistryContent(CompileRequest{RunID: "run-2", Profile: ProfilePreview,
		RequestedAt: failure.FailedAt, FailedGroups: []FailedGroup{failure}}, content, nil, nil)
	if err != nil || retained.Teams[0].Venue.State != StateRetainedAfterFailure ||
		retained.Teams[0].Venue.Value == nil || !retained.Teams[0].Venue.PossiblyStale {
		t.Fatalf("retained venue = %#v, %v", retained, err)
	}
}

func TestCompleteOfficialIdentityListControlsCurrentMembership(t *testing.T) {
	base := validIdentityBatch()
	base.Season = "2025-26"
	complete := base
	partial := base
	partial.CompletePagination = false
	partial.Teams = []TeamIdentityFact{{TeamID: "nba-old-team", OfficialName: "Old Team"}}
	policy := validFactPolicy()
	prior := map[string]CompiledTeam{
		"nba-boston-celtics": {TeamID: "nba-boston-celtics", League: LeagueNBA, Season: base.Season},
		"nba-old-team":       {TeamID: "nba-old-team", League: LeagueNBA, Season: base.Season},
	}
	ids, err := selectCurrentTeams(ProfilePreview, prior,
		[]stagedFact{{complete, policy}, {partial, policy}},
		map[LeagueCode]LeagueSeason{LeagueNBA: {League: LeagueNBA, Season: base.Season}})
	if err != nil || len(ids) != 1 || ids[0] != "nba-boston-celtics" {
		t.Fatalf("complete membership with partial older observation = %#v, %v", ids, err)
	}
}

func TestCompileV1WithFourLeagueCoverage(t *testing.T) {
	at := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	var observations []stagedFact
	for _, sample := range []struct {
		league LeagueCode
		group  string
		other  string
	}{
		{LeagueNBA, "Eastern Conference", "Western Conference"},
		{LeagueNFL, "AFC", "NFC"},
		{LeagueMLB, "American League", "National League"},
		{LeagueNHL, "Eastern Conference", "Western Conference"},
	} {
		identity := validIdentityBatch()
		identity.League = sample.league
		identity.SourceID = "official-" + string(sample.league)
		identity.Teams[0].TeamID = "team-" + string(sample.league)
		identity.Teams[0].OfficialName = "Example " + string(sample.league)
		identity.Teams[0].OfficialGroup = sample.group
		identity.Teams[0].Division = "Example Division"
		identity.SeasonEvidence = &OfficialSeasonEvidence{League: sample.league,
			Season: identity.Season, EffectiveAt: at.AddDate(-1, 0, 0),
			RosterPublishedAt: at.AddDate(-1, 0, 1), SourceURL: "https://example.org/season",
			Groups: []OfficialGroupEvidence{{Name: sample.group, Divisions: []string{"Example Division"}},
				{Name: sample.other, Divisions: []string{"Other Division"}}}}
		venue := VenueBatch{FactMetadata: identity.FactMetadata,
			Venues: []VenueFact{{TeamID: identity.Teams[0].TeamID,
				VenueID: "venue-" + string(sample.league), OfficialName: "Example Arena",
				City: "Example City", Region: "Example State", CountryCode: "US",
				Latitude: 40, Longitude: -75, IsPrimary: true}}}
		policy := validFactPolicy()
		policy.League = sample.league
		policy.SourceID = identity.SourceID
		policy.FactGroups = []DataGroupKind{GroupIdentity, GroupVenue}
		observations = append(observations, stagedFact{identity, policy}, stagedFact{venue, policy})
	}
	compiled, err := compileRegistryContent(CompileRequest{RunID: "v1-four-leagues",
		Profile: ProfileV1, RequestedAt: at}, CompiledRegistryContent{}, observations, nil)
	if err != nil || len(compiled.Teams) != 4 {
		t.Fatalf("four-league V1 content = %#v, %v", compiled, err)
	}
}
