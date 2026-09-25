package publishedatlas

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"time"
)

type PublicationProfile string

const (
	ProfilePreview PublicationProfile = "PREVIEW"
	ProfileV1      PublicationProfile = "V1"
)

type PublicationBaseline struct {
	SnapshotID            SnapshotID
	RegistryBaselineToken string
	Profile               PublicationProfile
}

type PublicationCandidate struct {
	RunID                     string
	BaseSnapshotID            SnapshotID
	BaseRegistryBaselineToken string
	NextRegistryBaselineToken string
	Profile                   PublicationProfile
	SchemaVersion             int
	Map                       MapDocument
}

type PublicationReceipt struct {
	RunID         string
	SnapshotID    SnapshotID
	CandidateHash string
}

type BuildPhase string

const (
	BuildInitial BuildPhase = "INITIAL"
	BuildRebase  BuildPhase = "REBASE"
)

type PublicationBuildContext struct {
	Phase           BuildPhase
	Baseline        PublicationBaseline
	SupersededRunID string
}

type PublicationExecution struct {
	Kind       string
	Receipt    PublicationReceipt
	Fault      error
	RetryAfter time.Duration
}

var (
	ErrPublicationInProgress = errors.New("publication in progress")
	ErrLeaseLost             = errors.New("publication lease lost")
	ErrBaseChanged           = errors.New("publication base changed")
	ErrCandidateInvalid      = errors.New("publication candidate invalid")
	ErrProfileDowngrade      = errors.New("publication profile downgrade")
)

type LeaseAcquisition struct {
	Kind            string // ACQUIRED, IN_PROGRESS, RECOVERED
	Baseline        PublicationBaseline
	SupersededRunID string
	Receipt         PublicationReceipt
	RetryAfter      time.Duration
}

type CommitOutcome struct {
	Receipt         PublicationReceipt
	CurrentBaseline PublicationBaseline
	BaseChanged     bool
}

// PublicationStore owns the transaction boundary. Commit checks the lease and
// current pointer together; implementations must never expose staged candidates.
type PublicationStore interface {
	AcquireLease(context.Context, string, string, time.Duration) (LeaseAcquisition, error)
	RenewLease(context.Context, string, string, time.Duration) (bool, error)
	StageCandidate(context.Context, PublicationCandidate, string) error
	CommitCandidate(context.Context, string, string, PublicationCandidate, string) (CommitOutcome, error)
	ReleaseLease(context.Context, string, string) error
}

type Publisher struct {
	store    PublicationStore
	leaseTTL time.Duration
}

func NewPublisher(store PublicationStore, leaseTTL time.Duration) *Publisher {
	return &Publisher{store: store, leaseTTL: leaseTTL}
}

func (publisher *Publisher) Execute(ctx context.Context, runID string, build func(context.Context, PublicationBuildContext) (PublicationCandidate, error)) PublicationExecution {
	if publisher == nil || publisher.store == nil || publisher.leaseTTL < time.Second || runID == "" || build == nil {
		return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: ErrCandidateInvalid}
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
	}
	token := hex.EncodeToString(tokenBytes)
	acquired, err := publisher.store.AcquireLease(ctx, runID, token, publisher.leaseTTL)
	if err != nil {
		return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
	}
	switch acquired.Kind {
	case "IN_PROGRESS":
		return PublicationExecution{Kind: "IN_PROGRESS", RetryAfter: acquired.RetryAfter}
	case "RECOVERED":
		return PublicationExecution{Kind: "RECOVERED", Receipt: acquired.Receipt}
	case "ACQUIRED":
	default:
		return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: fmt.Errorf("unknown lease result %q", acquired.Kind)}
	}
	defer func() { _ = publisher.store.ReleaseLease(context.Background(), runID, token) }()
	buildCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var leaseLost atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(publisher.leaseTTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-buildCtx.Done():
				return
			case <-ticker.C:
				valid, err := publisher.store.RenewLease(buildCtx, runID, token, publisher.leaseTTL)
				if err != nil || !valid {
					leaseLost.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	defer func() { cancel(); <-done }()

	baseline := acquired.Baseline
	for attempt := 0; attempt < 2; attempt++ {
		phase := BuildInitial
		if attempt == 1 {
			phase = BuildRebase
		}
		candidate, err := build(buildCtx, PublicationBuildContext{
			Phase: phase, Baseline: baseline, SupersededRunID: acquired.SupersededRunID,
		})
		if leaseLost.Load() {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: ErrLeaseLost}
		}
		if err != nil {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
		}
		if err := validatePublicationCandidate(candidate, runID, baseline); err != nil {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
		}
		hash, err := candidateHash(candidate)
		if err != nil {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
		}
		if err := publisher.store.StageCandidate(buildCtx, candidate, hash); err != nil {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
		}
		outcome, err := publisher.store.CommitCandidate(buildCtx, runID, token, candidate, hash)
		if err != nil {
			return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: err}
		}
		if !outcome.BaseChanged {
			return PublicationExecution{Kind: "PUBLISHED", Receipt: outcome.Receipt}
		}
		baseline = outcome.CurrentBaseline
	}
	return PublicationExecution{Kind: "NOT_PUBLISHED", Fault: ErrBaseChanged}
}

func validatePublicationCandidate(candidate PublicationCandidate, runID string, baseline PublicationBaseline) error {
	if candidate.RunID != runID || candidate.BaseSnapshotID != baseline.SnapshotID ||
		candidate.BaseRegistryBaselineToken != baseline.RegistryBaselineToken ||
		candidate.NextRegistryBaselineToken == "" || candidate.SchemaVersion != 1 ||
		(candidate.Profile != ProfilePreview && candidate.Profile != ProfileV1) ||
		candidate.Map.SnapshotID == "" {
		return ErrCandidateInvalid
	}
	if _, err := ParseSnapshotID(string(candidate.Map.SnapshotID)); err != nil {
		return fmt.Errorf("%w: %v", ErrCandidateInvalid, err)
	}
	if err := validatePreviewMapDocument(candidate.Map); err != nil {
		return fmt.Errorf("%w: %v", ErrCandidateInvalid, err)
	}
	return nil
}

func candidateHash(candidate PublicationCandidate) (string, error) {
	document := cloneMapDocument(candidate.Map)
	sort.Slice(document.Leagues, func(i, j int) bool { return document.Leagues[i].Code < document.Leagues[j].Code })
	sort.Slice(document.Places, func(i, j int) bool { return document.Places[i].VenueID < document.Places[j].VenueID })
	for i := range document.Places {
		sort.Slice(document.Places[i].Teams, func(a, b int) bool {
			return document.Places[i].Teams[a].TeamID < document.Places[i].Teams[b].TeamID
		})
	}
	canonical := struct {
		SchemaVersion             int
		Profile                   PublicationProfile
		BaseSnapshotID            SnapshotID
		BaseRegistryBaselineToken string
		NextRegistryBaselineToken string
		Content                   MapDocument
	}{candidate.SchemaVersion, candidate.Profile, candidate.BaseSnapshotID,
		candidate.BaseRegistryBaselineToken, candidate.NextRegistryBaselineToken, document}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
