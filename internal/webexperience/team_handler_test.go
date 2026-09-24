package webexperience

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

func TestPreviewTeamPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/teams/boston-celtics", nil)
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Boston Celtics", "TD Garden", "Eastern Conference", "Atlantic Division", `content="noindex"`,
		`href="https://www.nba.com/celtics/"`, "Not available in this preview.",
		"Regular-game capacity: 19,156", "Opened: 1995",
		`href="https://www.tdgarden.com/about-td-garden"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
}

func TestPreviewBruinsTeamPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/teams/boston-bruins", nil)
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Boston Bruins", "TD Garden", "Eastern Conference", "Atlantic Division", `content="noindex"`,
		`href="https://www.nhl.com/bruins/"`, "Not available in this preview.",
		"Regular-game capacity: 17,850", "Opened: 1995",
		`href="https://www.tdgarden.com/about-td-garden"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
}

func TestPreviewGiantsTeamPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/teams/new-york-giants", nil)
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"New York Giants", "MetLife Stadium", "NFC", "NFC East", `content="noindex"`,
		`href="https://www.giants.com/"`, "Not available in this preview.",
		"Regular-game capacity: 82,500", "Opened: 2010",
		`href="https://www.metlifestadium.com/stadium/about-metlife-stadium"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
}

func TestPreviewJetsTeamPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/teams/new-york-jets", nil)
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"New York Jets", "MetLife Stadium", "AFC", "AFC East", `content="noindex"`,
		`href="https://www.newyorkjets.com/"`, "Not available in this preview.",
		"Regular-game capacity: 82,500", "Opened: 2010",
		`href="https://www.metlifestadium.com/stadium/about-metlife-stadium"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
}

func TestPreviewMarinersTeamPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/teams/seattle-mariners", nil)
	response := httptest.NewRecorder()
	newTestHandler(t).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, expected := range []string{
		"Seattle Mariners", "T-Mobile Park", "American League", "American League West", `content="noindex"`,
		`href="https://www.mlb.com/mariners"`, "Not available in this preview.",
		"Regular-game capacity: 47,943", "Opened: 1999",
		`href="https://www.mlb.com/mariners/history/ballparks"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("page does not contain %q", expected)
		}
	}
}

func TestTeamPageRejectsUnknownAndInvalidSlugs(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{"/teams/unknown-team", "/teams/not%20valid"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestTeamPageWithoutPublishedSnapshot(t *testing.T) {
	handler := New(t.TempDir(), publishedatlas.NewMemoryReader())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teams/boston-celtics", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestTeamPageEscapesNamesAndRejectsUnsafeOfficialLink(t *testing.T) {
	handler := New(t.TempDir(), readerFunc(func(context.Context, publishedatlas.ReadRequest) (publishedatlas.ReadResult, error) {
		return publishedatlas.ReadResult{Team: &publishedatlas.TeamPageDocument{
			Name: "<script>alert(1)</script>", OfficialWebsiteURL: "javascript:alert(1)",
			VenueFactsSourceURL: "javascript:alert(1)", RegularGameCapacity: -1, OpenedYear: 3000,
		}}, nil
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teams/example", nil))
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "<script>") ||
		strings.Contains(response.Body.String(), "Visit official team website") ||
		strings.Contains(response.Body.String(), "Source:") ||
		!strings.Contains(response.Body.String(), "Regular-game capacity: Not available") ||
		!strings.Contains(response.Body.String(), "Opened: Not available") {
		t.Fatalf("unsafe page response: %d %q", response.Code, response.Body.String())
	}
}
