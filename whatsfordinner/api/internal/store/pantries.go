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

// CreatePantry inserts a new pantry and returns the stored row.
func (s *Store) CreatePantry(ctx context.Context, name string) (models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO pantry (name)
		VALUES ($1)
		RETURNING `+pantryColumnsAll, name)
	if err != nil {
		return models.Pantry{}, mapError(err)
	}

	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return models.Pantry{}, mapError(err)
	}
	return p, nil
}

// UpdatePantry renames an existing pantry and returns the stored row.
func (s *Store) UpdatePantry(ctx context.Context, id int64, name string) (models.Pantry, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE pantry
		SET name = $2, updated_at = now()
		WHERE id = $1
		RETURNING `+pantryColumnsAll, id, name)
	if err != nil {
		return models.Pantry{}, mapError(err)
	}

	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Pantry])
	if err != nil {
		return models.Pantry{}, mapError(err)
	}
	return p, nil
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

// UserPantryAccess is one row of the user_pantry join table: userID has been
// granted access to pantryID.
type UserPantryAccess struct {
	UserID   int64
	PantryID int64
}

// ListUserPantryAccess returns every user_pantry row, for building the
// admin page's user/pantry access matrix.
func (s *Store) ListUserPantryAccess(ctx context.Context) ([]UserPantryAccess, error) {
	rows, err := s.pool.Query(ctx, `SELECT id_user, id_pantry FROM user_pantry`)
	if err != nil {
		return nil, mapError(err)
	}

	access, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (UserPantryAccess, error) {
		var a UserPantryAccess
		err := row.Scan(&a.UserID, &a.PantryID)
		return a, err
	})
	if err != nil {
		return nil, mapError(err)
	}
	return access, nil
}

// SetUserPantryAccess replaces every user_pantry row with access (duplicate
// pairs are the caller's responsibility to have already collapsed — there's
// no unique constraint on (id_pantry, id_user) to lean on here), in a single
// transaction — the admin page always submits the full desired state of the
// matrix rather than one change at a time.
func (s *Store) SetUserPantryAccess(ctx context.Context, access []UserPantryAccess) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return mapError(err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM user_pantry`); err != nil {
		return mapError(err)
	}
	for _, a := range access {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_pantry (id_pantry, id_user)
			VALUES ($1, $2)`, a.PantryID, a.UserID); err != nil {
			return mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapError(err)
	}
	return nil
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
