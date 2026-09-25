package atlasregistry

import (
	"errors"
	"testing"
)

func TestValidateRosterBatch(t *testing.T) {
	zero := "0"
	batch := RosterBatch{FactMetadata: validIdentityBatch().FactMetadata, Rosters: []TeamRosterFact{{
		TeamID: "nba-boston-celtics", Entries: []RosterEntryFact{{
			PersonID: "player-1", OfficialName: "Example Player", Number: &zero,
			Position: "Guard", Status: "Active", StatusOrder: 1,
		}},
	}}}
	if err := ValidateRosterBatch("run-1", batch); err != nil {
		t.Fatal(err)
	}
	batch.Rosters[0].Entries = append(batch.Rosters[0].Entries, batch.Rosters[0].Entries[0])
	if err := ValidateRosterBatch("run-1", batch); !errors.Is(err, ErrInvalidRosterBatch) {
		t.Fatalf("duplicate player error = %v", err)
	}
}
