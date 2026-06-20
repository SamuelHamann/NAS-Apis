package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ListUnits returns a page of units ordered by name.
func (s *Store) ListUnits(ctx context.Context, limit, offset int) ([]models.Unit, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, created_at
		FROM units
		ORDER BY name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	units, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Unit])
	if err != nil {
		return nil, mapError(err)
	}
	return units, nil
}

// GetUnit returns a single unit by ID.
func (s *Store) GetUnit(ctx context.Context, id int64) (models.Unit, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, created_at FROM units WHERE id = $1`, id)
	if err != nil {
		return models.Unit{}, mapError(err)
	}

	unit, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Unit])
	if err != nil {
		return models.Unit{}, mapError(err)
	}
	return unit, nil
}

// CreateUnit inserts a new unit and returns the stored row.
func (s *Store) CreateUnit(ctx context.Context, name string) (models.Unit, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO units (name)
		VALUES ($1)
		RETURNING id, name, created_at`, name)
	if err != nil {
		return models.Unit{}, mapError(err)
	}

	unit, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Unit])
	if err != nil {
		return models.Unit{}, mapError(err)
	}
	return unit, nil
}

// UpdateUnit renames an existing unit and returns the stored row.
func (s *Store) UpdateUnit(ctx context.Context, id int64, name string) (models.Unit, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE units
		SET name = $2
		WHERE id = $1
		RETURNING id, name, created_at`, id, name)
	if err != nil {
		return models.Unit{}, mapError(err)
	}

	unit, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Unit])
	if err != nil {
		return models.Unit{}, mapError(err)
	}
	return unit, nil
}

// DeleteUnit removes a unit by ID.
func (s *Store) DeleteUnit(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM units WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
