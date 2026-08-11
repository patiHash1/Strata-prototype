package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new connection pool from the provided DSN.
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	const maxRetries = 5
	const retryDelay = 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := pool.Ping(ctx); err != nil {
			if attempt == maxRetries {
				pool.Close()
				return nil, fmt.Errorf("ping database after %d attempts: %w", maxRetries, err)
			}
			time.Sleep(retryDelay)
			continue
		}
		return &DB{Pool: pool}, nil
	}

	pool.Close()
	return nil, fmt.Errorf("ping database: unreachable after %d attempts", maxRetries)
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// Migrate runs schema migrations idempotently — only applies migrations
// not yet recorded in the schema_migrations table.
func (db *DB) Migrate(ctx context.Context) error {
	// Ensure the tracking table exists.
	if _, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	for _, m := range migrations {
		// Skip the schema_migrations table creation itself — it was already handled above.
		if m.name == "create_schema_migrations" {
			continue
		}

		// Check if already applied.
		var alreadyApplied bool
		err := db.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			m.name,
		).Scan(&alreadyApplied)
		if err != nil {
			return fmt.Errorf("check migration %q: %w", m.name, err)
		}
		if alreadyApplied {
			continue
		}

		// Apply the migration.
		if _, err := db.Pool.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("migration %q: %w", m.name, err)
		}

		// Record it.
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES ($1, NOW())`,
			m.name,
		); err != nil {
			return fmt.Errorf("record migration %q: %w", m.name, err)
		}

		fmt.Printf("  ✓ %s\n", m.name)
	}

	return nil
}
