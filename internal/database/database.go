package database

import (
	"context"
	"fmt"
)

// DB is a placeholder database handle.
// Replace with your actual driver (pgx, database/sql, etc.) when ready.
type DB struct {
	DSN string
}

// New creates a new DB placeholder.
func New(dsn string) *DB {
	return &DB{DSN: dsn}
}

// Ping checks connectivity. Stub implementation.
func (db *DB) Ping(ctx context.Context) error {
	// TODO: implement real ping once a driver is wired in.
	_ = ctx
	return fmt.Errorf("database not yet configured")
}

// Close tears down the connection. Stub implementation.
func (db *DB) Close() error {
	// TODO: implement real close.
	return nil
}
