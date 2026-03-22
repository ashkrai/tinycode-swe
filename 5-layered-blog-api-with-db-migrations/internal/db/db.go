package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// ConfigFromEnv reads DB config from environment variables with sensible defaults.
func ConfigFromEnv() Config {
	return Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Name:     getEnv("DB_NAME", "blog"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// Connect opens and verifies a database connection.
func Connect(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	log.Printf("connected to PostgreSQL at %s:%s/%s", cfg.Host, cfg.Port, cfg.Name)
	return db, nil
}

// MigrateUp applies all pending up-migrations.
func MigrateUp(db *sql.DB, migrationsPath string) error {
	return runMigration(db, migrationsPath, func(m *migrate.Migrate) error {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	})
}

// MigrateDown rolls back the last applied migration (one step).
func MigrateDown(db *sql.DB, migrationsPath string) error {
	return runMigration(db, migrationsPath, func(m *migrate.Migrate) error {
		return m.Steps(-1)
	})
}

// MigrateVersion returns the current schema version.
func MigrateVersion(db *sql.DB, migrationsPath string) (uint, bool, error) {
	var version uint
	var dirty bool
	err := runMigration(db, migrationsPath, func(m *migrate.Migrate) error {
		v, d, e := m.Version()
		version = v
		dirty = d
		return e
	})
	return version, dirty, err
}

func runMigration(db *sql.DB, migrationsPath string, fn func(*migrate.Migrate) error) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	return fn(m)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
