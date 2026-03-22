package db

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Config holds PostgreSQL connection and pool parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns production-ready defaults.
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		DBName:          "taskdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}
}

// New opens a *sqlx.DB and configures the connection pool.
func New(cfg Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	database, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db.New: %w", err)
	}

	// Cap total (in-use + idle) connections.
	database.SetMaxOpenConns(cfg.MaxOpenConns)
	// Keep idle connections warm; extras are closed immediately.
	database.SetMaxIdleConns(cfg.MaxIdleConns)
	// Recycle connections — avoids stale handles dropped by firewalls / RDS.
	database.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	// Evict idle connections unused longer than this.
	database.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return database, nil
}
