package atlasregistry

import (
	"errors"
	"fmt"
)

type RosterEntryFact struct {
	PersonID     string
	OfficialName string
	Number       *string
	Position     string
	Status       string
	StatusOrder  int
}

type TeamRosterFact struct {
	TeamID  string
	Entries []RosterEntryFact
}

type RosterBatch struct {
	FactMetadata
	Rosters []TeamRosterFact
}

var ErrInvalidRosterBatch = errors.New("invalid roster batch")

func (batch RosterBatch) factGroup() DataGroupKind { return GroupRoster }
func (batch RosterBatch) metadata() FactMetadata   { return batch.FactMetadata }
func (batch RosterBatch) validate(runID string) error {
	return ValidateRosterBatch(runID, batch)
}

func ValidateRosterBatch(runID string, batch RosterBatch) error {
	if validateFactMetadata(runID, batch.FactMetadata) != nil || len(batch.Rosters) == 0 || len(batch.Rosters) > 100 {
		return ErrInvalidRosterBatch
	}
	teams := make(map[string]struct{}, len(batch.Rosters))
	for _, roster := range batch.Rosters {
		if !validToken(roster.TeamID, 64) || len(roster.Entries) > 150 {
			return fmt.Errorf("%w: team %q", ErrInvalidRosterBatch, roster.TeamID)
		}
		if _, duplicate := teams[roster.TeamID]; duplicate {
			return fmt.Errorf("%w: duplicate team %q", ErrInvalidRosterBatch, roster.TeamID)
		}
		teams[roster.TeamID] = struct{}{}
		people := make(map[string]struct{}, len(roster.Entries))
		for _, entry := range roster.Entries {
			if !validToken(entry.PersonID, 64) || !validText(entry.OfficialName, 160) ||
				!validText(entry.Position, 80) || !validText(entry.Status, 80) ||
				entry.StatusOrder < 0 || entry.StatusOrder > 1000 ||
				(entry.Number != nil && !validText(*entry.Number, 12)) {
				return fmt.Errorf("%w: person %q", ErrInvalidRosterBatch, entry.PersonID)
			}
			if _, duplicate := people[entry.PersonID]; duplicate {
				return fmt.Errorf("%w: duplicate person %q", ErrInvalidRosterBatch, entry.PersonID)
			}
			people[entry.PersonID] = struct{}{}
		}
	}
	return nil
}
