package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// IngredientInput holds the writable fields of an ingredient, including its
// tag associations (see the ingredient_tags junction table).
type IngredientInput struct {
	Name   string
	TagIDs []int64
}

// IngredientFilter narrows a ListIngredientsDetailed query.
//
// TagIDs, when non-empty, requires the ingredient to carry at least one of
// the given tag IDs (OR semantics), mirroring PantryFilter/RecipeStatusFilter.
type IngredientFilter struct {
	TagIDs []int64
}

// IngredientDetailRow is one row from ListIngredientsDetailed: the
// ingredient plus the names of every tag attached to it via ingredient_tags.
type IngredientDetailRow struct {
	ID       int64    `db:"id"`
	Name     string   `db:"name"`
	TagNames []string `db:"tag_names"`
}

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

// ListIngredientsDetailed returns every ingredient ordered alphabetically,
// each with the names of every tag attached to it (via ingredient_tags),
// optionally narrowed by filter.TagIDs.
func (s *Store) ListIngredientsDetailed(ctx context.Context, filter IngredientFilter) ([]IngredientDetailRow, error) {
	// pgx passes a nil slice as SQL NULL; force an empty array so
	// cardinality(...) is 0 and the tag filter short-circuits cleanly.
	tagIDs := filter.TagIDs
	if tagIDs == nil {
		tagIDs = []int64{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			i.id   AS id,
			i.name AS name,
			COALESCE(
				array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
				ARRAY[]::text[]
			) AS tag_names
		FROM ingredients i
		LEFT JOIN ingredient_tags it ON it.ingredient_id = i.id
		LEFT JOIN tags t             ON t.id = it.tag_id
		WHERE (
		  cardinality($1::bigint[]) = 0
		  OR EXISTS (
		    SELECT 1 FROM ingredient_tags it2
		    WHERE it2.ingredient_id = i.id AND it2.tag_id = ANY($1::bigint[])
		  )
		)
		GROUP BY i.id
		ORDER BY i.name`, tagIDs)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[IngredientDetailRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// CreateIngredientWithTags inserts a new ingredient and attaches the given
// tags, all in a single transaction.
func (s *Store) CreateIngredientWithTags(ctx context.Context, in IngredientInput) (models.Ingredient, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		INSERT INTO ingredients (name)
		VALUES ($1)
		RETURNING id, name, created_at`, in.Name)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	ingredient, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}

	if err := setIngredientTags(ctx, tx, ingredient.ID, in.TagIDs); err != nil {
		return models.Ingredient{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Ingredient{}, mapError(err)
	}
	return ingredient, nil
}

// UpdateIngredientWithTags renames an existing ingredient and replaces its
// tag associations, all in a single transaction.
func (s *Store) UpdateIngredientWithTags(ctx context.Context, id int64, in IngredientInput) (models.Ingredient, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE ingredients
		SET name = $2
		WHERE id = $1
		RETURNING id, name, created_at`, id, in.Name)
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}
	ingredient, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Ingredient])
	if err != nil {
		return models.Ingredient{}, mapError(err)
	}

	if err := setIngredientTags(ctx, tx, id, in.TagIDs); err != nil {
		return models.Ingredient{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Ingredient{}, mapError(err)
	}
	return ingredient, nil
}

// setIngredientTags replaces every ingredient_tags row for ingredientID with
// tagIDs. Called within a transaction from Create/UpdateIngredientWithTags.
func setIngredientTags(ctx context.Context, tx pgx.Tx, ingredientID int64, tagIDs []int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM ingredient_tags WHERE ingredient_id = $1`, ingredientID); err != nil {
		return mapError(err)
	}
	for _, tagID := range tagIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ingredient_tags (ingredient_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, ingredientID, tagID); err != nil {
			return mapError(err)
		}
	}
	return nil
}
