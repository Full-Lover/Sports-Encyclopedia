package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
	"github.com/go-sql-driver/mysql"
)

func openPublicationDB() (*sql.DB, error) {
	dsn := os.Getenv("SPORTS_DB_DSN")
	if dsn == "" {
		return nil, errors.New("SPORTS_DB_DSN is required for database publication")
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("invalid SPORTS_DB_DSN")
	}
	config.ParseTime = true
	config.Loc = time.UTC
	config.Timeout = 5 * time.Second
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open publication database: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to publication database: %w", err)
	}
	return db, nil
}

func runPublicationCommand(command string) error {
	db, err := openPublicationDB()
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if command == "migrate" {
		return publishedatlas.ApplyPublicationMigrations(ctx, db)
	}
	document, err := publishedatlas.PreviewMapDocument()
	if err != nil {
		return err
	}
	store := publishedatlas.NewMySQLPublicationStore(db)
	runID, err := randomPublicationID("fixture-run-")
	if err != nil {
		return err
	}
	result := publishedatlas.NewPublisher(store, 9*time.Second).Execute(ctx, runID,
		func(_ context.Context, build publishedatlas.PublicationBuildContext) (publishedatlas.PublicationCandidate, error) {
			snapshotID, err := randomPublicationID("preview-")
			if err != nil {
				return publishedatlas.PublicationCandidate{}, err
			}
			registryToken, err := randomPublicationID("fixture-baseline-")
			if err != nil {
				return publishedatlas.PublicationCandidate{}, err
			}
			candidateMap := document
			candidateMap.SnapshotID = publishedatlas.SnapshotID(snapshotID)
			return publishedatlas.PublicationCandidate{
				RunID: runID, BaseSnapshotID: build.Baseline.SnapshotID,
				BaseRegistryBaselineToken: build.Baseline.RegistryBaselineToken,
				NextRegistryBaselineToken: registryToken, Profile: publishedatlas.ProfilePreview,
				SchemaVersion: 1, Map: candidateMap,
			}, nil
		})
	if result.Kind != "PUBLISHED" {
		if result.Fault == nil {
			return fmt.Errorf("preview was not published: %s", result.Kind)
		}
		return fmt.Errorf("preview was not published: %s: %w", result.Kind, result.Fault)
	}
	log.Printf("published preview snapshot %s", result.Receipt.SnapshotID)
	return nil
}

func randomPublicationID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}
