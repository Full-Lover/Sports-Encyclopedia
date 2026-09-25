package atlasregistry

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

func openRegistryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("SPORTS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set SPORTS_TEST_MYSQL_DSN for the dedicated MySQL test database")
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil || !strings.Contains(strings.ToLower(config.DBName), "test") {
		t.Fatal("test DSN must name a dedicated database containing 'test'")
	}
	config.ParseTime = true
	config.Loc = time.UTC
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect to registry test database: %v", err)
	}
	return db
}

func TestApplyRegistryMigrations(t *testing.T) {
	db := openRegistryTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyRegistryMigrations(ctx, db); err != nil {
		t.Fatalf("migration replay: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name LIKE 'registry_%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 9 {
		t.Fatalf("registry tables = %d, want at least 9", count)
	}
}
