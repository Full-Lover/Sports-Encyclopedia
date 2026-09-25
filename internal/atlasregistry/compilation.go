package atlasregistry

import (
	"context"
	"time"
)

type PublicationProfile string

const (
	ProfilePreview PublicationProfile = "PREVIEW"
	ProfileV1      PublicationProfile = "V1"
)

type DataGroupState string

const (
	StateCurrent              DataGroupState = "CURRENT"
	StateRetainedAfterFailure DataGroupState = "RETAINED_AFTER_FAILURE"
	StateConflict             DataGroupState = "CONFLICT"
	StateUnavailable          DataGroupState = "UNAVAILABLE"
)

type UnavailableReason string

const (
	UnavailableNeverSynced UnavailableReason = "NEVER_SYNCED"
	UnavailableMissing     UnavailableReason = "MISSING"
)

type VerificationRule string

const (
	RuleOfficial   VerificationRule = "OFFICIAL"
	RuleTwoSources VerificationRule = "TWO_SOURCES"
)

type Verification struct {
	VerifiedAt time.Time
	Rule       VerificationRule
}

type SourceCitation struct {
	SourceID      string
	CapabilityKey string
	SourceURL     string
	FetchedAt     time.Time
}

type CompiledGroup[T any] struct {
	State                DataGroupState
	Value                *T
	LastVerifiedValue    *T
	LastVerifiedAt       time.Time
	SyncedAt             time.Time
	LastSuccessfulSyncAt time.Time
	FailedAt             time.Time
	PossiblyStale        bool
	ConflictDetectedAt   time.Time
	Verification         *Verification
	Sources              []SourceCitation
	UnavailableReason    UnavailableReason
}

type SelectedMedia struct {
	SourceID      string
	CapabilityKey string
	AssetID       string
	Kind          MediaKind
	EntityID      string
	FileURL       string
	SourcePageURL string
	ContentHash   string
	FetchedAt     time.Time
	Rights        MediaRightsOption
}

type TeamVisualKind string

const (
	TeamVisualMedia        TeamVisualKind = "REUSABLE_MEDIA"
	TeamVisualAbbreviation TeamVisualKind = "ABBREVIATION"
)

type TeamVisual struct {
	Kind         TeamVisualKind
	Media        *SelectedMedia
	Abbreviation string
}

type PhotoSelectionKind string

const (
	PhotoReusableMedia PhotoSelectionKind = "REUSABLE_MEDIA"
	PhotoPlaceholder   PhotoSelectionKind = "PLACEHOLDER"
)

type PhotoSelection struct {
	Kind  PhotoSelectionKind
	Media *SelectedMedia
}

type PlayerPhotoSelection struct {
	PersonID string
	Photo    PhotoSelection
}

type CompiledTeam struct {
	TeamID       string
	League       LeagueCode
	Season       string
	Offseason    bool
	Slug         string
	SlugHistory  []string
	Aliases      []string
	Identity     CompiledGroup[TeamIdentityFact]
	Venue        CompiledGroup[VenueFact]
	Leader       CompiledGroup[LeaderFact]
	Roster       CompiledGroup[TeamRosterFact]
	Visual       TeamVisual
	VenuePhoto   PhotoSelection
	PlayerPhotos []PlayerPhotoSelection
}

type CompiledRegistryContent struct {
	Teams          []CompiledTeam
	SeasonEvidence []OfficialSeasonEvidence
}

type CompileRequest struct {
	RunID                    string
	BaselineToken            string
	Profile                  PublicationProfile
	ConfigurationFingerprint string
	RequestedAt              time.Time
	FailedGroups             []FailedGroup
}

type FailedGroup struct {
	League        LeagueCode
	TeamID        string
	Group         DataGroupKind
	FailedAt      time.Time
	PossiblyStale bool
}

type RegistryCompilation struct {
	RunID             string
	BaselineTokenUsed string
	NextBaselineToken string
	Profile           PublicationProfile
	SchemaVersion     int
	Content           CompiledRegistryContent
	FailedGroups      []FailedGroup
}

// Registry is the only refresh-facing capability. Policy installation and
// migrations are intentionally absent from this interface.
type Registry interface {
	StageFacts(context.Context, string, FactBatch) (ReconcileReceipt, error)
	StageMedia(context.Context, string, MediaAssetBatch) (ReconcileReceipt, error)
	AbandonRun(context.Context, string) error
	CompilePublicationContent(context.Context, CompileRequest) (RegistryCompilation, error)
}
