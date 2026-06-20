package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// PantryInput holds the writable fields of a pantry stock entry.
type PantryInput struct {
	IngredientID int64
	Quantity     float64
	UnitID       int64
	Note         *string
	IsQuantified bool
}

const pantryColumns = `id, ingredient_id, quantity, unit_id, note, is_quantified, updated_at`

// ListPantry returns a page of pantry entries.
func (s *Store) ListPantry(ctx context.Context, limit, offset int) ([]models.PantryIngredient, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT `+pantryColumns+`
		FROM pantry_ingredients
		ORDER BY updated_at DESC, id
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.PantryIngredient])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// GetPantryItem returns a single pantry entry by ID.
func (s *Store) GetPantryItem(ctx context.Context, id uuid.UUID) (models.PantryIngredient, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+pantryColumns+` FROM pantry_ingredients WHERE id = $1`, id)
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PantryIngredient])
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}
	return item, nil
}

// CreatePantryItem inserts a new pantry entry and returns the stored row.
func (s *Store) CreatePantryItem(ctx context.Context, in PantryInput) (models.PantryIngredient, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO pantry_ingredients (ingredient_id, quantity, unit_id, note, is_quantified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+pantryColumns,
		in.IngredientID, in.Quantity, in.UnitID, in.Note, in.IsQuantified)
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PantryIngredient])
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}
	return item, nil
}

// UpdatePantryItem overwrites an existing pantry entry and returns the stored row.
func (s *Store) UpdatePantryItem(ctx context.Context, id uuid.UUID, in PantryInput) (models.PantryIngredient, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE pantry_ingredients
		SET ingredient_id = $2, quantity = $3, unit_id = $4, note = $5,
		    is_quantified = $6, updated_at = now()
		WHERE id = $1
		RETURNING `+pantryColumns,
		id, in.IngredientID, in.Quantity, in.UnitID, in.Note, in.IsQuantified)
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PantryIngredient])
	if err != nil {
		return models.PantryIngredient{}, mapError(err)
	}
	return item, nil
}

// DeletePantryItem removes a pantry entry by ID.
func (s *Store) DeletePantryItem(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pantry_ingredients WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
