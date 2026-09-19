// Package publishedatlas owns immutable documents exposed by a published snapshot.
package publishedatlas

import (
	"context"
	"fmt"
)

type SnapshotID string

func ParseSnapshotID(raw string) (SnapshotID, error) {
	if len(raw) == 0 || len(raw) > 64 {
		return "", fmt.Errorf("invalid snapshot id")
	}
	for _, character := range raw {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			continue
		}
		return "", fmt.Errorf("invalid snapshot id")
	}
	return SnapshotID(raw), nil
}

type LeagueCode string

type TeamID string

type VenueID string

const (
	LeagueNBA LeagueCode = "NBA"
	LeagueNFL LeagueCode = "NFL"
	LeagueMLB LeagueCode = "MLB"
	LeagueNHL LeagueCode = "NHL"
)

type LeagueSummary struct {
	Code LeagueCode `json:"code"`
	Name string     `json:"name"`
	Path string     `json:"path"`
}

type TeamVisualKind string

const TeamVisualAbbreviation TeamVisualKind = "ABBREVIATION"

type TeamVisual struct {
	Kind TeamVisualKind `json:"kind"`
	Text string         `json:"text"`
	Alt  string         `json:"alt"`
}

type PhotoKind string

const PhotoPlaceholder PhotoKind = "PLACEHOLDER"

type Photo struct {
	Kind PhotoKind `json:"kind"`
	Alt  string    `json:"alt"`
}

type TeamPreviewActions struct {
	OfficialWebsiteURL string `json:"officialWebsiteUrl"`
	SharePath          string `json:"sharePath"`
	DetailsPath        string `json:"detailsPath"`
}

type TeamPreview struct {
	TeamID     TeamID             `json:"teamId"`
	TeamName   string             `json:"teamName"`
	TeamVisual TeamVisual         `json:"teamVisual"`
	VenuePhoto Photo              `json:"venuePhoto"`
	VenueName  string             `json:"venueName"`
	League     LeagueCode         `json:"league"`
	Actions    TeamPreviewActions `json:"actions"`
}

type MapTeam struct {
	TeamID    TeamID      `json:"teamId"`
	Name      string      `json:"name"`
	League    LeagueCode  `json:"league"`
	VenueName string      `json:"venueName"`
	Visual    TeamVisual  `json:"visual"`
	Preview   TeamPreview `json:"preview"`
}

type MapPlace struct {
	VenueID     VenueID `json:"venueId"`
	Coordinates struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"coordinates"`
	AccessibleName string    `json:"accessibleName"`
	Teams          []MapTeam `json:"teams"`
}

type MapDocument struct {
	SnapshotID SnapshotID      `json:"snapshotId"`
	Leagues    []LeagueSummary `json:"leagues"`
	Places     []MapPlace      `json:"places"`
}

type HomeDocument struct {
	SnapshotID SnapshotID
}

type ReadKind string

const (
	ReadHome ReadKind = "HOME"
	ReadMap  ReadKind = "MAP"
)

type ReadRequest struct {
	Kind       ReadKind
	SnapshotID SnapshotID
}

type ReadResult struct {
	Home *HomeDocument
	Map  *MapDocument
}

type ReadFaultCode string

const (
	FaultInvalidRequest  ReadFaultCode = "INVALID_REQUEST"
	FaultSnapshotMissing ReadFaultCode = "SNAPSHOT_NOT_FOUND"
	FaultNotPublished    ReadFaultCode = "SNAPSHOT_NOT_PUBLISHED"
)

type ReadFault struct {
	Code ReadFaultCode
}

func (fault *ReadFault) Error() string {
	return fmt.Sprintf("published atlas read failed: %s", fault.Code)
}

type Reader interface {
	Read(context.Context, ReadRequest) (ReadResult, error)
}
