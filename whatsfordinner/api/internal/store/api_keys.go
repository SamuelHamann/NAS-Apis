package store

import (
	"context"

	"github.com/google/uuid"
)

// APIKeyExists reports whether the given UUID exists in the api_keys table.
// It uses QueryRow + Scan directly since we only need a single boolean scalar.
//
// TODO: add a short-lived in-memory cache here if per-request DB latency
// becomes noticeable (most home NAS workloads will be fine without it).
func (s *Store) APIKeyExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM api_keys WHERE id = $1)`, id,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
