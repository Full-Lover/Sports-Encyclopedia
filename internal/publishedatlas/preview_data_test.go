package publishedatlas

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEmbeddedPreviewMatchesPublishedSnapshot(t *testing.T) {
	loaded, err := PreviewMapDocument()
	if err != nil {
		t.Fatalf("load embedded preview: %v", err)
	}
	if want := Preview0005MapDocument(); !reflect.DeepEqual(loaded, want) {
		t.Fatalf("embedded preview differs from published preview-0005")
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
