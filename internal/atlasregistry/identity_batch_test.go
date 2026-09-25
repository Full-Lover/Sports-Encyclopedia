package atlasregistry

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validIdentityBatch() TeamIdentityBatch {
	return TeamIdentityBatch{
		SourceID: "nba-official", CapabilityKey: "teams.identity", League: LeagueNBA,
		Season: "2025-26", SourceURL: "https://www.nba.com/teams",
		FetchedAt:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		ContentHash: strings.Repeat("a", 64), CompletePagination: true,
		Teams: []TeamIdentityFact{{
			TeamID: "nba-boston-celtics", OfficialName: "Boston Celtics",
			OfficialGroup: "Eastern Conference", Division: "Atlantic Division",
			OfficialWebsiteURL: "https://www.nba.com/celtics/",
		}},
	}
}

func TestValidateTeamIdentityBatch(t *testing.T) {
	if err := ValidateTeamIdentityBatch("run-1", validIdentityBatch()); err != nil {
		t.Fatal(err)
	}
	incomplete := validIdentityBatch()
	incomplete.CompletePagination = false
	if err := ValidateTeamIdentityBatch("run-1", incomplete); err != nil {
		t.Fatalf("incomplete observations must remain representable: %v", err)
	}
	tests := map[string]func(*TeamIdentityBatch){
		"unsupported league": func(batch *TeamIdentityBatch) { batch.League = "MLS" },
		"missing source":     func(batch *TeamIdentityBatch) { batch.SourceID = "" },
		"bad hash":           func(batch *TeamIdentityBatch) { batch.ContentHash = "not-sha256" },
		"unsafe citation":    func(batch *TeamIdentityBatch) { batch.SourceURL = "http://example.com" },
		"unsafe official link": func(batch *TeamIdentityBatch) {
			batch.Teams[0].OfficialWebsiteURL = "javascript:alert(1)"
		},
		"control in name": func(batch *TeamIdentityBatch) { batch.Teams[0].OfficialName = "Celtics\nOther" },
		"duplicate team": func(batch *TeamIdentityBatch) {
			batch.Teams = append(batch.Teams, batch.Teams[0])
		},
		"missing group": func(batch *TeamIdentityBatch) { batch.Teams[0].OfficialGroup = "" },
	}
	for name, alter := range tests {
		t.Run(name, func(t *testing.T) {
			batch := validIdentityBatch()
			alter(&batch)
			if err := ValidateTeamIdentityBatch("run-1", batch); !errors.Is(err, ErrInvalidTeamIdentityBatch) {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
	if err := ValidateTeamIdentityBatch("", validIdentityBatch()); !errors.Is(err, ErrInvalidTeamIdentityBatch) {
		t.Fatalf("missing run ID error = %v", err)
	}
}
