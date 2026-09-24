package publishedatlas

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"strings"
)

//go:embed preview-0006.json
var currentPreviewData []byte

// PreviewMapDocument loads the current embedded fixture; it does not synchronize external sources.
func PreviewMapDocument() (MapDocument, error) {
	document, err := decodePreviewMapDocument(bytes.NewReader(currentPreviewData))
	if err != nil {
		return MapDocument{}, fmt.Errorf("load current preview: %w", err)
	}
	if document.SnapshotID != PreviewSnapshotID {
		return MapDocument{}, fmt.Errorf("current preview has snapshot %q, want %q", document.SnapshotID, PreviewSnapshotID)
	}
	return document, nil
}

func decodePreviewMapDocument(reader io.Reader) (MapDocument, error) {
	const maxPreviewBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(reader, maxPreviewBytes+1))
	if err != nil {
		return MapDocument{}, fmt.Errorf("read preview map: %w", err)
	}
	if len(data) > maxPreviewBytes {
		return MapDocument{}, fmt.Errorf("preview map exceeds %d bytes", maxPreviewBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document MapDocument
	if err := decoder.Decode(&document); err != nil {
		return MapDocument{}, fmt.Errorf("decode preview map: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return MapDocument{}, fmt.Errorf("preview map has trailing content")
	}
	if err := validatePreviewMapDocument(document); err != nil {
		return MapDocument{}, err
	}
	return document, nil
}

func validatePreviewMapDocument(document MapDocument) error {
	if _, err := ParseSnapshotID(string(document.SnapshotID)); err != nil {
		return fmt.Errorf("invalid preview snapshot ID: %w", err)
	}
	leagues := make(map[LeagueCode]bool)
	for _, league := range document.Leagues {
		if league.Code != LeagueNBA && league.Code != LeagueNFL && league.Code != LeagueMLB && league.Code != LeagueNHL ||
			strings.TrimSpace(league.Name) == "" ||
			league.Path != "/leagues/"+strings.ToLower(string(league.Code)) {
			return fmt.Errorf("league %q has invalid identity or path", league.Code)
		}
		if leagues[league.Code] {
			return fmt.Errorf("duplicate league %q", league.Code)
		}
		leagues[league.Code] = true
	}
	if len(leagues) != 4 || len(document.Places) == 0 {
		return fmt.Errorf("preview map needs four leagues and at least one place")
	}
	venues := make(map[VenueID]bool)
	teams := make(map[TeamID]bool)
	paths := make(map[string]bool)
	for _, place := range document.Places {
		latitude, longitude := place.Coordinates.Latitude, place.Coordinates.Longitude
		if place.VenueID == "" || strings.TrimSpace(place.AccessibleName) == "" ||
			math.IsNaN(latitude) || math.IsNaN(longitude) ||
			math.Abs(latitude) > 90 || math.Abs(longitude) > 180 || len(place.Teams) == 0 {
			return fmt.Errorf("venue %q has invalid identity, coordinates or teams", place.VenueID)
		}
		if venues[place.VenueID] {
			return fmt.Errorf("duplicate venue %q", place.VenueID)
		}
		venues[place.VenueID] = true
		for _, team := range place.Teams {
			if teams[team.TeamID] {
				return fmt.Errorf("duplicate team ID %q", team.TeamID)
			}
			if err := validatePreviewTeam(team, leagues); err != nil {
				return err
			}
			path := team.Preview.Actions.DetailsPath
			if paths[path] {
				return fmt.Errorf("duplicate team details path %q", path)
			}
			teams[team.TeamID] = true
			paths[path] = true
		}
	}
	return nil
}

func validatePreviewTeam(team MapTeam, leagues map[LeagueCode]bool) error {
	preview := team.Preview
	if team.TeamID == "" || strings.TrimSpace(team.Name) == "" || !leagues[team.League] ||
		strings.TrimSpace(team.OfficialGroup) == "" || strings.TrimSpace(team.Division) == "" ||
		strings.TrimSpace(team.VenueName) == "" {
		return fmt.Errorf("team %q has incomplete identity or alignment", team.TeamID)
	}
	if team.Visual.Kind != TeamVisualAbbreviation || strings.TrimSpace(team.Visual.Text) == "" ||
		strings.TrimSpace(team.Visual.Alt) == "" || preview.VenuePhoto.Kind != PhotoPlaceholder ||
		strings.TrimSpace(preview.VenuePhoto.Alt) == "" {
		return fmt.Errorf("team %q has invalid or inaccessible visuals", team.TeamID)
	}
	if preview.TeamID != team.TeamID || preview.TeamName != team.Name ||
		preview.League != team.League || preview.VenueName != team.VenueName ||
		preview.TeamVisual != team.Visual {
		return fmt.Errorf("team %q preview identity differs from map entry", team.TeamID)
	}
	if preview.RegularGameCapacity < 0 || preview.OpenedYear < 0 ||
		(preview.VenueFactsSourceURL != "" && !validPreviewHTTPSURL(preview.VenueFactsSourceURL)) {
		return fmt.Errorf("team %q has invalid venue facts or source", team.TeamID)
	}
	if !validPreviewTeamPath(preview.Actions.DetailsPath) ||
		preview.Actions.SharePath != preview.Actions.DetailsPath {
		return fmt.Errorf("team %q has invalid detail or share path", team.TeamID)
	}
	if !validPreviewHTTPSURL(preview.Actions.OfficialWebsiteURL) {
		return fmt.Errorf("team %q has unsafe official website", team.TeamID)
	}
	return nil
}

func validPreviewTeamPath(path string) bool {
	if !strings.HasPrefix(path, "/teams/") {
		return false
	}
	slug := strings.TrimPrefix(path, "/teams/")
	if slug == "" || len(slug) > 80 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for _, character := range slug {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func validPreviewHTTPSURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil
}
