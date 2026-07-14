package store

import (
	"context"

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

// RecipeIngredientInput holds the writable fields of one recipe_ingredients
// row, as submitted from the recipe form.
type RecipeIngredientInput struct {
	IngredientID int64
	Quantity     *float64
	UnitID       *int64
	Note         *string
}

// RecipeWithRelationsInput bundles a recipe's own fields with its full
// ingredient list and tag set — both replaced wholesale on every save (see
// setRecipeIngredients/setRecipeTags), the same "submit the full desired
// state" approach as the ingredients page's tag editing.
type RecipeWithRelationsInput struct {
	RecipeInput
	Ingredients []RecipeIngredientInput
	TagIDs      []int64
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

// RecipeStatusFilter narrows what ListRecipesWithStatus returns. Empty
// slices / nil pointers mean "no filter on that dimension".
type RecipeStatusFilter struct {
	// TagIDs, if non-empty, restricts results to recipes that carry ALL of
	// these tags — matching CookableFilter's semantics.
	TagIDs []int64
}

// RecipeStatusRow is a recipe plus the "how close am I to cooking this?"
// data used by the /recipes page: how many of the recipe's ingredients are
// missing from the selected pantry, and the recipe's tag names for the row's
// chip strip.
//
// MissingCount == 0 means every ingredient the recipe needs is stocked
// (recipes with no listed ingredients are trivially "ready" — MissingCount
// stays 0).
type RecipeStatusRow struct {
	models.Recipe

	MissingCount int64    `db:"missing_count"`
	TagNames     []string `db:"tag_names"`
}

// ListRecipesWithStatus returns every recipe, each annotated with how many
// of its required ingredients are missing from the given pantry and the
// list of its tag names. Results are ordered by MissingCount ASC then name,
// which matches the /recipes page's default "by ingredient availability"
// sort so callers can re-group in Go without re-sorting.
//
// When pantryID is 0 the missing-count query is skipped and every row is
// treated as "no data" (MissingCount = number of the recipe's ingredients).
// The pantry-less case is only reached by the handler when there is no
// selected pantry, and it falls back to alphabetical sort anyway; the
// numeric value on each row is not shown.
func (s *Store) ListRecipesWithStatus(ctx context.Context, pantryID int64, f RecipeStatusFilter) ([]RecipeStatusRow, error) {
	var tagParam any
	if len(f.TagIDs) > 0 {
		tagParam = f.TagIDs
	}

	// $1 = pantry id, $2 = tag ids (nullable bigint[]).
	//
	// The missing-count subquery counts each recipe ingredient that is
	// NOT stocked in $1's pantry with quantity > 0. LEFT JOIN LATERAL is
	// used so recipes with no ingredients still return a row (with count 0).
	//
	// Tag names come from an ordered array_agg so the template can render
	// them without a second query.
	rows, err := s.pool.Query(ctx, `
		SELECT
		    r.id, r.name, r.description, r.instructions, r.source_url,
		    r.servings, r.prep_time_minutes, r.cook_time_minutes,
		    r.created_at, r.updated_at,
		    COALESCE(missing.cnt, 0)::bigint AS missing_count,
		    COALESCE(
		        (SELECT array_agg(t.name ORDER BY t.name)
		         FROM   recipe_tags rt
		         JOIN   tags t ON t.id = rt.tag_id
		         WHERE  rt.recipe_id = r.id),
		        ARRAY[]::text[]
		    ) AS tag_names
		FROM   recipes r
		LEFT JOIN LATERAL (
		    SELECT COUNT(*) AS cnt
		    FROM   recipe_ingredients ri
		    WHERE  ri.recipe_id = r.id
		      AND  NOT EXISTS (
		               SELECT 1
		               FROM   pantry_ingredients pi
		               WHERE  pi.pantry_id     = $1
		                 AND  pi.ingredient_id = ri.ingredient_id
		                 AND  pi.quantity      > 0
		           )
		) missing ON true
		WHERE
		    -- Optional AND tag filter: recipe must carry every requested tag.
		    ($2::bigint[] IS NULL OR (
		        SELECT COUNT(*)
		        FROM   recipe_tags rt
		        WHERE  rt.recipe_id = r.id
		          AND  rt.tag_id = ANY($2::bigint[])
		    ) = array_length($2::bigint[], 1))
		ORDER BY missing_count ASC, r.name`,
		pantryID, tagParam)
	if err != nil {
		return nil, mapError(err)
	}

	recipes, err := pgx.CollectRows(rows, pgx.RowToStructByName[RecipeStatusRow])
	if err != nil {
		return nil, mapError(err)
	}
	return recipes, nil
}

// GetRecipe returns a single recipe by ID.
func (s *Store) GetRecipe(ctx context.Context, id int64) (models.Recipe, error) {
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
func (s *Store) UpdateRecipe(ctx context.Context, id int64, in RecipeInput) (models.Recipe, error) {
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
func (s *Store) DeleteRecipe(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateRecipeWithRelations inserts a new recipe and attaches its ingredient
// list and tags, all in a single transaction.
func (s *Store) CreateRecipeWithRelations(ctx context.Context, in RecipeWithRelationsInput) (models.Recipe, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Recipe{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
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

	if err := setRecipeIngredients(ctx, tx, recipe.ID, in.Ingredients); err != nil {
		return models.Recipe{}, err
	}
	if err := setRecipeTags(ctx, tx, recipe.ID, in.TagIDs); err != nil {
		return models.Recipe{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Recipe{}, mapError(err)
	}
	return recipe, nil
}

// UpdateRecipeWithRelations overwrites an existing recipe and replaces its
// ingredient list and tags, all in a single transaction.
func (s *Store) UpdateRecipeWithRelations(ctx context.Context, id int64, in RecipeWithRelationsInput) (models.Recipe, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Recipe{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
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

	if err := setRecipeIngredients(ctx, tx, id, in.Ingredients); err != nil {
		return models.Recipe{}, err
	}
	if err := setRecipeTags(ctx, tx, id, in.TagIDs); err != nil {
		return models.Recipe{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Recipe{}, mapError(err)
	}
	return recipe, nil
}

// setRecipeIngredients replaces every recipe_ingredients row for recipeID
// with items. Called within a transaction from Create/UpdateRecipeWithRelations.
// Callers must ensure items contains no duplicate IngredientID (the
// junction's primary key is (recipe_id, ingredient_id)) — see
// parseRecipeForm, which rejects duplicates before this is reached.
func setRecipeIngredients(ctx context.Context, tx pgx.Tx, recipeID int64, items []RecipeIngredientInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM recipe_ingredients WHERE recipe_id = $1`, recipeID); err != nil {
		return mapError(err)
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit_id, note)
			VALUES ($1, $2, $3, $4, $5)`, recipeID, item.IngredientID, item.Quantity, item.UnitID, item.Note); err != nil {
			return mapError(err)
		}
	}
	return nil
}

// setRecipeTags replaces every recipe_tags row for recipeID with tagIDs.
// Called within a transaction from Create/UpdateRecipeWithRelations.
func setRecipeTags(ctx context.Context, tx pgx.Tx, recipeID int64, tagIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM recipe_tags WHERE recipe_id = $1`, recipeID); err != nil {
		return mapError(err)
	}
	for _, tagID := range tagIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO recipe_tags (recipe_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, recipeID, tagID); err != nil {
			return mapError(err)
		}
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

// RecipeIngredientDetailRow is one row of a recipe's ingredient list, joined
// with the ingredient/unit display names and flagged against a pantry's
// current stock. Used by the recipe detail page (GET /recipes/{id}).
type RecipeIngredientDetailRow struct {
	IngredientID   int64    `db:"ingredient_id"`
	IngredientName string   `db:"ingredient_name"`
	Quantity       *float64 `db:"quantity"`
	UnitName       *string  `db:"unit_name"`
	Note           *string  `db:"note"`
	// Missing is true when pantryID has none of this ingredient in stock
	// (quantity > 0). When pantryID is 0 (no pantry selected/exists) every
	// row comes back missing, which is correct: with no pantry there is,
	// by definition, nothing in stock anywhere.
	Missing bool `db:"missing"`
}

// ListRecipeIngredients returns every ingredient a recipe calls for, joined
// with the ingredient/unit names, ordered alphabetically by ingredient name.
// Each row is flagged Missing against pantryID's current stock so the
// detail page can float missing ingredients to the top in red.
func (s *Store) ListRecipeIngredients(ctx context.Context, recipeID, pantryID int64) ([]RecipeIngredientDetailRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
		    ri.ingredient_id AS ingredient_id,
		    i.name           AS ingredient_name,
		    ri.quantity      AS quantity,
		    u.name           AS unit_name,
		    ri.note          AS note,
		    NOT EXISTS (
		        SELECT 1
		        FROM   pantry_ingredients pi
		        WHERE  pi.pantry_id     = $2
		          AND  pi.ingredient_id = ri.ingredient_id
		          AND  pi.quantity      > 0
		    ) AS missing
		FROM   recipe_ingredients ri
		JOIN   ingredients i ON i.id = ri.ingredient_id
		LEFT JOIN units u    ON u.id = ri.unit_id
		WHERE  ri.recipe_id = $1
		ORDER  BY i.name`,
		recipeID, pantryID)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[RecipeIngredientDetailRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// ListRecipeTagNames returns the names of every tag attached to a recipe,
// alphabetically. Used by the recipe detail page; the recipes list page
// gets the same data more cheaply via ListRecipesWithStatus's array_agg.
func (s *Store) ListRecipeTagNames(ctx context.Context, recipeID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.name
		FROM   recipe_tags rt
		JOIN   tags t ON t.id = rt.tag_id
		WHERE  rt.recipe_id = $1
		ORDER  BY t.name`, recipeID)
	if err != nil {
		return nil, mapError(err)
	}

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, mapError(err)
	}
	return names, nil
}

// ListRecipeTagIDs returns the IDs of every tag attached to a recipe. Used
// by the recipe form (GET /recipes/{id}/edit) to pre-check the tag chips —
// ListRecipeTagNames (names only) is enough for the read-only detail page.
func (s *Store) ListRecipeTagIDs(ctx context.Context, recipeID int64) ([]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT tag_id FROM recipe_tags WHERE recipe_id = $1`, recipeID)
	if err != nil {
		return nil, mapError(err)
	}

	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, mapError(err)
	}
	return ids, nil
}

// RecipeIngredientEditRow is one row of a recipe's ingredient list as needed
// by the recipe form (GET /recipes/{id}/edit) — unlike
// RecipeIngredientDetailRow, this carries UnitID (to pre-select the row's
// unit dropdown) and has no pantry/Missing concept, since the form isn't
// checked against any particular pantry.
type RecipeIngredientEditRow struct {
	IngredientID   int64    `db:"ingredient_id"`
	IngredientName string   `db:"ingredient_name"`
	Quantity       *float64 `db:"quantity"`
	UnitID         *int64   `db:"unit_id"`
	UnitName       *string  `db:"unit_name"`
	Note           *string  `db:"note"`
}

// ListRecipeIngredientsForEdit returns every ingredient a recipe calls for,
// joined with the ingredient/unit names, ordered alphabetically by
// ingredient name (the same order autocomplete/tab-through would expect,
// matching ListRecipeIngredients — though this recipe form has no
// missing-first reordering, since there's no pantry context here).
func (s *Store) ListRecipeIngredientsForEdit(ctx context.Context, recipeID int64) ([]RecipeIngredientEditRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
		    ri.ingredient_id AS ingredient_id,
		    i.name           AS ingredient_name,
		    ri.quantity      AS quantity,
		    ri.unit_id       AS unit_id,
		    u.name           AS unit_name,
		    ri.note          AS note
		FROM   recipe_ingredients ri
		JOIN   ingredients i ON i.id = ri.ingredient_id
		LEFT JOIN units u    ON u.id = ri.unit_id
		WHERE  ri.recipe_id = $1
		ORDER  BY i.name`,
		recipeID)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[RecipeIngredientEditRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}
