package webexperience

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return New(t.TempDir(), publishedatlas.NewMemoryReader(previewDocument(t)))
}

func previewDocument(t *testing.T) publishedatlas.MapDocument {
	t.Helper()
	document, err := publishedatlas.PreviewMapDocument()
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestHomePage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", result.StatusCode, http.StatusOK)
	}
	if contentType := result.Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", contentType)
	}
	for _, expected := range []string{
		`<html lang="en">`,
		`id="map-app"`,
		`data-snapshot-id="preview-0007"`,
		`/assets/app.css`,
		`/assets/app.js`,
	} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("body does not contain %q", expected)
		}
	}
}

func TestHomeWithoutPublishedSnapshot(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	New(t.TempDir(), publishedatlas.NewMemoryReader()).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", response.Code, response.Body.String())
	}
}

func TestFaviconIsEmptyUntilBrandingIsDecided(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	response := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestUnknownPage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestHomeRejectsPost(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()

	newTestHandler(t).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
