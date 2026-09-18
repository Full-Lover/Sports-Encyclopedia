package publishedatlas

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryReaderReturnsAnImmutableMapSnapshot(t *testing.T) {
	document := MapDocument{
		SnapshotID: "snapshot-1",
		Leagues: []LeagueSummary{{
			Code: LeagueNBA,
			Name: "National Basketball Association",
			Path: "/leagues/nba",
		}},
		Places: []MapPlace{{
			VenueID:        "venue-1",
			AccessibleName: "Example Arena, home of Example Team",
			Teams:          []MapTeam{{TeamID: "team-1", Name: "Example Team"}},
		}},
	}
	reader := NewMemoryReader(document)

	first, err := reader.Read(context.Background(), ReadRequest{Kind: ReadMap, SnapshotID: "snapshot-1"})
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	first.Map.Leagues[0].Name = "Changed by caller"
	first.Map.Places[0].Teams[0].Name = "Changed by caller"

	second, err := reader.Read(context.Background(), ReadRequest{Kind: ReadMap, SnapshotID: "snapshot-1"})
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if got := second.Map.Leagues[0].Name; got != "National Basketball Association" {
		t.Fatalf("league name = %q", got)
	}
	if got := second.Map.Places[0].Teams[0].Name; got != "Example Team" {
		t.Fatalf("team name = %q", got)
	}
}

func TestMemoryReaderRejectsUnknownSnapshot(t *testing.T) {
	reader := NewMemoryReader()

	_, err := reader.Read(context.Background(), ReadRequest{Kind: ReadMap, SnapshotID: "missing"})
	var fault *ReadFault
	if !errors.As(err, &fault) || fault.Code != FaultSnapshotMissing {
		t.Fatalf("fault = %v, want %s", err, FaultSnapshotMissing)
	}
}

func TestMemoryReaderRejectsUnsupportedRequest(t *testing.T) {
	reader := NewMemoryReader()

	_, err := reader.Read(context.Background(), ReadRequest{})
	var fault *ReadFault
	if !errors.As(err, &fault) || fault.Code != FaultInvalidRequest {
		t.Fatalf("fault = %v, want %s", err, FaultInvalidRequest)
	}
}
