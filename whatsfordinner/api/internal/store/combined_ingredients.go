package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// CombinedIngredientItemInput holds the writable fields of one component
// ingredient within a combined ingredient.
type CombinedIngredientItemInput struct {
	IngredientID int64
	Quantity     *float64
	UnitID       *int64
	Note         *string
}

// CombinedIngredientInput holds the writable fields of a combined
// ingredient, including its component items (see the
// combined_ingredient_items junction table).
type CombinedIngredientInput struct {
	Name     string
	Quantity float64
	UnitID   *int64
	Note     *string
	Items    []CombinedIngredientItemInput
}

// CombinedIngredientItemDetailRow is one component ingredient within a
// CombinedIngredientDetailRow, joined against ingredients/units for display.
type CombinedIngredientItemDetailRow struct {
	IngredientID   int64    `db:"ingredient_id" json:"ingredient_id"`
	IngredientName string   `db:"ingredient_name" json:"ingredient_name"`
	Quantity       *float64 `db:"quantity" json:"quantity"`
	UnitID         *int64   `db:"unit_id" json:"unit_id"`
	UnitName       *string  `db:"unit_name" json:"unit_name"`
	Note           *string  `db:"note" json:"note"`
}

// CombinedIngredientDetailRow is one row from ListCombinedIngredientsDetailed:
// the combined ingredient plus every component item attached to it via
// combined_ingredient_items.
type CombinedIngredientDetailRow struct {
	ID       int64                             `db:"id"`
	Name     string                            `db:"name"`
	Quantity float64                           `db:"quantity"`
	UnitID   *int64                            `db:"unit_id"`
	UnitName *string                           `db:"unit_name"`
	Note     *string                           `db:"note"`
	Items    []CombinedIngredientItemDetailRow `db:"items"`
}

// ListCombinedIngredients returns a page of combined ingredients ordered by
// name.
func (s *Store) ListCombinedIngredients(ctx context.Context, limit, offset int) ([]models.CombinedIngredient, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, quantity, unit_id, note, created_at, updated_at
		FROM combined_ingredients
		ORDER BY name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	combined, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return nil, mapError(err)
	}
	return combined, nil
}

// GetCombinedIngredient returns a single combined ingredient by ID.
func (s *Store) GetCombinedIngredient(ctx context.Context, id int64) (models.CombinedIngredient, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, quantity, unit_id, note, created_at, updated_at
		FROM combined_ingredients WHERE id = $1`, id)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}

	combined, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	return combined, nil
}

// CreateCombinedIngredient inserts a new combined ingredient (its own
// quantity/unit/note only, no items) and returns the stored row.
func (s *Store) CreateCombinedIngredient(ctx context.Context, name string, quantity float64, unitID *int64, note *string) (models.CombinedIngredient, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO combined_ingredients (name, quantity, unit_id, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, quantity, unit_id, note, created_at, updated_at`, name, quantity, unitID, note)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}

	combined, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	return combined, nil
}

// UpdateCombinedIngredient updates a combined ingredient's own fields (name,
// quantity, unit, note) and returns the stored row.
func (s *Store) UpdateCombinedIngredient(ctx context.Context, id int64, name string, quantity float64, unitID *int64, note *string) (models.CombinedIngredient, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE combined_ingredients
		SET name = $2, quantity = $3, unit_id = $4, note = $5, updated_at = now()
		WHERE id = $1
		RETURNING id, name, quantity, unit_id, note, created_at, updated_at`, id, name, quantity, unitID, note)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}

	combined, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	return combined, nil
}

// DeleteCombinedIngredient removes a combined ingredient by ID. Its
// component items are removed automatically (combined_ingredient_items.
// combined_ingredient_id is ON DELETE CASCADE).
func (s *Store) DeleteCombinedIngredient(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM combined_ingredients WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListCombinedIngredientsDetailed returns every combined ingredient ordered
// alphabetically, each with its component items (ingredient name, quantity,
// unit name, note).
func (s *Store) ListCombinedIngredientsDetailed(ctx context.Context) ([]CombinedIngredientDetailRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			ci.id       AS id,
			ci.name     AS name,
			ci.quantity AS quantity,
			ci.unit_id  AS unit_id,
			u.name      AS unit_name,
			ci.note     AS note,
			COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'ingredient_id',   cii.ingredient_id,
					'ingredient_name', i.name,
					'quantity',        cii.quantity,
					'unit_id',         cii.unit_id,
					'unit_name',       iu.name,
					'note',            cii.note
				) ORDER BY i.name)
				FROM combined_ingredient_items cii
				JOIN ingredients i ON i.id = cii.ingredient_id
				LEFT JOIN units iu ON iu.id = cii.unit_id
				WHERE cii.combined_ingredient_id = ci.id
			), '[]'::jsonb) AS items
		FROM combined_ingredients ci
		LEFT JOIN units u ON u.id = ci.unit_id
		ORDER BY ci.name`)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[CombinedIngredientDetailRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// CreateCombinedIngredientWithItems inserts a new combined ingredient and
// attaches its component items, all in a single transaction.
func (s *Store) CreateCombinedIngredientWithItems(ctx context.Context, in CombinedIngredientInput) (models.CombinedIngredient, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		INSERT INTO combined_ingredients (name, quantity, unit_id, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, quantity, unit_id, note, created_at, updated_at`, in.Name, in.Quantity, in.UnitID, in.Note)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	combined, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}

	if err := setCombinedIngredientItems(ctx, tx, combined.ID, in.Items); err != nil {
		return models.CombinedIngredient{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	return combined, nil
}

// UpdateCombinedIngredientWithItems updates an existing combined ingredient
// and replaces its component items, all in a single transaction.
func (s *Store) UpdateCombinedIngredientWithItems(ctx context.Context, id int64, in CombinedIngredientInput) (models.CombinedIngredient, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE combined_ingredients
		SET name = $2, quantity = $3, unit_id = $4, note = $5, updated_at = now()
		WHERE id = $1
		RETURNING id, name, quantity, unit_id, note, created_at, updated_at`, id, in.Name, in.Quantity, in.UnitID, in.Note)
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	combined, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.CombinedIngredient])
	if err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}

	if err := setCombinedIngredientItems(ctx, tx, id, in.Items); err != nil {
		return models.CombinedIngredient{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.CombinedIngredient{}, mapError(err)
	}
	return combined, nil
}

// setCombinedIngredientItems replaces every combined_ingredient_items row
// for combinedIngredientID with items. Called within a transaction from
// Create/UpdateCombinedIngredientWithItems. Callers must ensure items
// contains no duplicate IngredientID (the junction's primary key is
// (combined_ingredient_id, ingredient_id)) — see
// parseCombinedIngredientForm, which rejects duplicates before this is
// reached so a raw unique-violation here is never conflated with the
// parent's own name-uniqueness conflict.
func setCombinedIngredientItems(ctx context.Context, tx pgx.Tx, combinedIngredientID int64, items []CombinedIngredientItemInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM combined_ingredient_items WHERE combined_ingredient_id = $1`, combinedIngredientID); err != nil {
		return mapError(err)
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO combined_ingredient_items (combined_ingredient_id, ingredient_id, quantity, unit_id, note)
			VALUES ($1, $2, $3, $4, $5)`, combinedIngredientID, item.IngredientID, item.Quantity, item.UnitID, item.Note); err != nil {
			return mapError(err)
		}
	}
	return nil
}
