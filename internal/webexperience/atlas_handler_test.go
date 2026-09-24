package webexperience

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

type readerFunc func(context.Context, publishedatlas.ReadRequest) (publishedatlas.ReadResult, error)

func (read readerFunc) Read(ctx context.Context, request publishedatlas.ReadRequest) (publishedatlas.ReadResult, error) {
	return read(ctx, request)
}

func TestMapDocument(t *testing.T) {
	document := publishedatlas.PreviewMapDocument()
	handler := New(t.TempDir(), publishedatlas.NewMemoryReader(document))
	request := httptest.NewRequest(http.MethodGet, "/_atlas/snapshots/preview-0005/map", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("cache control = %q", got)
	}
	var result publishedatlas.MapDocument
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.SnapshotID != publishedatlas.PreviewSnapshotID {
		t.Fatalf("snapshot id = %q", result.SnapshotID)
	}
	if len(result.Leagues) != 4 || len(result.Places) != 3 {
		t.Fatalf("map document contains %d leagues and %d places", len(result.Leagues), len(result.Places))
	}
	if got := len(result.Places[0].Teams); got != 2 {
		t.Fatalf("TD Garden contains %d teams, want 2", got)
	}
	if got := result.Places[0].Teams[0].Visual.Kind; got != publishedatlas.TeamVisualAbbreviation {
		t.Fatalf("visual kind = %q", got)
	}
	if team := result.Places[0].Teams[0]; team.OfficialGroup != "Eastern Conference" || team.Division != "Atlantic Division" {
		t.Fatalf("team alignment = %q / %q", team.OfficialGroup, team.Division)
	}
	if preview := result.Places[0].Teams[0].Preview; preview.RegularGameCapacity != 19156 ||
		preview.OpenedYear != 1995 || preview.VenueFactsSourceURL != "https://www.tdgarden.com/about-td-garden" {
		t.Fatalf("venue facts = %#v", preview)
	}
	bruins := result.Places[0].Teams[1]
	if bruins.TeamID != "nhl-boston-bruins" || bruins.League != publishedatlas.LeagueNHL ||
		bruins.OfficialGroup != "Eastern Conference" || bruins.Division != "Atlantic Division" ||
		bruins.Preview.RegularGameCapacity != 17850 || bruins.Preview.OpenedYear != 1995 ||
		bruins.Preview.VenueFactsSourceURL != "https://www.tdgarden.com/about-td-garden" ||
		bruins.Preview.TeamID != bruins.TeamID ||
		bruins.Preview.Actions.OfficialWebsiteURL != "https://www.nhl.com/bruins/" ||
		bruins.Preview.Actions.SharePath != "/teams/boston-bruins" ||
		bruins.Preview.Actions.DetailsPath != "/teams/boston-bruins" {
		t.Fatalf("Bruins map team = %#v", bruins)
	}
	metLife := result.Places[1]
	if metLife.VenueID != "metlife-stadium" || len(metLife.Teams) != 2 ||
		metLife.AccessibleName != "MetLife Stadium, home of New York Giants and New York Jets" ||
		metLife.Coordinates.Latitude != 40.81352 || metLife.Coordinates.Longitude != -74.07435 {
		t.Fatalf("MetLife place = %#v", metLife)
	}
	giants := metLife.Teams[0]
	if giants.TeamID != "nfl-new-york-giants" || giants.League != publishedatlas.LeagueNFL ||
		giants.OfficialGroup != "NFC" || giants.Division != "NFC East" ||
		giants.Preview.TeamID != giants.TeamID || giants.Preview.RegularGameCapacity != 82500 ||
		giants.Preview.OpenedYear != 2010 ||
		giants.Preview.VenueFactsSourceURL != "https://www.metlifestadium.com/stadium/about-metlife-stadium" ||
		giants.Preview.Actions.OfficialWebsiteURL != "https://www.giants.com/" ||
		giants.Preview.Actions.SharePath != "/teams/new-york-giants" ||
		giants.Preview.Actions.DetailsPath != "/teams/new-york-giants" {
		t.Fatalf("Giants map team = %#v", giants)
	}
	jets := metLife.Teams[1]
	if jets.TeamID != "nfl-new-york-jets" || jets.League != publishedatlas.LeagueNFL ||
		jets.OfficialGroup != "AFC" || jets.Division != "AFC East" ||
		jets.Visual.Kind != publishedatlas.TeamVisualAbbreviation || jets.Visual.Text != "NYJ" ||
		jets.Preview.TeamID != jets.TeamID || jets.Preview.RegularGameCapacity != 82500 ||
		jets.Preview.OpenedYear != 2010 ||
		jets.Preview.VenueFactsSourceURL != "https://www.metlifestadium.com/stadium/about-metlife-stadium" ||
		jets.Preview.Actions.OfficialWebsiteURL != "https://www.newyorkjets.com/" ||
		jets.Preview.Actions.SharePath != "/teams/new-york-jets" ||
		jets.Preview.Actions.DetailsPath != "/teams/new-york-jets" {
		t.Fatalf("Jets map team = %#v", jets)
	}
	seattle := result.Places[2]
	if seattle.VenueID != "t-mobile-park" || len(seattle.Teams) != 1 ||
		seattle.Coordinates.Latitude != 47.59093 || seattle.Coordinates.Longitude != -122.33263 {
		t.Fatalf("T-Mobile Park place = %#v", seattle)
	}
	mariners := seattle.Teams[0]
	if mariners.TeamID != "mlb-seattle-mariners" || mariners.League != publishedatlas.LeagueMLB ||
		mariners.OfficialGroup != "American League" || mariners.Division != "American League West" ||
		mariners.Visual.Kind != publishedatlas.TeamVisualAbbreviation || mariners.Visual.Text != "SEA" ||
		mariners.Preview.TeamID != mariners.TeamID || mariners.Preview.RegularGameCapacity != 47943 ||
		mariners.Preview.OpenedYear != 1999 ||
		mariners.Preview.VenueFactsSourceURL != "https://www.mlb.com/mariners/history/ballparks" ||
		mariners.Preview.Actions.OfficialWebsiteURL != "https://www.mlb.com/mariners" ||
		mariners.Preview.Actions.SharePath != "/teams/seattle-mariners" ||
		mariners.Preview.Actions.DetailsPath != "/teams/seattle-mariners" {
		t.Fatalf("Mariners map team = %#v", mariners)
	}
}

func TestHistoricalPreviewMapDocumentsRemainAvailable(t *testing.T) {
	handler := New(t.TempDir(), publishedatlas.NewMemoryReader(
		publishedatlas.PreviewMapDocument(),
		publishedatlas.Preview0004MapDocument(),
		publishedatlas.Preview0003MapDocument(),
		publishedatlas.Preview0002MapDocument(),
		publishedatlas.Preview0001MapDocument(),
	))
	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), `data-snapshot-id="preview-0005"`) {
		t.Fatalf("active home page = %d %q", home.Code, home.Body.String())
	}
	for _, historical := range []struct {
		id              string
		placeCount      int
		firstPlaceTeams int
	}{
		{"preview-0001", 1, 1},
		{"preview-0002", 1, 2},
		{"preview-0003", 2, 2},
		{"preview-0004", 3, 2},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/_atlas/snapshots/"+historical.id+"/map", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", historical.id, response.Code, http.StatusOK)
		}
		var result publishedatlas.MapDocument
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if result.SnapshotID != publishedatlas.SnapshotID(historical.id) ||
			len(result.Places) != historical.placeCount ||
			len(result.Places[0].Teams) != historical.firstPlaceTeams || result.Places[0].Teams[0].Name != "Boston Celtics" {
			t.Fatalf("%s: historical map document = %#v", historical.id, result)
		}
		if historical.id == "preview-0004" && len(result.Places[1].Teams) != 1 {
			t.Fatalf("%s: MetLife historical teams = %#v", historical.id, result.Places[1].Teams)
		}
	}
}

func TestMapDocumentRejectsInvalidSnapshotID(t *testing.T) {
	handler := New(t.TempDir(), publishedatlas.NewMemoryReader())
	request := httptest.NewRequest(http.MethodGet, "/_atlas/snapshots/not%20valid/map", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertAtlasError(t, response, http.StatusBadRequest, "INVALID_REQUEST")
}

func TestMapDocumentHidesMissingSnapshotState(t *testing.T) {
	handler := New(t.TempDir(), publishedatlas.NewMemoryReader())
	request := httptest.NewRequest(http.MethodGet, "/_atlas/snapshots/missing/map", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertAtlasError(t, response, http.StatusNotFound, "SNAPSHOT_NOT_FOUND")
}

func TestMapDocumentRejectsAnEmptyReaderResult(t *testing.T) {
	handler := New(t.TempDir(), readerFunc(func(context.Context, publishedatlas.ReadRequest) (publishedatlas.ReadResult, error) {
		return publishedatlas.ReadResult{}, nil
	}))
	request := httptest.NewRequest(http.MethodGet, "/_atlas/snapshots/preview-0005/map", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertAtlasError(t, response, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func assertAtlasError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d", response.Code, status)
	}
	var payload errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.Error.Code != code || payload.Error.RequestID == "" {
		t.Fatalf("error = %#v", payload.Error)
	}
}
