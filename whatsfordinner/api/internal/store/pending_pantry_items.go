package store

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// pending_pantry_items.status values, per the table's CHECK constraint. A
// scanned item with a derivable UPC starts at Pending; the queue worker
// (internal/worker) claims it (-> Processing) and resolves it against
// OpenFoodFacts, landing on Approved (found) or Rejected (lookup failed, or
// no UPC could ever be derived in the first place).
const (
	PendingPantryStatusPending    = "pending"
	PendingPantryStatusProcessing = "processing"
	PendingPantryStatusApproved   = "approved"
	PendingPantryStatusRejected   = "rejected"
)

// pendingPantryItemColumns is the column list shared by every query that
// returns a full models.PendingPantryItem row.
const pendingPantryItemColumns = `id, pantry_id, upc, name, quantity, unit_id, price, status, created_at, updated_at`

// PendingPantryItemInput holds the writable fields for a receipt-scanned
// item awaiting review. Unit is free text (e.g. from a Gemini receipt scan)
// and is resolved/created against the units table by CreatePendingPantryItem.
type PendingPantryItemInput struct {
	PantryID int64
	UPC      string
	Name     string
	Quantity float64
	Unit     string
	Price    *float64
	// Status is one of the PendingPantryStatus* constants above.
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
		INSERT INTO pending_pantry_items (pantry_id, upc, name, quantity, unit_id, price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+pendingPantryItemColumns,
		in.PantryID, in.UPC, in.Name, in.Quantity, unitID, in.Price, in.Status)
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
// item plus its unit's display name (nil when UnitID is nil) and its
// destination pantry's name.
type PendingPantryItemRow struct {
	models.PendingPantryItem

	UnitName   *string `db:"unit_name"`
	PantryName string  `db:"pantry_name"`
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
			p.id, p.pantry_id, p.upc, p.name, p.quantity, p.unit_id, p.price,
			p.status, p.created_at, p.updated_at,
			u.name AS unit_name,
			pt.name AS pantry_name
		FROM pending_pantry_items p
		LEFT JOIN units u   ON u.id = p.unit_id
		JOIN pantry pt      ON pt.id = p.pantry_id
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

// ClaimPendingPantryItemsForProcessing atomically selects up to limit
// pending_pantry_items rows in PendingPantryStatusPending — oldest
// (created_at ASC) first — flips them to PendingPantryStatusProcessing, and
// returns them. FOR UPDATE SKIP LOCKED means concurrent callers never claim
// the same row twice, so this is safe even if more than one worker runs.
func (s *Store) ClaimPendingPantryItemsForProcessing(ctx context.Context, limit int) ([]models.PendingPantryItem, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE pending_pantry_items
		SET status = $1, updated_at = now()
		WHERE id IN (
			SELECT id FROM pending_pantry_items
			WHERE status = $2
			ORDER BY created_at ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+pendingPantryItemColumns,
		PendingPantryStatusProcessing, PendingPantryStatusPending, limit)
	if err != nil {
		return nil, mapError(err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.PendingPantryItem])
	if err != nil {
		return nil, mapError(err)
	}
	return items, nil
}

// SetPendingPantryItemStatus updates a single pending_pantry_items row's
// status (e.g. once the queue worker has resolved it against OpenFoodFacts).
func (s *Store) SetPendingPantryItemStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE pending_pantry_items SET status = $2, updated_at = now() WHERE id = $1`,
		id, status)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
