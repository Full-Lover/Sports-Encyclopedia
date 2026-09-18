// Package publishedatlas owns immutable documents exposed by a published snapshot.
package publishedatlas

import (
	"context"
	"fmt"
)

type SnapshotID string

type LeagueCode string

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

type TeamVisual struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	Alt  string `json:"alt"`
}

type Photo struct {
	Kind string `json:"kind"`
	Alt  string `json:"alt"`
}

type TeamPreview struct {
	TeamID     string     `json:"teamId"`
	TeamName   string     `json:"teamName"`
	TeamVisual TeamVisual `json:"teamVisual"`
	VenuePhoto Photo      `json:"venuePhoto"`
	VenueName  string     `json:"venueName"`
	League     LeagueCode `json:"league"`
	Actions    struct {
		OfficialWebsiteURL string `json:"officialWebsiteUrl"`
		SharePath          string `json:"sharePath"`
		DetailsPath        string `json:"detailsPath"`
	} `json:"actions"`
}

type MapTeam struct {
	TeamID    string      `json:"teamId"`
	Name      string      `json:"name"`
	League    LeagueCode  `json:"league"`
	VenueName string      `json:"venueName"`
	Visual    TeamVisual  `json:"visual"`
	Preview   TeamPreview `json:"preview"`
}

type MapPlace struct {
	VenueID     string `json:"venueId"`
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

type ReadKind string

const ReadMap ReadKind = "MAP"

type ReadRequest struct {
	Kind       ReadKind
	SnapshotID SnapshotID
}

type ReadResult struct {
	Map *MapDocument
}

type ReadFaultCode string

const (
	FaultInvalidRequest  ReadFaultCode = "INVALID_REQUEST"
	FaultSnapshotMissing ReadFaultCode = "SNAPSHOT_NOT_FOUND"
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
