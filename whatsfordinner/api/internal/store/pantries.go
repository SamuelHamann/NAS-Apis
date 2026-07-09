package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// pantryColumnsAll is the full column list for the pantry table. Kept as a
// constant so every helper stays in sync with the models.Pantry struct
// (pgx.RowToStructByName panics if a field has no matching column).
const pantryColumnsAll = `id, name, created_at, updated_at`

// ListPantries returns every pantry ordered by name. There is no pagination:
// the list is expected to stay small (a handful of physical locations —
// kitchen, freezer, cabin, ...) and doubles as the pantry-picker dropdown
// on the home page.
//
// Use ListPantriesForUser when you want only the pantries a specific user
// has been granted access to via the user_pantry join table.
func (s *Store) ListPantries(ctx context.Context) ([]models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+pantryColumnsAll+`
		FROM pantry
		ORDER BY name`)
	if err != nil {
		return nil, mapError(err)
	}

	pantries, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return nil, mapError(err)
	}
	return pantries, nil
}

// ListPantriesForUser returns the pantries linked to userID via the
// user_pantry join table, ordered by name. Returns an empty slice (not an
// error) when the user has no memberships yet — callers typically fall back
// to ListPantries in that case so a fresh install isn't a dead end.
func (s *Store) ListPantriesForUser(ctx context.Context, userID int64) ([]models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.name, p.created_at, p.updated_at
		FROM pantry p
		JOIN user_pantry up ON up.id_pantry = p.id
		WHERE up.id_user = $1
		ORDER BY p.name`, userID)
	if err != nil {
		return nil, mapError(err)
	}

	pantries, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return nil, mapError(err)
	}
	return pantries, nil
}

// GetPantry returns a single pantry by ID.
func (s *Store) GetPantry(ctx context.Context, id int64) (models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+pantryColumnsAll+`
		FROM pantry
		WHERE id = $1`, id)
	if err != nil {
		return models.Pantry{}, mapError(err)
	}

	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return models.Pantry{}, mapError(err)
	}
	return p, nil
}

// GetDefaultPantry returns the first pantry (by id). It is used as the
// "currently selected pantry" placeholder while there is no multi-pantry
// picker on the home page yet; once a picker exists the choice will be
// stored in the session and this method will only be used as a fallback.
func (s *Store) GetDefaultPantry(ctx context.Context) (models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+pantryColumnsAll+`
		FROM pantry
		ORDER BY id
		LIMIT 1`)
	if err != nil {
		return models.Pantry{}, mapError(err)
	}

	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return models.Pantry{}, mapError(err)
	}
	return p, nil
}
