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
		}}, nil
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/teams/example", nil))
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "<script>") ||
		strings.Contains(response.Body.String(), "Visit official team website") {
		t.Fatalf("unsafe page response: %d %q", response.Code, response.Body.String())
	}
}
