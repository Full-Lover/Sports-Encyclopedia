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
	TeamID               string
	OfficialName         string
	OfficialAbbreviation string
	OfficialGroup        string
	Division             string
	OfficialWebsiteURL   string
}

type DataGroupKind string

const (
	GroupIdentity DataGroupKind = "IDENTITY"
	GroupVenue    DataGroupKind = "VENUE"
	GroupLeader   DataGroupKind = "LEADER"
	GroupRoster   DataGroupKind = "ROSTER"
)

type FactMetadata struct {
	SourceID           string
	CapabilityKey      string
	League             LeagueCode
	Season             string
	SourceURL          string
	FetchedAt          time.Time
	ContentHash        string
	CompletePagination bool
}

// TeamIdentityBatch is one source/capability/league/season observation. An
// incomplete page set may be staged, but it must never imply deletion.
type TeamIdentityBatch struct {
	FactMetadata
	SeasonEvidence *OfficialSeasonEvidence
	Teams          []TeamIdentityFact
}

var ErrInvalidTeamIdentityBatch = errors.New("invalid team identity batch")

// ValidateTeamIdentityBatch checks the boundary shape, not the truth or display
// rights of the source. Registry source policy and evidence review come later.
func ValidateTeamIdentityBatch(runID string, batch TeamIdentityBatch) error {
	if err := validateFactMetadata(runID, batch.FactMetadata); err != nil ||
		len(batch.Teams) == 0 || len(batch.Teams) > 100 {
		return ErrInvalidTeamIdentityBatch
	}
	if batch.SeasonEvidence != nil {
		if err := ValidateOfficialSeasonEvidence(*batch.SeasonEvidence); err != nil ||
			batch.SeasonEvidence.AuthoritySourceID != "" || batch.SeasonEvidence.AuthorityCapabilityKey != "" ||
			batch.SeasonEvidence.League != batch.League || batch.SeasonEvidence.Season != batch.Season {
			return ErrInvalidTeamIdentityBatch
		}
	}
	seen := make(map[string]struct{}, len(batch.Teams))
	for _, team := range batch.Teams {
		if !validToken(team.TeamID, 64) || !validText(team.OfficialName, 160) ||
			!validTeamAbbreviation(team.OfficialAbbreviation) ||
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

func validTeamAbbreviation(value string) bool {
	if len(value) < 2 || len(value) > 5 {
		return false
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func validateFactMetadata(runID string, metadata FactMetadata) error {
	if !validToken(runID, 128) || !validToken(metadata.SourceID, 128) ||
		!validToken(metadata.CapabilityKey, 128) || !validText(metadata.Season, 40) ||
		metadata.FetchedAt.IsZero() || !validHTTPSURL(metadata.SourceURL) {
		return ErrInvalidTeamIdentityBatch
	}
	switch metadata.League {
	case LeagueNBA, LeagueNFL, LeagueMLB, LeagueNHL:
	default:
		return ErrInvalidTeamIdentityBatch
	}
	if digest, err := hex.DecodeString(metadata.ContentHash); err != nil || len(digest) != 32 {
		return ErrInvalidTeamIdentityBatch
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
