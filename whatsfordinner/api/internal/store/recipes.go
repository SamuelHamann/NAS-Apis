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

// CookableFilter holds the optional constraints for ListCookableRecipes.
type CookableFilter struct {
	// TagIDs, if non-empty, restricts results to recipes that carry ALL of
	// these tags. An empty slice means "no tag filter".
	TagIDs []int64
	// MaxPrep, MaxCook and MaxTotal are inclusive upper bounds in minutes.
	// A nil pointer means "no constraint on that dimension".
	// Recipes with a NULL value in the corresponding column are excluded when
	// the filter is set (we can't make a time promise without data).
	MaxPrep  *int32 // prep_time_minutes <=
	MaxCook  *int32 // cook_time_minutes <=
	MaxTotal *int32 // prep_time_minutes + cook_time_minutes <=
}

// ListCookableRecipes returns recipes whose every required ingredient is
// present in the pantry (pantry_ingredients.quantity > 0). The optional
// CookableFilter fields narrow the result further. Results are ordered by name.
//
// The pantry check uses a double-NOT-EXISTS (relational division):
//
//	"there is no ingredient this recipe needs that is missing from the pantry"
//
// A recipe with no listed ingredients is considered trivially cookable.
func (s *Store) ListCookableRecipes(ctx context.Context, f CookableFilter, limit, offset int) ([]models.Recipe, error) {
	limit, offset = clampPage(limit, offset)

	// Send nil (→ SQL NULL) when no tags are requested so the conditional
	// inside the query short-circuits and skips the tag subquery entirely.
	var tagParam any
	if len(f.TagIDs) > 0 {
		tagParam = f.TagIDs
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+recipeColumns+`
		FROM   recipes r
		WHERE
		    -- Every ingredient the recipe needs is stocked in the pantry.
		    NOT EXISTS (
		        SELECT 1
		        FROM   recipe_ingredients ri
		        WHERE  ri.recipe_id = r.id
		          AND  NOT EXISTS (
		                   SELECT 1
		                   FROM   pantry_ingredients pi
		                   WHERE  pi.ingredient_id = ri.ingredient_id
		                     AND  pi.quantity > 0
		               )
		    )
		    -- Optional: recipe must carry ALL of the requested tags.
		    AND ($1::bigint[] IS NULL OR (
		            SELECT COUNT(*)
		            FROM   recipe_tags rt
		            WHERE  rt.recipe_id = r.id
		              AND  rt.tag_id = ANY($1::bigint[])
		        ) = array_length($1::bigint[], 1))
		    -- Optional: max prep time.
		    AND ($2::int IS NULL OR (
		            r.prep_time_minutes IS NOT NULL
		            AND r.prep_time_minutes <= $2
		        ))
		    -- Optional: max cook time.
		    AND ($3::int IS NULL OR (
		            r.cook_time_minutes IS NOT NULL
		            AND r.cook_time_minutes <= $3
		        ))
		    -- Optional: max total time (prep + cook combined).
		    AND ($4::int IS NULL OR (
		            r.prep_time_minutes IS NOT NULL
		            AND r.cook_time_minutes IS NOT NULL
		            AND r.prep_time_minutes + r.cook_time_minutes <= $4
		        ))
		ORDER  BY r.name
		LIMIT  $5 OFFSET $6`,
		tagParam, f.MaxPrep, f.MaxCook, f.MaxTotal, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	recipes, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Recipe])
	if err != nil {
		return nil, mapError(err)
	}
	return recipes, nil
}
