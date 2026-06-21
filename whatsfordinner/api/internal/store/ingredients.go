package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ListIngredients returns a page of ingredients ordered by name.
func (s *Store) ListIngredients(ctx context.Context, limit, offset int) ([]models.Ingredient, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, created_at
		FROM ingredients
		ORDER BY name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	ingredients, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return nil, mapError(err)
	}
	return ingredients, nil
}

// GetIngredient returns a single ingredient by ID.
func (s *Store) GetIngredient(ctx context.Context, id int64) (models.Ingredient, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, created_at FROM ingredients WHERE id = $1`, id)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}

	ingredient, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	return ingredient, nil
}

// CreateIngredient inserts a new ingredient and returns the stored row.
func (s *Store) CreateIngredient(ctx context.Context, name string) (models.Ingredient, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO ingredients (name)
		VALUES ($1)
		RETURNING id, name, created_at`, name)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}

	ingredient, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	return ingredient, nil
}

// UpdateIngredient renames an existing ingredient and returns the stored row.
func (s *Store) UpdateIngredient(ctx context.Context, id int64, name string) (models.Ingredient, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE ingredients
		SET name = $2
		WHERE id = $1
		RETURNING id, name, created_at`, id, name)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}

	ingredient, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	return ingredient, nil
}

// DeleteIngredient removes an ingredient by ID.
func (s *Store) DeleteIngredient(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ingredients WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
