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

func TestMemoryReaderReturnsTheActiveHomeSnapshot(t *testing.T) {
	reader := NewMemoryReader(MapDocument{SnapshotID: "snapshot-1"})

	result, err := reader.Read(context.Background(), ReadRequest{Kind: ReadHome})
	if err != nil {
		t.Fatalf("read home: %v", err)
	}
	if result.Home == nil || result.Home.SnapshotID != "snapshot-1" {
		t.Fatalf("home = %#v", result.Home)
	}
}

func TestMemoryReaderRejectsHomeWithoutPublication(t *testing.T) {
	_, err := NewMemoryReader().Read(context.Background(), ReadRequest{Kind: ReadHome})
	var fault *ReadFault
	if !errors.As(err, &fault) || fault.Code != FaultNotPublished {
		t.Fatalf("fault = %v, want %s", err, FaultNotPublished)
	}
}

func TestMemoryReaderReturnsPreviewTeamPage(t *testing.T) {
	reader := NewMemoryReader(PreviewMapDocument())
	result, err := reader.Read(context.Background(), ReadRequest{Kind: ReadTeam, Slug: "boston-celtics"})
	if err != nil {
		t.Fatalf("read team: %v", err)
	}
	if result.Team == nil || result.Team.SnapshotID != PreviewSnapshotID ||
		result.Team.Name != "Boston Celtics" || result.Team.VenueName != "TD Garden" || !result.Team.Preview {
		t.Fatalf("team = %#v", result.Team)
	}
}

func TestMemoryReaderRejectsUnknownTeam(t *testing.T) {
	reader := NewMemoryReader(PreviewMapDocument())
	_, err := reader.Read(context.Background(), ReadRequest{Kind: ReadTeam, Slug: "unknown-team"})
	var fault *ReadFault
	if !errors.As(err, &fault) || fault.Code != FaultTeamMissing {
		t.Fatalf("fault = %v, want %s", err, FaultTeamMissing)
	}
}
