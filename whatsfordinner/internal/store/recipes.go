package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// RecipeInput holds the writable fields of a recipe.
type RecipeInput struct {
	Name            string
	Description     *string
	Instructions    *string
	SourceURL       *string
	Servings        *int32
	PrepTimeMinutes *int32
	CookTimeMinutes *int32
}

const recipeColumns = `id, name, description, instructions, source_url,
	servings, prep_time_minutes, cook_time_minutes, created_at, updated_at`

// ListRecipes returns a page of recipes, newest first.
func (s *Store) ListRecipes(ctx context.Context, limit, offset int) ([]models.Recipe, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT `+recipeColumns+`
		FROM recipes
		ORDER BY created_at DESC, id
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	recipes, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Recipe])
	if err != nil {
		return nil, mapError(err)
	}
	return recipes, nil
}

// GetRecipe returns a single recipe by ID.
func (s *Store) GetRecipe(ctx context.Context, id uuid.UUID) (models.Recipe, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+recipeColumns+` FROM recipes WHERE id = $1`, id)
	if err != nil {
		return models.Recipe{}, mapError(err)
	}

	recipe, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Recipe])
	if err != nil {
		return models.Recipe{}, mapError(err)
	}
	return recipe, nil
}

// CreateRecipe inserts a new recipe and returns the stored row.
func (s *Store) CreateRecipe(ctx context.Context, in RecipeInput) (models.Recipe, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO recipes
			(name, description, instructions, source_url, servings, prep_time_minutes, cook_time_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+recipeColumns,
		in.Name, in.Description, in.Instructions, in.SourceURL,
		in.Servings, in.PrepTimeMinutes, in.CookTimeMinutes)
	if err != nil {
		return models.Recipe{}, mapError(err)
	}

	recipe, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Recipe])
	if err != nil {
		return models.Recipe{}, mapError(err)
	}
	return recipe, nil
}

// UpdateRecipe overwrites an existing recipe and returns the stored row.
func (s *Store) UpdateRecipe(ctx context.Context, id uuid.UUID, in RecipeInput) (models.Recipe, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE recipes
		SET name = $2, description = $3, instructions = $4, source_url = $5,
		    servings = $6, prep_time_minutes = $7, cook_time_minutes = $8,
		    updated_at = now()
		WHERE id = $1
		RETURNING `+recipeColumns,
		id, in.Name, in.Description, in.Instructions, in.SourceURL,
		in.Servings, in.PrepTimeMinutes, in.CookTimeMinutes)
	if err != nil {
		return models.Recipe{}, mapError(err)
	}

	recipe, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Recipe])
	if err != nil {
		return models.Recipe{}, mapError(err)
	}
	return recipe, nil
}

// DeleteRecipe removes a recipe by ID.
func (s *Store) DeleteRecipe(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
