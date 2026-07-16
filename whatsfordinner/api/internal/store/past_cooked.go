package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// PastCookedInput holds the writable fields of a past_cooked_recipes row.
type PastCookedInput struct {
	RecipeID     int64      // used in Create; not updated
	TimesCooked  int32      // must be >= 1
	LastCookedAt *time.Time // nil → database uses now()
}

const pastCookedColumns = `id, recipe_id, times_cooked, last_cooked_at`

// ListPastCooked returns a page of past-cooked entries, most recently cooked
// first.
func (s *Store) ListPastCooked(ctx context.Context, limit, offset int) ([]models.PastCookedRecipe, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT `+pastCookedColumns+`
		FROM   past_cooked_recipes
		ORDER  BY last_cooked_at DESC, id
		LIMIT  $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// GetPastCooked returns a single past-cooked entry by ID.
func (s *Store) GetPastCooked(ctx context.Context, id uuid.UUID) (models.PastCookedRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+pastCookedColumns+`
		FROM   past_cooked_recipes
		WHERE  id = $1`, id)
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}
	return item, nil
}

// GetPastCookedByRecipeID returns the past-cooked entry for a single recipe
// (recipe_id is unique in past_cooked_recipes — CookRecipe upserts on it),
// or ErrNotFound if the recipe has never been cooked.
func (s *Store) GetPastCookedByRecipeID(ctx context.Context, recipeID int64) (models.PastCookedRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+pastCookedColumns+`
		FROM   past_cooked_recipes
		WHERE  recipe_id = $1`, recipeID)
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}
	return item, nil
}

// CreatePastCooked inserts a new past-cooked entry and returns the stored row.
// A nil LastCookedAt defaults to the database's now().
func (s *Store) CreatePastCooked(ctx context.Context, in PastCookedInput) (models.PastCookedRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO past_cooked_recipes (recipe_id, times_cooked, last_cooked_at)
		VALUES ($1, $2, COALESCE($3, now()))
		RETURNING `+pastCookedColumns,
		in.RecipeID, in.TimesCooked, in.LastCookedAt)
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}
	return item, nil
}

// UpdatePastCooked replaces the mutable fields of an existing row.
// A nil LastCookedAt resets last_cooked_at to now().
func (s *Store) UpdatePastCooked(ctx context.Context, id uuid.UUID, in PastCookedInput) (models.PastCookedRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE past_cooked_recipes
		SET    times_cooked  = $2,
		       last_cooked_at = COALESCE($3, now())
		WHERE  id = $1
		RETURNING `+pastCookedColumns,
		id, in.TimesCooked, in.LastCookedAt)
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}

	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return models.PastCookedRecipe{}, mapError(err)
	}
	return item, nil
}

// DeletePastCooked removes a past-cooked entry by ID.
func (s *Store) DeletePastCooked(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM past_cooked_recipes WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListMostCooked returns past-cooked entries ordered by times_cooked descending
// (most cooked first), with last_cooked_at as a tie-breaker.
func (s *Store) ListMostCooked(ctx context.Context, limit, offset int) ([]models.PastCookedRecipe, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT `+pastCookedColumns+`
		FROM   past_cooked_recipes
		ORDER  BY times_cooked DESC, last_cooked_at DESC, id
		LIMIT  $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.PastCookedRecipe])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}
