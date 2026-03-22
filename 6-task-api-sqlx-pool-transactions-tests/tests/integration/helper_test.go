package integration_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ashkrai/taskapi/internal/db"
)

// testDB opens a *sqlx.DB pointed at the integration-test Postgres instance.
// The test is skipped when the DB is unreachable.
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()

	cfg := db.Config{
		Host:            envOr("TEST_DB_HOST", "localhost"),
		Port:            dbPort(),
		User:            envOr("TEST_DB_USER", "postgres"),
		Password:        envOr("TEST_DB_PASSWORD", "postgres"),
		DBName:          envOr("TEST_DB_NAME", "taskdb_test"),
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: 30 * time.Second,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Skipf("skipping integration test — cannot reach test DB: %v", err)
	}

	applySchema(t, database)
	truncate(t, database)

	t.Cleanup(func() {
		truncate(t, database)
		database.Close()
	})

	return database
}

func applySchema(t *testing.T, database *sqlx.DB) {
	t.Helper()
	_, err := database.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS tasks (
			id          TEXT        PRIMARY KEY,
			title       TEXT        NOT NULL,
			description TEXT        NOT NULL DEFAULT '',
			status      TEXT        NOT NULL DEFAULT 'pending'
			            CHECK (status IN ('pending','in_progress','done')),
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at  TIMESTAMPTZ
		)`)
	if err != nil {
		t.Fatalf("applySchema: %v", err)
	}
}

func truncate(t *testing.T, database *sqlx.DB) {
	t.Helper()
	if _, err := database.ExecContext(context.Background(), "TRUNCATE TABLE tasks"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func dbPort() int {
	if v := os.Getenv("TEST_DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 5433
}
