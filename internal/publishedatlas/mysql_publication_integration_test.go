package publishedatlas

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func TestMySQLPublicationLifecycle(t *testing.T) {
	dsn := os.Getenv("SPORTS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set SPORTS_TEST_MYSQL_DSN for the dedicated MySQL test database")
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	if !strings.Contains(strings.ToLower(config.DBName), "test") {
		t.Skip("refusing to write publication fixtures outside a database named *test*")
	}
	config.ParseTime = true
	config.Loc = time.UTC
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect to test MySQL: %v", err)
	}
	if err := ApplyPublicationMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := NewMySQLPublicationStore(db)
	publisher := NewPublisher(store, 5*time.Second)
	document, err := PreviewMapDocument()
	if err != nil {
		t.Fatal(err)
	}
	snapshotID := SnapshotID(testPublicationID(t, "integration-"))
	runID := testPublicationID(t, "run-")
	nextToken := testPublicationID(t, "baseline-")
	result := publisher.Execute(ctx, runID, func(_ context.Context, build PublicationBuildContext) (PublicationCandidate, error) {
		candidateMap := document
		candidateMap.SnapshotID = snapshotID
		return PublicationCandidate{
			RunID: runID, BaseSnapshotID: build.Baseline.SnapshotID,
			BaseRegistryBaselineToken: build.Baseline.RegistryBaselineToken,
			NextRegistryBaselineToken: nextToken, Profile: ProfilePreview,
			SchemaVersion: 1, Map: candidateMap,
		}, nil
	})
	if result.Kind != "PUBLISHED" || result.Receipt.SnapshotID != snapshotID {
		t.Fatalf("publish result = %#v", result)
	}
	replayed := publisher.Execute(ctx, runID, func(context.Context, PublicationBuildContext) (PublicationCandidate, error) {
		t.Fatal("published run should replay its receipt")
		return PublicationCandidate{}, nil
	})
	if replayed.Kind != "RECOVERED" || replayed.Receipt.SnapshotID != snapshotID {
		t.Fatalf("published run replay = %#v", replayed)
	}
	page, err := store.Read(ctx, ReadRequest{Kind: ReadHome})
	if err != nil || page.Home == nil || page.Home.SnapshotID != snapshotID {
		t.Fatalf("active home = %#v, %v", page, err)
	}
	team, err := store.Read(ctx, ReadRequest{Kind: ReadTeam, Slug: "toronto-maple-leafs"})
	if err != nil || team.Team == nil || team.Team.SnapshotID != snapshotID {
		t.Fatalf("active team = %#v, %v", team, err)
	}
	stagedID := SnapshotID(testPublicationID(t, "staged-"))
	stagedMap := document
	stagedMap.SnapshotID = stagedID
	staged := PublicationCandidate{
		RunID: testPublicationID(t, "stage-run-"), BaseSnapshotID: snapshotID,
		BaseRegistryBaselineToken: nextToken,
		NextRegistryBaselineToken: testPublicationID(t, "stage-token-"),
		Profile:                   ProfilePreview, SchemaVersion: 1, Map: stagedMap,
	}
	hash, err := candidateHash(staged)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.StageCandidate(ctx, staged, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(ctx, ReadRequest{Kind: ReadMap, SnapshotID: stagedID}); !isReadFault(err, FaultSnapshotMissing) {
		t.Fatalf("staged candidate became visible: %v", err)
	}
	casRun := testPublicationID(t, "cas-run-")
	casToken := strings.Repeat("d", 64)
	if lease, err := store.AcquireLease(ctx, casRun, casToken, 5*time.Second); err != nil || lease.Kind != "ACQUIRED" {
		t.Fatalf("CAS test lease = %#v, %v", lease, err)
	}
	staleMap := document
	staleMap.SnapshotID = SnapshotID(testPublicationID(t, "stale-"))
	stale := PublicationCandidate{RunID: casRun, BaseSnapshotID: "wrong-baseline",
		BaseRegistryBaselineToken: "wrong-token", NextRegistryBaselineToken: "unused-token",
		Profile: ProfilePreview, SchemaVersion: 1, Map: staleMap}
	staleHash, err := candidateHash(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.StageCandidate(ctx, stale, staleHash); err != nil {
		t.Fatal(err)
	}
	casOutcome, err := store.CommitCandidate(ctx, casRun, casToken, stale, staleHash)
	if err != nil || !casOutcome.BaseChanged || casOutcome.CurrentBaseline.SnapshotID != snapshotID {
		t.Fatalf("stale baseline CAS = %#v, %v", casOutcome, err)
	}
	if err := store.ReleaseLease(ctx, casRun, casToken); err != nil {
		t.Fatal(err)
	}
	failed := publisher.Execute(ctx, testPublicationID(t, "failed-"), func(context.Context, PublicationBuildContext) (PublicationCandidate, error) {
		return PublicationCandidate{}, errors.New("fixture source failed")
	})
	if failed.Kind != "NOT_PUBLISHED" {
		t.Fatalf("failed build = %#v", failed)
	}
	page, err = store.Read(ctx, ReadRequest{Kind: ReadHome})
	if err != nil || page.Home == nil || page.Home.SnapshotID != snapshotID {
		t.Fatalf("old pointer not retained: %#v, %v", page, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE publication_schema_gate SET paused = TRUE WHERE singleton_id = 1`); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), `UPDATE publication_schema_gate SET paused = FALSE WHERE singleton_id = 1`)
	pausedRun := testPublicationID(t, "paused-")
	paused := publisher.Execute(ctx, pausedRun, func(_ context.Context, build PublicationBuildContext) (PublicationCandidate, error) {
		pausedMap := document
		pausedMap.SnapshotID = SnapshotID(testPublicationID(t, "paused-candidate-"))
		return PublicationCandidate{RunID: pausedRun, BaseSnapshotID: build.Baseline.SnapshotID,
			BaseRegistryBaselineToken: build.Baseline.RegistryBaselineToken,
			NextRegistryBaselineToken: testPublicationID(t, "paused-token-"),
			Profile:                   ProfilePreview, SchemaVersion: 1, Map: pausedMap}, nil
	})
	if paused.Kind != "NOT_PUBLISHED" || !errors.Is(paused.Fault, ErrPublicationPaused) {
		t.Fatalf("paused gate = %#v", paused)
	}
	page, err = store.Read(ctx, ReadRequest{Kind: ReadHome})
	if err != nil || page.Home == nil || page.Home.SnapshotID != snapshotID {
		t.Fatalf("paused publication changed active pointer: %#v, %v", page, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE publication_schema_gate SET paused = FALSE WHERE singleton_id = 1`); err != nil {
		t.Fatal(err)
	}
	leaseToken := strings.Repeat("a", 64)
	if lease, err := store.AcquireLease(ctx, testPublicationID(t, "hold-"), leaseToken, 5*time.Second); err != nil || lease.Kind != "ACQUIRED" {
		t.Fatalf("lease acquisition = %#v, %v", lease, err)
	}
	blocked := publisher.Execute(ctx, testPublicationID(t, "blocked-"), func(context.Context, PublicationBuildContext) (PublicationCandidate, error) {
		t.Fatal("blocked publisher invoked candidate builder")
		return PublicationCandidate{}, nil
	})
	if blocked.Kind != "IN_PROGRESS" {
		t.Fatalf("concurrent execution = %#v", blocked)
	}
	if err := store.ReleaseLease(ctx, "wrong-owner", leaseToken); err != nil {
		t.Fatal(err)
	}
	if lease, err := store.AcquireLease(ctx, "still-blocked", strings.Repeat("b", 64), 5*time.Second); err != nil || lease.Kind != "IN_PROGRESS" {
		t.Fatalf("foreign release cleared lease: %#v, %v", lease, err)
	}
	var heldRun string
	if err := db.QueryRowContext(ctx, `SELECT run_id FROM publication_lease WHERE singleton_id = 1`).Scan(&heldRun); err != nil {
		t.Fatal(err)
	}
	if err := store.ReleaseLease(ctx, heldRun, leaseToken); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE publication_lease SET run_id = ?, token = ?,
		expires_at = DATE_SUB(UTC_TIMESTAMP(6), INTERVAL 1 SECOND) WHERE singleton_id = 1`,
		runID, strings.Repeat("c", 64)); err != nil {
		t.Fatal(err)
	}
	recovered := publisher.Execute(ctx, testPublicationID(t, "reconcile-"), func(context.Context, PublicationBuildContext) (PublicationCandidate, error) {
		t.Fatal("an expired lease with a committed receipt must not rebuild")
		return PublicationCandidate{}, nil
	})
	if recovered.Kind != "RECOVERED" || recovered.Receipt.RunID != runID || recovered.Receipt.SnapshotID != snapshotID {
		t.Fatalf("expired receipt recovery = %#v", recovered)
	}
	missingReceiptRun := testPublicationID(t, "expired-without-receipt-")
	if _, err := db.ExecContext(ctx, `UPDATE publication_lease SET run_id = ?, token = ?,
		expires_at = DATE_SUB(UTC_TIMESTAMP(6), INTERVAL 1 SECOND) WHERE singleton_id = 1`,
		missingReceiptRun, strings.Repeat("e", 64)); err != nil {
		t.Fatal(err)
	}
	newOwnerRun := testPublicationID(t, "new-owner-")
	newOwnerToken := strings.Repeat("f", 64)
	takeover, err := store.AcquireLease(ctx, newOwnerRun, newOwnerToken, 5*time.Second)
	if err != nil || takeover.Kind != "ACQUIRED" || takeover.SupersededRunID != missingReceiptRun ||
		takeover.Baseline.SnapshotID != snapshotID {
		t.Fatalf("expired lease takeover = %#v, %v", takeover, err)
	}
	if err := store.ReleaseLease(ctx, missingReceiptRun, strings.Repeat("e", 64)); err != nil {
		t.Fatal(err)
	}
	if blocked, err := store.AcquireLease(ctx, "late-old-owner", strings.Repeat("a", 64), 5*time.Second); err != nil || blocked.Kind != "IN_PROGRESS" {
		t.Fatalf("old owner released replacement lease: %#v, %v", blocked, err)
	}
	if err := store.ReleaseLease(ctx, newOwnerRun, newOwnerToken); err != nil {
		t.Fatal(err)
	}
}

func testPublicationID(t *testing.T, prefix string) string {
	t.Helper()
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	return prefix + hex.EncodeToString(data)
}

func isReadFault(err error, code ReadFaultCode) bool {
	var fault *ReadFault
	return errors.As(err, &fault) && fault.Code == code
}
