package atlasregistry

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed migrations/0001_registry.up.sql
var registryMigration0001 string

// ApplyRegistryMigrations adds only AtlasRegistry-owned tables. MySQL DDL
// commits implicitly; each statement can be retried before the ledger is set.
func ApplyRegistryMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("registry database is nil")
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS registry_migrations (
		version INT NOT NULL PRIMARY KEY, applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("create registry migration ledger: %w", err)
	}
	var version int
	err := db.QueryRowContext(ctx, `SELECT version FROM registry_migrations WHERE version = 1`).Scan(&version)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read registry migration ledger: %w", err)
	}
	for _, raw := range strings.Split(registryMigration0001, ";") {
		statement := strings.TrimSpace(raw)
		if statement == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply registry migration 0001: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO registry_migrations (version) VALUES (1)`); err != nil {
		return fmt.Errorf("record registry migration 0001: %w", err)
	}
	return nil
}
