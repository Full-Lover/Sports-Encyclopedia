package atlasregistry

import (
	"fmt"
	"sort"
)

func validateCompiledContent(content CompiledRegistryContent) error {
	if len(content.Teams) == 0 {
		return ErrCompilationRejected
	}
	seenTeams := make(map[string]struct{}, len(content.Teams))
	seenSlugs := make(map[string]struct{}, len(content.Teams))
	for _, team := range content.Teams {
		if _, duplicate := seenTeams[team.TeamID]; duplicate {
			return ErrCompilationRejected
		}
		seenTeams[team.TeamID] = struct{}{}
		if _, duplicate := seenSlugs[team.Slug]; duplicate {
			return ErrCompilationRejected
		}
		seenSlugs[team.Slug] = struct{}{}
		if _, err := PolicyForLeague(team.League); err != nil || !validToken(team.TeamID, 64) ||
			team.Season == "" || team.Slug == "" {
			return ErrCompilationRejected
		}
		if err := validateCompiledGroup(team.Identity); err != nil {
			return err
		}
		if err := validateCompiledGroup(team.Venue); err != nil {
			return err
		}
		if err := validateCompiledGroup(team.Leader); err != nil {
			return err
		}
		if err := validateCompiledGroup(team.Roster); err != nil {
			return err
		}
		identity := displayValue(team.Identity)
		if identity == nil || identity.TeamID != team.TeamID ||
			!validTeamAbbreviation(identity.OfficialAbbreviation) ||
			team.Slug != teamSlug(identity.OfficialName) {
			return ErrCompilationRejected
		}
		switch team.Visual.Kind {
		case TeamVisualAbbreviation:
			if team.Visual.Media != nil || team.Visual.Abbreviation != identity.OfficialAbbreviation {
				return ErrCompilationRejected
			}
		case TeamVisualMedia:
			if team.Visual.Abbreviation != "" || validateSelectedMedia(team.Visual.Media, MediaLogo, team.TeamID) != nil {
				return ErrCompilationRejected
			}
		default:
			return ErrCompilationRejected
		}
		venue := displayValue(team.Venue)
		venueID := ""
		if venue != nil {
			venueID = venue.VenueID
		}
		if err := validatePhotoSelection(team.VenuePhoto, MediaVenuePhoto, venueID); err != nil {
			return err
		}
		roster := displayValue(team.Roster)
		expectedPeople := make(map[string]struct{})
		if roster != nil {
			for _, entry := range roster.Entries {
				expectedPeople[entry.PersonID] = struct{}{}
			}
		}
		if len(team.PlayerPhotos) != len(expectedPeople) {
			return ErrCompilationRejected
		}
		for _, binding := range team.PlayerPhotos {
			if _, exists := expectedPeople[binding.PersonID]; !exists {
				return ErrCompilationRejected
			}
			delete(expectedPeople, binding.PersonID)
			if err := validatePhotoSelection(binding.Photo, MediaPlayerPhoto, binding.PersonID); err != nil {
				return err
			}
		}
		if len(expectedPeople) != 0 {
			return ErrCompilationRejected
		}
		if !sort.StringsAreSorted(team.Aliases) || !sort.StringsAreSorted(team.SlugHistory) {
			return ErrCompilationRejected
		}
	}
	return nil
}

func validateCompiledGroup[T any](group CompiledGroup[T]) error {
	switch group.State {
	case StateCurrent:
		if group.Value == nil || group.SyncedAt.IsZero() || len(group.Sources) == 0 ||
			group.LastVerifiedValue != nil || !group.LastVerifiedAt.IsZero() || group.UnavailableReason != "" ||
			!group.ConflictDetectedAt.IsZero() || !group.FailedAt.IsZero() {
			return ErrCompilationRejected
		}
	case StateRetainedAfterFailure:
		if group.Value == nil || group.LastSuccessfulSyncAt.IsZero() || group.FailedAt.IsZero() ||
			len(group.Sources) == 0 || group.UnavailableReason != "" || group.LastVerifiedValue != nil ||
			!group.ConflictDetectedAt.IsZero() {
			return ErrCompilationRejected
		}
	case StateConflict:
		if group.Value != nil || group.Verification != nil || len(group.Sources) < 2 ||
			group.ConflictDetectedAt.IsZero() || group.UnavailableReason != "" ||
			(group.LastVerifiedValue == nil) != group.LastVerifiedAt.IsZero() {
			return ErrCompilationRejected
		}
	case StateUnavailable:
		if group.Value != nil || group.LastVerifiedValue != nil || group.Verification != nil ||
			len(group.Sources) != 0 || (group.UnavailableReason != UnavailableNeverSynced &&
			group.UnavailableReason != UnavailableMissing) {
			return ErrCompilationRejected
		}
	default:
		return ErrCompilationRejected
	}
	if group.Verification != nil {
		if group.Verification.VerifiedAt.IsZero() ||
			(group.Verification.Rule != RuleOfficial && group.Verification.Rule != RuleTwoSources) {
			return ErrCompilationRejected
		}
	}
	return nil
}

func validatePhotoSelection(photo PhotoSelection, kind MediaKind, entityID string) error {
	switch photo.Kind {
	case PhotoPlaceholder:
		if photo.Media != nil {
			return ErrCompilationRejected
		}
	case PhotoReusableMedia:
		if entityID == "" {
			return ErrCompilationRejected
		}
		if err := validateSelectedMedia(photo.Media, kind, entityID); err != nil {
			return err
		}
	default:
		return ErrCompilationRejected
	}
	return nil
}

func validateSelectedMedia(media *SelectedMedia, kind MediaKind, entityID string) error {
	if media == nil || media.Kind != kind || media.EntityID != entityID {
		return ErrCompilationRejected
	}
	batch := MediaAssetBatch{SourceID: media.SourceID, CapabilityKey: media.CapabilityKey,
		Kind: media.Kind, AssetID: media.AssetID, EntityID: media.EntityID,
		FileURL: media.FileURL, SourcePageURL: media.SourcePageURL,
		ContentHash: media.ContentHash, FetchedAt: media.FetchedAt,
		RightsOptions: []MediaRightsOption{media.Rights}}
	if err := ValidateMediaAssetBatch("compiled", batch); err != nil {
		return fmt.Errorf("%w: invalid selected media", ErrCompilationRejected)
	}
	return nil
}
