// Package atlasregistry owns normalized, sourced facts before publication.
package atlasregistry

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type LeagueCode string

const (
	LeagueNBA LeagueCode = "NBA"
	LeagueNFL LeagueCode = "NFL"
	LeagueMLB LeagueCode = "MLB"
	LeagueNHL LeagueCode = "NHL"
)

type TeamIdentityFact struct {
	TeamID             string
	OfficialName       string
	OfficialGroup      string
	Division           string
	OfficialWebsiteURL string
}

// TeamIdentityBatch is one source/capability/league/season observation. An
// incomplete page set may be staged, but it must never imply deletion.
type TeamIdentityBatch struct {
	SourceID           string
	CapabilityKey      string
	League             LeagueCode
	Season             string
	SourceURL          string
	FetchedAt          time.Time
	ContentHash        string
	CompletePagination bool
	Teams              []TeamIdentityFact
}

var ErrInvalidTeamIdentityBatch = errors.New("invalid team identity batch")

// ValidateTeamIdentityBatch checks the boundary shape, not the truth or display
// rights of the source. Registry source policy and evidence review come later.
func ValidateTeamIdentityBatch(runID string, batch TeamIdentityBatch) error {
	if !validToken(runID, 128) || !validToken(batch.SourceID, 128) ||
		!validToken(batch.CapabilityKey, 128) || !validText(batch.Season, 40) ||
		batch.FetchedAt.IsZero() || !validHTTPSURL(batch.SourceURL) ||
		len(batch.Teams) == 0 || len(batch.Teams) > 100 {
		return ErrInvalidTeamIdentityBatch
	}
	switch batch.League {
	case LeagueNBA, LeagueNFL, LeagueMLB, LeagueNHL:
	default:
		return ErrInvalidTeamIdentityBatch
	}
	if digest, err := hex.DecodeString(batch.ContentHash); err != nil || len(digest) != 32 {
		return ErrInvalidTeamIdentityBatch
	}
	seen := make(map[string]struct{}, len(batch.Teams))
	for _, team := range batch.Teams {
		if !validToken(team.TeamID, 64) || !validText(team.OfficialName, 160) ||
			!validText(team.OfficialGroup, 80) ||
			(team.Division != "" && !validText(team.Division, 80)) ||
			!validHTTPSURL(team.OfficialWebsiteURL) {
			return fmt.Errorf("%w: team %q", ErrInvalidTeamIdentityBatch, team.TeamID)
		}
		if _, duplicate := seen[team.TeamID]; duplicate {
			return fmt.Errorf("%w: duplicate team %q", ErrInvalidTeamIdentityBatch, team.TeamID)
		}
		seen[team.TeamID] = struct{}{}
	}
	return nil
}

func validToken(value string, maximum int) bool {
	if len(value) == 0 || len(value) > maximum {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' || character == ':' {
			continue
		}
		return false
	}
	return true
}

func validText(value string, maximum int) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validHTTPSURL(raw string) bool {
	if len(raw) == 0 || len(raw) > 2048 {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" &&
		parsed.User == nil && parsed.Fragment == ""
}
