package atlasregistry

import (
	"errors"
	"fmt"
	"math"
)

// FactBatch is limited to the four Registry-owned data groups. Source adapters
// cannot supply a synthetic group or bypass the group's validator.
type FactBatch interface {
	factGroup() DataGroupKind
	metadata() FactMetadata
	validate(string) error
}

func (batch TeamIdentityBatch) factGroup() DataGroupKind { return GroupIdentity }
func (batch TeamIdentityBatch) metadata() FactMetadata   { return batch.FactMetadata }
func (batch TeamIdentityBatch) validate(runID string) error {
	return ValidateTeamIdentityBatch(runID, batch)
}

type VenueFact struct {
	TeamID              string
	VenueID             string
	OfficialName        string
	Latitude            float64
	Longitude           float64
	IsPrimary           bool
	RegularGameCapacity int
	OpenedYear          int
}

type VenueBatch struct {
	FactMetadata
	Venues []VenueFact
}

var ErrInvalidVenueBatch = errors.New("invalid venue batch")

func (batch VenueBatch) factGroup() DataGroupKind { return GroupVenue }
func (batch VenueBatch) metadata() FactMetadata   { return batch.FactMetadata }
func (batch VenueBatch) validate(runID string) error {
	return ValidateVenueBatch(runID, batch)
}

func ValidateVenueBatch(runID string, batch VenueBatch) error {
	if validateFactMetadata(runID, batch.FactMetadata) != nil || len(batch.Venues) == 0 || len(batch.Venues) > 200 {
		return ErrInvalidVenueBatch
	}
	seen := make(map[[2]string]struct{}, len(batch.Venues))
	for _, venue := range batch.Venues {
		if !validToken(venue.TeamID, 64) || !validToken(venue.VenueID, 64) ||
			!validText(venue.OfficialName, 160) || math.IsNaN(venue.Latitude) ||
			math.IsNaN(venue.Longitude) || math.IsInf(venue.Latitude, 0) ||
			math.IsInf(venue.Longitude, 0) || venue.Latitude < -90 || venue.Latitude > 90 ||
			venue.Longitude < -180 || venue.Longitude > 180 ||
			venue.RegularGameCapacity < 0 || venue.OpenedYear < 0 || venue.OpenedYear > 2100 {
			return fmt.Errorf("%w: venue %q", ErrInvalidVenueBatch, venue.VenueID)
		}
		key := [2]string{venue.TeamID, venue.VenueID}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("%w: duplicate team venue %q/%q", ErrInvalidVenueBatch, venue.TeamID, venue.VenueID)
		}
		seen[key] = struct{}{}
	}
	return nil
}

type LeaderRole string

const (
	RoleHeadCoach LeaderRole = "HEAD_COACH"
	RoleManager   LeaderRole = "MANAGER"
)

type LeaderFact struct {
	TeamID       string
	PersonID     string
	OfficialName string
	Role         LeaderRole
}

type LeaderBatch struct {
	FactMetadata
	Leaders []LeaderFact
}

var ErrInvalidLeaderBatch = errors.New("invalid leader batch")

func (batch LeaderBatch) factGroup() DataGroupKind { return GroupLeader }
func (batch LeaderBatch) metadata() FactMetadata   { return batch.FactMetadata }
func (batch LeaderBatch) validate(runID string) error {
	return ValidateLeaderBatch(runID, batch)
}

func ValidateLeaderBatch(runID string, batch LeaderBatch) error {
	if validateFactMetadata(runID, batch.FactMetadata) != nil || len(batch.Leaders) == 0 || len(batch.Leaders) > 100 {
		return ErrInvalidLeaderBatch
	}
	seen := make(map[string]struct{}, len(batch.Leaders))
	for _, leader := range batch.Leaders {
		if !validToken(leader.TeamID, 64) || !validToken(leader.PersonID, 64) ||
			!validText(leader.OfficialName, 160) ||
			(leader.Role != RoleHeadCoach && leader.Role != RoleManager) {
			return fmt.Errorf("%w: team %q", ErrInvalidLeaderBatch, leader.TeamID)
		}
		if _, duplicate := seen[leader.TeamID]; duplicate {
			return fmt.Errorf("%w: duplicate team %q", ErrInvalidLeaderBatch, leader.TeamID)
		}
		seen[leader.TeamID] = struct{}{}
	}
	return nil
}
