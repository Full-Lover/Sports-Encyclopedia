package atlasregistry

import (
	"errors"
	"math"
	"testing"
)

func TestValidateVenueBatch(t *testing.T) {
	batch := VenueBatch{FactMetadata: validIdentityBatch().FactMetadata, Venues: []VenueFact{{
		TeamID: "nba-boston-celtics", VenueID: "td-garden", OfficialName: "TD Garden",
		Latitude: 42.366303, Longitude: -71.062228, IsPrimary: true,
		RegularGameCapacity: 19156, OpenedYear: 1995,
	}}}
	if err := ValidateVenueBatch("run-1", batch); err != nil {
		t.Fatal(err)
	}
	batch.Venues[0].Latitude = math.NaN()
	if err := ValidateVenueBatch("run-1", batch); !errors.Is(err, ErrInvalidVenueBatch) {
		t.Fatalf("NaN coordinate error = %v", err)
	}
}

func TestValidateLeaderBatch(t *testing.T) {
	batch := LeaderBatch{FactMetadata: validIdentityBatch().FactMetadata, Leaders: []LeaderFact{{
		TeamID: "nba-boston-celtics", PersonID: "coach-1", OfficialName: "Test Coach", Role: RoleHeadCoach,
	}}}
	if err := ValidateLeaderBatch("run-1", batch); err != nil {
		t.Fatal(err)
	}
	batch.Leaders[0].Role = RoleManager
	if err := ValidateLeaderBatch("run-1", batch); err != nil {
		t.Fatalf("role shape remains valid until league policy checks it: %v", err)
	}
	batch.Leaders = append(batch.Leaders, batch.Leaders[0])
	if err := ValidateLeaderBatch("run-1", batch); !errors.Is(err, ErrInvalidLeaderBatch) {
		t.Fatalf("duplicate team error = %v", err)
	}
}
