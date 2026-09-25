package atlasregistry

import (
	"encoding/hex"
	"errors"
	"time"
)

type MediaKind string

const (
	MediaLogo        MediaKind = "LOGO"
	MediaVenuePhoto  MediaKind = "VENUE_PHOTO"
	MediaPlayerPhoto MediaKind = "PLAYER_PHOTO"
)

type RightsKind string

const (
	RightsOpenLicense  RightsKind = "OPEN_LICENSE"
	RightsPublicDomain RightsKind = "PUBLIC_DOMAIN"
)

type MediaRightsOption struct {
	Kind           RightsKind
	Author         string
	SourceURL      string
	LicenseName    string
	LicenseVersion string
	LicenseURL     string
	PublicBasis    string
	RetrievedAt    time.Time
}

// MediaAssetBatch represents one source file and its independently recorded
// rights. Empty RightsOptions are retained for review but cannot be displayed.
type MediaAssetBatch struct {
	SourceID      string
	CapabilityKey string
	Kind          MediaKind
	AssetID       string
	EntityID      string
	FileURL       string
	SourcePageURL string
	ContentHash   string
	FetchedAt     time.Time
	RightsOptions []MediaRightsOption
}

var ErrInvalidMediaBatch = errors.New("invalid media batch")

func ValidateMediaAssetBatch(runID string, batch MediaAssetBatch) error {
	if !validToken(runID, 128) || !validToken(batch.SourceID, 128) ||
		!validToken(batch.CapabilityKey, 128) || !validToken(batch.AssetID, 128) ||
		!validToken(batch.EntityID, 64) || !validHTTPSURL(batch.FileURL) ||
		!validHTTPSURL(batch.SourcePageURL) || batch.FetchedAt.IsZero() ||
		len(batch.RightsOptions) > 8 {
		return ErrInvalidMediaBatch
	}
	switch batch.Kind {
	case MediaLogo, MediaVenuePhoto, MediaPlayerPhoto:
	default:
		return ErrInvalidMediaBatch
	}
	if digest, err := hex.DecodeString(batch.ContentHash); err != nil || len(digest) != 32 {
		return ErrInvalidMediaBatch
	}
	for _, option := range batch.RightsOptions {
		if !validHTTPSURL(option.SourceURL) || option.RetrievedAt.IsZero() {
			return ErrInvalidMediaBatch
		}
		switch option.Kind {
		case RightsOpenLicense:
			if !validText(option.Author, 160) || !validText(option.LicenseName, 120) ||
				!validHTTPSURL(option.LicenseURL) ||
				(option.LicenseVersion != "" && !validText(option.LicenseVersion, 40)) ||
				option.PublicBasis != "" {
				return ErrInvalidMediaBatch
			}
		case RightsPublicDomain:
			if !validText(option.PublicBasis, 240) || option.LicenseName != "" ||
				option.LicenseVersion != "" || option.LicenseURL != "" ||
				(option.Author != "" && !validText(option.Author, 160)) {
				return ErrInvalidMediaBatch
			}
		default:
			return ErrInvalidMediaBatch
		}
	}
	return nil
}
