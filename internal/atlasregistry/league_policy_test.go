package atlasregistry

import (
	"errors"
	"testing"
	"time"
)

func leagueEvidence(league LeagueCode, season, first, second string, effective, released time.Time) OfficialSeasonEvidence {
	return OfficialSeasonEvidence{League: league, Season: season, EffectiveAt: effective,
		RosterPublishedAt: released, SourceURL: "https://example.org/official-season",
		Groups: []OfficialGroupEvidence{
			{Name: first, Divisions: []string{first + " East"}},
			{Name: second, Divisions: []string{second + " West"}},
		}}
}

func TestFourLeaguePolicies(t *testing.T) {
	from := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		league LeagueCode
		first  string
		second string
		role   LeaderRole
	}{
		{LeagueNBA, "Eastern Conference", "Western Conference", RoleHeadCoach},
		{LeagueNFL, "AFC", "NFC", RoleHeadCoach},
		{LeagueMLB, "American League", "National League", RoleManager},
		{LeagueNHL, "Eastern Conference", "Western Conference", RoleHeadCoach},
	} {
		t.Run(string(test.league), func(t *testing.T) {
			policy, err := PolicyForLeague(test.league)
			if err != nil || policy.LeaderRole() != test.role {
				t.Fatalf("league policy = %T, %v", policy, err)
			}
			evidence := leagueEvidence(test.league, "2025-26", test.first, test.second, from, from)
			if err := policy.ValidateAlignment(AlignmentDraft{Group: test.first, Division: test.first + " East"}, evidence); err != nil {
				t.Fatal(err)
			}
			if err := policy.ValidateAlignment(AlignmentDraft{Group: test.first, Division: test.second + " West"}, evidence); !errors.Is(err, ErrLeagueRule) {
				t.Fatalf("cross-group division error = %v", err)
			}
		})
	}
}

func TestSeasonNeedsPublishedRosterEvidence(t *testing.T) {
	oldStart := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	newStart := oldStart.AddDate(1, 0, 0)
	old := leagueEvidence(LeagueNBA, "2024-25", "Eastern Conference", "Western Conference", oldStart, oldStart)
	newSeason := leagueEvidence(LeagueNBA, "2025-26", "Eastern Conference", "Western Conference", newStart, time.Time{})
	policy, _ := PolicyForLeague(LeagueNBA)
	selected, err := policy.SelectPublishedSeason(newStart.Add(time.Hour), []OfficialSeasonEvidence{old, newSeason})
	if err != nil || selected.Season != old.Season || !selected.Offseason {
		t.Fatalf("unreleased new season = %#v, %v", selected, err)
	}
	newSeason.RosterPublishedAt = newStart.Add(2 * time.Hour)
	selected, err = policy.SelectPublishedSeason(newStart.Add(3*time.Hour), []OfficialSeasonEvidence{old, newSeason})
	if err != nil || selected.Season != newSeason.Season || selected.Offseason {
		t.Fatalf("released new season = %#v, %v", selected, err)
	}
}

func TestRosterPresentationKeepsOfficialStatusAndSortsNumbers(t *testing.T) {
	policy, _ := PolicyForLeague(LeagueNHL)
	three, one := "3", "1"
	groups := policy.RosterPresentation([]RosterEntryFact{
		{PersonID: "p3", OfficialName: "C", Number: &three, Status: "Active", StatusOrder: 0},
		{PersonID: "p1", OfficialName: "A", Number: &one, Status: "Active", StatusOrder: 0},
		{PersonID: "p2", OfficialName: "B", Status: "Injured Reserve", StatusOrder: 1},
	})
	if len(groups) != 2 || groups[0].Status != "Active" || groups[0].Entries[0].PersonID != "p1" ||
		groups[1].Status != "Injured Reserve" {
		t.Fatalf("roster groups = %#v", groups)
	}
}
