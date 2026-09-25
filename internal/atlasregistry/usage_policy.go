package atlasregistry

import (
	"errors"
	"time"
)

type CapabilityKind string

const (
	CapabilityFacts CapabilityKind = "FACTS"
	CapabilityMedia CapabilityKind = "MEDIA"
)

// UsagePolicy is trusted repository curation, never an adapter-supplied claim.
// Active and AllowWebsite default to false when omitted.
type UsagePolicy struct {
	SourceID          string
	CapabilityKey     string
	Kind              CapabilityKind
	League            LeagueCode
	FactGroups        []DataGroupKind
	MediaKinds        []MediaKind
	OfficialAuthority bool
	IndependenceKey   string
	AllowWebsite      bool
	AllowRepository   bool
	AllowPublicAPI    bool
	EvidenceURL       string
	ReviewedBy        string
	ReviewedAt        time.Time
	Active            bool
}

var ErrInvalidUsagePolicy = errors.New("invalid registry usage policy")

func ValidateUsagePolicy(policy UsagePolicy) error {
	if !validToken(policy.SourceID, 128) || !validToken(policy.CapabilityKey, 128) ||
		!validToken(policy.IndependenceKey, 128) || !validHTTPSURL(policy.EvidenceURL) ||
		!validText(policy.ReviewedBy, 160) || policy.ReviewedAt.IsZero() {
		return ErrInvalidUsagePolicy
	}
	switch policy.Kind {
	case CapabilityFacts:
		if len(policy.FactGroups) == 0 || len(policy.FactGroups) > 4 || len(policy.MediaKinds) != 0 {
			return ErrInvalidUsagePolicy
		}
		switch policy.League {
		case LeagueNBA, LeagueNFL, LeagueMLB, LeagueNHL:
		default:
			return ErrInvalidUsagePolicy
		}
		seen := make(map[DataGroupKind]struct{}, len(policy.FactGroups))
		for _, group := range policy.FactGroups {
			switch group {
			case GroupIdentity, GroupVenue, GroupLeader, GroupRoster:
			default:
				return ErrInvalidUsagePolicy
			}
			if _, duplicate := seen[group]; duplicate {
				return ErrInvalidUsagePolicy
			}
			seen[group] = struct{}{}
		}
	case CapabilityMedia:
		if policy.League != "" || len(policy.FactGroups) != 0 ||
			len(policy.MediaKinds) == 0 || len(policy.MediaKinds) > 3 {
			return ErrInvalidUsagePolicy
		}
		seen := make(map[MediaKind]struct{}, len(policy.MediaKinds))
		for _, kind := range policy.MediaKinds {
			switch kind {
			case MediaLogo, MediaVenuePhoto, MediaPlayerPhoto:
			default:
				return ErrInvalidUsagePolicy
			}
			if _, duplicate := seen[kind]; duplicate {
				return ErrInvalidUsagePolicy
			}
			seen[kind] = struct{}{}
		}
	default:
		return ErrInvalidUsagePolicy
	}
	return nil
}
