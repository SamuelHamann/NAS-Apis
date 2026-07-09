package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// PantryInput holds the writable fields of a pantry stock entry. PantryID is
// which pantry the entry belongs to (fridge, freezer, ...); ExpirationDate
// is optional and drives the color-coded "expired / expiring soon / other"
// grouping on the pantry page.
type PantryInput struct {
	PantryID       int64
	IngredientID   int64
	Quantity       float64
	UnitID         int64
	Note           *string
	IsQuantified   bool
	LocationID     *int64     // optional — which food_location stores this item
	ExpirationDate *time.Time // optional — nil for "no expiration on record"
}

const pantryColumns = `id, pantry_id, ingredient_id, quantity, unit_id, note, is_quantified, location_id, expiration_date, updated_at`

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
		INSERT INTO pantry_ingredients (
			pantry_id, ingredient_id, quantity, unit_id, note,
			is_quantified, location_id, expiration_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+pantryColumns,
		in.PantryID, in.IngredientID, in.Quantity, in.UnitID, in.Note,
		in.IsQuantified, in.LocationID, in.ExpirationDate)
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
		SET pantry_id       = $2,
		    ingredient_id   = $3,
		    quantity        = $4,
		    unit_id         = $5,
		    note            = $6,
		    is_quantified   = $7,
		    location_id     = $8,
		    expiration_date = $9,
		    updated_at      = now()
		WHERE id = $1
		RETURNING `+pantryColumns,
		id, in.PantryID, in.IngredientID, in.Quantity, in.UnitID, in.Note,
		in.IsQuantified, in.LocationID, in.ExpirationDate)
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

// PantryFilter narrows a ListPantryDetailed query.
//
// TagIDs, when non-empty, requires the pantry item's ingredient to appear in
// at least one recipe carrying any of the given tag IDs (OR semantics). This
// piggy-backs on the existing recipes/recipe_ingredients/recipe_tags join
// rather than adding a pantry-side tag table.
type PantryFilter struct {
	TagIDs []int64
}

// PantryDetailedRow is one row from ListPantryDetailed. It is the pantry
// entry plus the joined display names it needs to render on the pantry page
// without further round-trips.
type PantryDetailedRow struct {
	ID             uuid.UUID  `db:"id"`
	PantryID       int64      `db:"pantry_id"`
	IngredientID   int64      `db:"ingredient_id"`
	IngredientName string     `db:"ingredient_name"`
	Quantity       float64    `db:"quantity"`
	UnitID         int64      `db:"unit_id"`
	UnitName       string     `db:"unit_name"`
	LocationID     *int64     `db:"location_id"`
	LocationName   *string    `db:"location_name"`
	Note           *string    `db:"note"`
	IsQuantified   bool       `db:"is_quantified"`
	ExpirationDate *time.Time `db:"expiration_date"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// ListPantryDetailed returns every entry of a given pantry, joined with the
// ingredient / unit / location names needed to render the pantry page. Rows
// come back ordered by (expiration_date NULLS LAST, ingredient name) — the
// handler regroups them per sort mode, so this ordering is just a sane
// starting point and also the effective secondary sort for other modes.
func (s *Store) ListPantryDetailed(ctx context.Context, pantryID int64, filter PantryFilter) ([]PantryDetailedRow, error) {
	// pgx passes a nil slice as SQL NULL; force an empty array so
	// cardinality(...) is 0 and the tag filter short-circuits cleanly.
	tagIDs := filter.TagIDs
	if tagIDs == nil {
		tagIDs = []int64{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			p.id                  AS id,
			p.pantry_id           AS pantry_id,
			p.ingredient_id       AS ingredient_id,
			i.name                AS ingredient_name,
			p.quantity            AS quantity,
			p.unit_id             AS unit_id,
			u.name                AS unit_name,
			p.location_id         AS location_id,
			fl.name               AS location_name,
			p.note                AS note,
			p.is_quantified       AS is_quantified,
			p.expiration_date     AS expiration_date,
			p.updated_at          AS updated_at
		FROM pantry_ingredients p
		JOIN ingredients i     ON i.id  = p.ingredient_id
		JOIN units u           ON u.id  = p.unit_id
		LEFT JOIN food_locations fl ON fl.id = p.location_id
		WHERE p.pantry_id = $1
		  AND (
		    cardinality($2::bigint[]) = 0
		    OR EXISTS (
		      SELECT 1
		      FROM recipe_ingredients ri
		      JOIN recipe_tags rt ON rt.recipe_id = ri.recipe_id
		      WHERE ri.ingredient_id = p.ingredient_id
		        AND rt.tag_id = ANY($2::bigint[])
		    )
		  )
		ORDER BY p.expiration_date NULLS LAST, i.name`,
		pantryID, tagIDs)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[PantryDetailedRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}
