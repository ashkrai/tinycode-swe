package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Connect opens a *sql.DB, configures the pool, and retries the ping up to 10
// times with a 2-second delay between attempts. This handles the race between
// the API container starting and Postgres finishing its own init.
func Connect(dsn string) (*sql.DB, error) {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)
	database.SetConnMaxIdleTime(1 * time.Minute)

	for i := 1; i <= 10; i++ {
		if err = database.Ping(); err == nil {
			log.Printf("database connected (attempt %d)", i)
			return database, nil
		}
		log.Printf("db ping attempt %d/10 failed: %v — retrying in 2s", i, err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("could not connect to database after 10 attempts: %w", err)
}

// Migrate runs idempotent DDL to create required tables.
func Migrate(db *sql.DB) error {
	const ddl = `
	CREATE TABLE IF NOT EXISTS users (
		id         SERIAL       PRIMARY KEY,
		name       TEXT         NOT NULL,
		email      TEXT         NOT NULL UNIQUE,
		created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
	`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("migration: %w", err)
	}
	log.Println("database migration ok")
	return nil
}
