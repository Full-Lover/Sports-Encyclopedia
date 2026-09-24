package publishedatlas

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestHistoricalPreviewJSONMatchesPublishedSnapshot(t *testing.T) {
	data, err := os.ReadFile("preview-0005.json")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := decodePreviewMapDocument(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("load historical preview: %v", err)
	}
	if want := Preview0005MapDocument(); !reflect.DeepEqual(loaded, want) {
		t.Fatal("historical preview-0005 JSON differs from its published snapshot")
	}
}

func TestEmbeddedPreviewPreservesHistoryAndAddsRaptors(t *testing.T) {
	loaded, err := PreviewMapDocument()
	if err != nil {
		t.Fatalf("load embedded preview: %v", err)
	}
	want := Preview0005MapDocument()
	if loaded.SnapshotID != "preview-0006" || !reflect.DeepEqual(loaded.Leagues, want.Leagues) ||
		len(loaded.Places) != len(want.Places)+1 || !reflect.DeepEqual(loaded.Places[:len(want.Places)], want.Places) {
		t.Fatal("current preview does not preserve preview-0005 and add one venue")
	}
	place := loaded.Places[3]
	if place.VenueID != "scotiabank-arena" || len(place.Teams) != 1 || place.Teams[0].TeamID != "nba-toronto-raptors" {
		t.Fatalf("Toronto venue = %#v", place)
	}
}

func TestDecodePreviewRejectsInvalidDocuments(t *testing.T) {
	valid := Preview0005MapDocument()
	encode := func(document MapDocument) []byte {
		t.Helper()
		data, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"malformed JSON", []byte(`{`)},
		{"oversized input", bytes.Repeat([]byte(" "), (1<<20)+1)},
		{"unknown field", []byte(strings.Replace(string(encode(valid)), `"places":`, `"unexpected":true,"places":`, 1))},
		{"trailing document", append(encode(valid), []byte(` {}`)...)},
		{"missing league", func() []byte {
			changed := Preview0005MapDocument()
			changed.Leagues = changed.Leagues[:3]
			return encode(changed)
		}()},
		{"duplicate venue", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[1].VenueID = changed.Places[0].VenueID
			return encode(changed)
		}()},
		{"duplicate team ID", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[0].Teams[1].TeamID = changed.Places[0].Teams[0].TeamID
			return encode(changed)
		}()},
		{"mismatched preview", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[0].Teams[0].Preview.TeamID = "other"
			return encode(changed)
		}()},
		{"invalid coordinate", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[0].Coordinates.Latitude = 91
			return encode(changed)
		}()},
		{"unsafe website", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[0].Teams[0].Preview.Actions.OfficialWebsiteURL = "javascript:alert(1)"
			return encode(changed)
		}()},
		{"duplicate detail path", func() []byte {
			changed := Preview0005MapDocument()
			changed.Places[0].Teams[1].Preview.Actions.DetailsPath = changed.Places[0].Teams[0].Preview.Actions.DetailsPath
			return encode(changed)
		}()},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodePreviewMapDocument(bytes.NewReader(test.data)); err == nil {
				t.Fatal("invalid document was accepted")
			}
		})
	}
}
