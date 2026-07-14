package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// PendingPantryItemInput holds the writable fields for a receipt-scanned
// item awaiting review. Unit is free text (e.g. from a Gemini receipt scan)
// and is resolved/created against the units table by CreatePendingPantryItem.
type PendingPantryItemInput struct {
	UPC      string
	Name     string
	Quantity float64
	Unit     string
	Price    *float64
	// Status is one of pending_pantry_items' allowed status values
	// ("pending", "processing", "approved", "rejected").
	Status string
}

// CreatePendingPantryItem inserts a receipt-scanned item, resolving Unit to
// a units.id (creating the unit if no existing one matches by name) in the
// same transaction.
func (s *Store) CreatePendingPantryItem(ctx context.Context, in PendingPantryItemInput) (models.PendingPantryItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.PendingPantryItem{}, mapError(err)
	}
	defer tx.Rollback(ctx)

	unitID, err := findOrCreateUnit(ctx, tx, in.Unit)
	if err != nil {
		return models.PendingPantryItem{}, err
	}

	rows, err := tx.Query(ctx, `
		INSERT INTO pending_pantry_items (upc, name, quantity, unit_id, price, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, upc, name, quantity, unit_id, price, status, created_at, updated_at`,
		in.UPC, in.Name, in.Quantity, unitID, in.Price, in.Status)
	if err != nil {
		return models.PendingPantryItem{}, mapError(err)
	}
	item, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.PendingPantryItem])
	if err != nil {
		return models.PendingPantryItem{}, mapError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PendingPantryItem{}, mapError(err)
	}
	return item, nil
}

// PendingPantryItemFilter narrows a ListPendingPantryItems query. An empty
// Statuses means "no filter" (every status returned).
type PendingPantryItemFilter struct {
	Statuses []string
}

// PendingPantryItemRow is one row from ListPendingPantryItems: the pending
// item plus its unit's display name (nil when UnitID is nil).
type PendingPantryItemRow struct {
	models.PendingPantryItem

	UnitName *string `db:"unit_name"`
}

// ListPendingPantryItems returns pending_pantry_items rows — optionally
// narrowed to filter.Statuses — newest first.
func (s *Store) ListPendingPantryItems(ctx context.Context, filter PendingPantryItemFilter) ([]PendingPantryItemRow, error) {
	// pgx passes a nil slice as SQL NULL; force an empty array so
	// cardinality(...) is 0 and the status filter short-circuits cleanly.
	statuses := filter.Statuses
	if statuses == nil {
		statuses = []string{}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT
			p.id, p.upc, p.name, p.quantity, p.unit_id, p.price, p.status,
			p.created_at, p.updated_at,
			u.name AS unit_name
		FROM pending_pantry_items p
		LEFT JOIN units u ON u.id = p.unit_id
		WHERE cardinality($1::text[]) = 0 OR p.status = ANY($1::text[])
		ORDER BY p.created_at DESC`, statuses)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[PendingPantryItemRow])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// findOrCreateUnit resolves a free-text unit name to a units.id, matching
// case-insensitively against existing units (name is unique
// case-insensitive) and creating a new one if none matches.
func findOrCreateUnit(ctx context.Context, tx pgx.Tx, name string) (int64, error) {
	name = strings.TrimSpace(name)

	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM units WHERE lower(name) = lower($1)`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, mapError(err)
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO units (name) VALUES ($1)
		RETURNING id`, name).Scan(&id); err != nil {
		return 0, mapError(err)
	}
	return id, nil
}
