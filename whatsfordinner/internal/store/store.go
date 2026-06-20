// Package store provides the data-access layer for the whatsfordinner database.
// Each file groups the CRUD operations for one resource; this file holds the
// shared Store type, sentinel errors and helpers.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors returned by the store. Handlers map these to HTTP statuses.
var (
	// ErrNotFound is returned when a requested row does not exist.
	ErrNotFound = errors.New("resource not found")
	// ErrConflict is returned on a unique-constraint violation.
	ErrConflict = errors.New("resource already exists")
	// ErrReference is returned on a foreign-key violation (referenced row missing
	// or still in use).
	ErrReference = errors.New("referenced resource does not exist or is still in use")
)

// PostgreSQL error codes we translate into sentinel errors.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// Default and maximum page sizes for list endpoints.
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// Store is the data-access layer backed by a pgx connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store using the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping verifies database connectivity (used by the readiness probe).
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// mapError converts driver-specific errors into the store's sentinel errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return ErrConflict
		case pgForeignKeyViolation:
			return ErrReference
		}
	}

	return err
}

// clampPage normalizes pagination parameters to safe bounds.
func clampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
