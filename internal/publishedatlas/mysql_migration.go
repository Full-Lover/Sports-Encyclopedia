package publishedatlas

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed migrations/0001_publication.up.sql
var publicationMigration0001 string

// ApplyPublicationMigrations is repeatable. MySQL DDL commits implicitly, so
// each statement is idempotent and the version marker is written last.
func ApplyPublicationMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("publication database is nil")
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS publication_migrations (
		version INT NOT NULL PRIMARY KEY, applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		return fmt.Errorf("create publication migration ledger: %w", err)
	}
	var version int
	err := db.QueryRowContext(ctx, `SELECT version FROM publication_migrations WHERE version = 1`).Scan(&version)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("read publication migration ledger: %w", err)
	}
	for _, raw := range strings.Split(publicationMigration0001, ";") {
		statement := strings.TrimSpace(raw)
		if statement == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply publication migration 0001: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO publication_migrations (version) VALUES (1)`); err != nil {
		return fmt.Errorf("record publication migration 0001: %w", err)
	}
	return nil
}
