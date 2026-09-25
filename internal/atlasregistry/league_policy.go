package atlasregistry

import (
	"errors"
	"sort"
	"strconv"
	"time"
)

type OfficialGroupEvidence struct {
	Name      string
	Divisions []string
}

type OfficialSeasonEvidence struct {
	League                 LeagueCode
	Season                 string
	EffectiveAt            time.Time
	RosterPublishedAt      time.Time
	SourceURL              string
	AuthoritySourceID      string
	AuthorityCapabilityKey string
	Groups                 []OfficialGroupEvidence
}

type LeagueSeason struct {
	League    LeagueCode
	Season    string
	Offseason bool
}

type AlignmentDraft struct {
	Group    string
	Division string
}

type RosterGroup struct {
	Status  string
	Entries []RosterEntryFact
}

var ErrLeagueRule = errors.New("league rule rejected")

type LeaguePolicy interface {
	League() LeagueCode
	SelectPublishedSeason(time.Time, []OfficialSeasonEvidence) (LeagueSeason, error)
	ValidateAlignment(AlignmentDraft, OfficialSeasonEvidence) error
	RosterPresentation([]RosterEntryFact) []RosterGroup
	LeaderRole() LeaderRole
}

func PolicyForLeague(league LeagueCode) (LeaguePolicy, error) {
	switch league {
	case LeagueNBA:
		return nbaPolicy{}, nil
	case LeagueNFL:
		return nflPolicy{}, nil
	case LeagueMLB:
		return mlbPolicy{}, nil
	case LeagueNHL:
		return nhlPolicy{}, nil
	default:
		return nil, ErrLeagueRule
	}
}

type nbaPolicy struct{}
type nflPolicy struct{}
type mlbPolicy struct{}
type nhlPolicy struct{}

func (nbaPolicy) League() LeagueCode { return LeagueNBA }
func (nflPolicy) League() LeagueCode { return LeagueNFL }
func (mlbPolicy) League() LeagueCode { return LeagueMLB }
func (nhlPolicy) League() LeagueCode { return LeagueNHL }

func (nbaPolicy) LeaderRole() LeaderRole { return RoleHeadCoach }
func (nflPolicy) LeaderRole() LeaderRole { return RoleHeadCoach }
func (mlbPolicy) LeaderRole() LeaderRole { return RoleManager }
func (nhlPolicy) LeaderRole() LeaderRole { return RoleHeadCoach }

func (policy nbaPolicy) SelectPublishedSeason(at time.Time, evidence []OfficialSeasonEvidence) (LeagueSeason, error) {
	return selectPublishedSeason(policy.League(), at, evidence)
}
func (policy nflPolicy) SelectPublishedSeason(at time.Time, evidence []OfficialSeasonEvidence) (LeagueSeason, error) {
	return selectPublishedSeason(policy.League(), at, evidence)
}
func (policy mlbPolicy) SelectPublishedSeason(at time.Time, evidence []OfficialSeasonEvidence) (LeagueSeason, error) {
	return selectPublishedSeason(policy.League(), at, evidence)
}
func (policy nhlPolicy) SelectPublishedSeason(at time.Time, evidence []OfficialSeasonEvidence) (LeagueSeason, error) {
	return selectPublishedSeason(policy.League(), at, evidence)
}

func (policy nbaPolicy) ValidateAlignment(draft AlignmentDraft, evidence OfficialSeasonEvidence) error {
	return validateAlignment(policy.League(), draft, evidence, "Eastern Conference", "Western Conference")
}
func (policy nflPolicy) ValidateAlignment(draft AlignmentDraft, evidence OfficialSeasonEvidence) error {
	return validateAlignment(policy.League(), draft, evidence, "AFC", "NFC")
}
func (policy mlbPolicy) ValidateAlignment(draft AlignmentDraft, evidence OfficialSeasonEvidence) error {
	return validateAlignment(policy.League(), draft, evidence, "American League", "National League")
}
func (policy nhlPolicy) ValidateAlignment(draft AlignmentDraft, evidence OfficialSeasonEvidence) error {
	return validateAlignment(policy.League(), draft, evidence, "Eastern Conference", "Western Conference")
}

func (nbaPolicy) RosterPresentation(entries []RosterEntryFact) []RosterGroup {
	return rosterGroups(entries)
}
func (nflPolicy) RosterPresentation(entries []RosterEntryFact) []RosterGroup {
	return rosterGroups(entries)
}
func (mlbPolicy) RosterPresentation(entries []RosterEntryFact) []RosterGroup {
	return rosterGroups(entries)
}
func (nhlPolicy) RosterPresentation(entries []RosterEntryFact) []RosterGroup {
	return rosterGroups(entries)
}

func ValidateOfficialSeasonEvidence(evidence OfficialSeasonEvidence) error {
	if !validText(evidence.Season, 40) || !validHTTPSURL(evidence.SourceURL) ||
		evidence.EffectiveAt.IsZero() || len(evidence.Groups) != 2 {
		return ErrLeagueRule
	}
	firstGroup, secondGroup, err := expectedGroups(evidence.League)
	if err != nil {
		return ErrLeagueRule
	}
	seen := make(map[string]struct{}, 2)
	for _, group := range evidence.Groups {
		if !validText(group.Name, 80) || len(group.Divisions) == 0 || len(group.Divisions) > 8 {
			return ErrLeagueRule
		}
		if _, duplicate := seen[group.Name]; duplicate {
			return ErrLeagueRule
		}
		seen[group.Name] = struct{}{}
		divisions := make(map[string]struct{}, len(group.Divisions))
		for _, division := range group.Divisions {
			if !validText(division, 80) {
				return ErrLeagueRule
			}
			if _, duplicate := divisions[division]; duplicate {
				return ErrLeagueRule
			}
			divisions[division] = struct{}{}
		}
	}
	if _, found := seen[firstGroup]; !found {
		return ErrLeagueRule
	}
	if _, found := seen[secondGroup]; !found {
		return ErrLeagueRule
	}
	return nil
}

func selectPublishedSeason(league LeagueCode, at time.Time, evidence []OfficialSeasonEvidence) (LeagueSeason, error) {
	var selected *OfficialSeasonEvidence
	var newestUnreleased time.Time
	for i := range evidence {
		item := &evidence[i]
		if item.League != league || ValidateOfficialSeasonEvidence(*item) != nil {
			return LeagueSeason{}, ErrLeagueRule
		}
		if item.EffectiveAt.After(at) {
			continue
		}
		if item.RosterPublishedAt.IsZero() || item.RosterPublishedAt.After(at) {
			if item.EffectiveAt.After(newestUnreleased) {
				newestUnreleased = item.EffectiveAt
			}
			continue
		}
		if selected == nil || item.EffectiveAt.After(selected.EffectiveAt) {
			selected = item
		}
	}
	if selected == nil {
		return LeagueSeason{}, ErrLeagueRule
	}
	return LeagueSeason{League: league, Season: selected.Season,
		Offseason: newestUnreleased.After(selected.EffectiveAt)}, nil
}

func expectedGroups(league LeagueCode) (string, string, error) {
	switch league {
	case LeagueNBA, LeagueNHL:
		return "Eastern Conference", "Western Conference", nil
	case LeagueNFL:
		return "AFC", "NFC", nil
	case LeagueMLB:
		return "American League", "National League", nil
	default:
		return "", "", ErrLeagueRule
	}
}

func validateAlignment(league LeagueCode, draft AlignmentDraft, evidence OfficialSeasonEvidence, allowed ...string) error {
	if evidence.League != league || ValidateOfficialSeasonEvidence(evidence) != nil {
		return ErrLeagueRule
	}
	validGroup := false
	for _, name := range allowed {
		if draft.Group == name {
			validGroup = true
		}
	}
	if !validGroup {
		return ErrLeagueRule
	}
	for _, group := range evidence.Groups {
		if group.Name != draft.Group {
			continue
		}
		for _, division := range group.Divisions {
			if division == draft.Division {
				return nil
			}
		}
	}
	return ErrLeagueRule
}

func rosterGroups(entries []RosterEntryFact) []RosterGroup {
	byStatus := make(map[string][]RosterEntryFact)
	order := make(map[string]int)
	for _, entry := range entries {
		byStatus[entry.Status] = append(byStatus[entry.Status], entry)
		if previous, exists := order[entry.Status]; !exists || entry.StatusOrder < previous {
			order[entry.Status] = entry.StatusOrder
		}
	}
	statuses := make([]string, 0, len(byStatus))
	for status := range byStatus {
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool {
		if order[statuses[i]] != order[statuses[j]] {
			return order[statuses[i]] < order[statuses[j]]
		}
		return statuses[i] < statuses[j]
	})
	groups := make([]RosterGroup, 0, len(statuses))
	for _, status := range statuses {
		members := append([]RosterEntryFact(nil), byStatus[status]...)
		sort.Slice(members, func(i, j int) bool {
			left, leftOK := jerseyNumber(members[i].Number)
			right, rightOK := jerseyNumber(members[j].Number)
			if leftOK != rightOK {
				return leftOK
			}
			if leftOK && left != right {
				return left < right
			}
			if members[i].OfficialName != members[j].OfficialName {
				return members[i].OfficialName < members[j].OfficialName
			}
			return members[i].PersonID < members[j].PersonID
		})
		groups = append(groups, RosterGroup{Status: status, Entries: members})
	}
	return groups
}

func jerseyNumber(value *string) (int, bool) {
	if value == nil {
		return 0, false
	}
	number, err := strconv.Atoi(*value)
	return number, err == nil && number >= 0
}
