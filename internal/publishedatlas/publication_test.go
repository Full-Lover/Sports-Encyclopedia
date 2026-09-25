package publishedatlas

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakePublicationStore struct {
	mu          sync.Mutex
	baseline    PublicationBaseline
	leaseRun    string
	leaseToken  string
	staged      map[string]PublicationCandidate
	receipts    map[string]PublicationReceipt
	failStage   bool
	commitCount int
}

func newFakePublicationStore() *fakePublicationStore {
	return &fakePublicationStore{staged: make(map[string]PublicationCandidate), receipts: make(map[string]PublicationReceipt)}
}

func (store *fakePublicationStore) AcquireLease(_ context.Context, runID, token string, _ time.Duration) (LeaseAcquisition, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.leaseRun != "" {
		return LeaseAcquisition{Kind: "IN_PROGRESS", RetryAfter: time.Second}, nil
	}
	if receipt, found := store.receipts[runID]; found {
		return LeaseAcquisition{Kind: "RECOVERED", Receipt: receipt}, nil
	}
	store.leaseRun, store.leaseToken = runID, token
	return LeaseAcquisition{Kind: "ACQUIRED", Baseline: store.baseline}, nil
}

func (store *fakePublicationStore) RenewLease(_ context.Context, runID, token string, _ time.Duration) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.leaseRun == runID && store.leaseToken == token, nil
}

func (store *fakePublicationStore) StageCandidate(_ context.Context, candidate PublicationCandidate, hash string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.failStage {
		return errors.New("candidate storage failed")
	}
	store.staged[hash] = candidate
	return nil
}

func (store *fakePublicationStore) CommitCandidate(_ context.Context, runID, token string, candidate PublicationCandidate, hash string) (CommitOutcome, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.leaseRun != runID || store.leaseToken != token {
		return CommitOutcome{}, ErrLeaseLost
	}
	if _, staged := store.staged[hash]; !staged {
		return CommitOutcome{}, ErrCandidateInvalid
	}
	if store.baseline.SnapshotID != candidate.BaseSnapshotID ||
		store.baseline.RegistryBaselineToken != candidate.BaseRegistryBaselineToken {
		return CommitOutcome{BaseChanged: true, CurrentBaseline: store.baseline}, nil
	}
	if store.baseline.Profile == ProfileV1 && candidate.Profile == ProfilePreview {
		return CommitOutcome{}, ErrProfileDowngrade
	}
	store.commitCount++
	store.baseline = PublicationBaseline{
		SnapshotID: candidate.Map.SnapshotID, RegistryBaselineToken: candidate.NextRegistryBaselineToken, Profile: candidate.Profile,
	}
	receipt := PublicationReceipt{RunID: runID, SnapshotID: candidate.Map.SnapshotID, CandidateHash: hash}
	store.receipts[runID] = receipt
	return CommitOutcome{Receipt: receipt}, nil
}

func (store *fakePublicationStore) ReleaseLease(_ context.Context, runID, token string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.leaseRun == runID && store.leaseToken == token {
		store.leaseRun, store.leaseToken = "", ""
	}
	return nil
}

func publicationCandidate(t *testing.T, runID string, baseline PublicationBaseline, snapshotID SnapshotID) PublicationCandidate {
	t.Helper()
	document, err := PreviewMapDocument()
	if err != nil {
		t.Fatal(err)
	}
	document.SnapshotID = snapshotID
	return PublicationCandidate{
		RunID: runID, BaseSnapshotID: baseline.SnapshotID,
		BaseRegistryBaselineToken: baseline.RegistryBaselineToken,
		NextRegistryBaselineToken: string(snapshotID) + "-registry-token",
		Profile:                   ProfilePreview, SchemaVersion: 1, Map: document,
	}
}

func TestPublisherKeepsOldPointerAfterBuildAndStorageFailures(t *testing.T) {
	store := newFakePublicationStore()
	store.baseline = PublicationBaseline{SnapshotID: "old", RegistryBaselineToken: "old-token", Profile: ProfilePreview}
	publisher := NewPublisher(store, time.Second)
	failed := publisher.Execute(context.Background(), "run-1", func(_ context.Context, _ PublicationBuildContext) (PublicationCandidate, error) {
		return PublicationCandidate{}, errors.New("source unavailable")
	})
	if failed.Kind != "NOT_PUBLISHED" || store.baseline.SnapshotID != "old" {
		t.Fatalf("build failure = %#v, baseline = %#v", failed, store.baseline)
	}
	store.failStage = true
	failed = publisher.Execute(context.Background(), "run-2", func(_ context.Context, context PublicationBuildContext) (PublicationCandidate, error) {
		return publicationCandidate(t, "run-2", context.Baseline, "new"), nil
	})
	if failed.Kind != "NOT_PUBLISHED" || store.baseline.SnapshotID != "old" || store.commitCount != 0 {
		t.Fatalf("storage failure = %#v, baseline = %#v", failed, store.baseline)
	}
}

func TestPublisherRebasesOnceAfterPointerChange(t *testing.T) {
	store := newFakePublicationStore()
	publisher := NewPublisher(store, time.Second)
	var phases []BuildPhase
	result := publisher.Execute(context.Background(), "run-rebase", func(_ context.Context, context PublicationBuildContext) (PublicationCandidate, error) {
		phases = append(phases, context.Phase)
		if context.Phase == BuildInitial {
			store.mu.Lock()
			store.baseline = PublicationBaseline{SnapshotID: "other", RegistryBaselineToken: "other-token", Profile: ProfilePreview}
			store.mu.Unlock()
		}
		return publicationCandidate(t, "run-rebase", context.Baseline, "published"), nil
	})
	if result.Kind != "PUBLISHED" || result.Receipt.SnapshotID != "published" ||
		len(phases) != 2 || phases[0] != BuildInitial || phases[1] != BuildRebase || store.commitCount != 1 {
		t.Fatalf("rebase result = %#v, phases = %#v", result, phases)
	}
}

func TestPublisherRejectsLostLeaseWithoutChangingPointer(t *testing.T) {
	store := newFakePublicationStore()
	publisher := NewPublisher(store, time.Second)
	result := publisher.Execute(context.Background(), "run-lost", func(_ context.Context, context PublicationBuildContext) (PublicationCandidate, error) {
		store.mu.Lock()
		store.leaseToken = "replacement-token"
		store.mu.Unlock()
		return publicationCandidate(t, "run-lost", context.Baseline, "should-not-publish"), nil
	})
	if result.Kind != "NOT_PUBLISHED" || !errors.Is(result.Fault, ErrLeaseLost) || store.baseline.SnapshotID != "" {
		t.Fatalf("lost lease result = %#v, baseline = %#v", result, store.baseline)
	}
}

func TestPublisherPreventsProfileDowngrade(t *testing.T) {
	store := newFakePublicationStore()
	store.baseline = PublicationBaseline{SnapshotID: "v1", RegistryBaselineToken: "v1-token", Profile: ProfileV1}
	result := NewPublisher(store, time.Second).Execute(context.Background(), "run-downgrade", func(_ context.Context, context PublicationBuildContext) (PublicationCandidate, error) {
		return publicationCandidate(t, "run-downgrade", context.Baseline, "preview"), nil
	})
	if result.Kind != "NOT_PUBLISHED" || !errors.Is(result.Fault, ErrProfileDowngrade) || store.baseline.SnapshotID != "v1" {
		t.Fatalf("downgrade result = %#v, baseline = %#v", result, store.baseline)
	}
}
