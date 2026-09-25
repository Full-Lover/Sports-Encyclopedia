package atlasregistry

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateMediaAssetBatch(t *testing.T) {
	batch := MediaAssetBatch{
		SourceID: "commons", CapabilityKey: "team-logo", Kind: MediaLogo,
		AssetID: "file-1", EntityID: "nba-boston-celtics",
		FileURL:       "https://upload.wikimedia.org/example.png",
		SourcePageURL: "https://commons.wikimedia.org/wiki/File:Example.png",
		ContentHash:   strings.Repeat("a", 64), FetchedAt: time.Now().UTC(),
	}
	if err := ValidateMediaAssetBatch("run-1", batch); err != nil {
		t.Fatalf("unlicensed file may be tracked but not displayed: %v", err)
	}
	batch.RightsOptions = []MediaRightsOption{{
		Kind: RightsOpenLicense, Author: "Example Photographer",
		SourceURL: batch.SourcePageURL, LicenseName: "CC BY-SA",
		LicenseVersion: "4.0", LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0/",
		RetrievedAt: time.Now().UTC(),
	}}
	if err := ValidateMediaAssetBatch("run-1", batch); err != nil {
		t.Fatal(err)
	}
	batch.RightsOptions[0].LicenseURL = ""
	if err := ValidateMediaAssetBatch("run-1", batch); !errors.Is(err, ErrInvalidMediaBatch) {
		t.Fatalf("missing license URL error = %v", err)
	}
}
