package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// collectionColumns is the full column list for the collections table.
const collectionColumns = `id, user_id, name, created_at, updated_at`

// ListCollectionsForUser returns a user's collections, alphabetical. There
// is no pagination — a household member is expected to have a handful of
// collections at most.
func (s *Store) ListCollectionsForUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+collectionColumns+`
		FROM collections
		WHERE user_id = $1
		ORDER BY name`, userID)
	if err != nil {
		return nil, mapError(err)
	}

	collections, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Collection])
	if err != nil {
		return nil, mapError(err)
	}
	return collections, nil
}

// GetCollection returns a single collection by ID.
func (s *Store) GetCollection(ctx context.Context, id int64) (models.Collection, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+collectionColumns+` FROM collections WHERE id = $1`, id)
	if err != nil {
		return models.Collection{}, mapError(err)
	}

	collection, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Collection])
	if err != nil {
		return models.Collection{}, mapError(err)
	}
	return collection, nil
}

// CreateCollection inserts a new collection owned by userID and returns the
// stored row. The (user_id, name) unique constraint means creating two
// collections with the same name under the same user surfaces as
// ErrConflict.
func (s *Store) CreateCollection(ctx context.Context, userID int64, name string) (models.Collection, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO collections (user_id, name)
		VALUES ($1, $2)
		RETURNING `+collectionColumns, userID, name)
	if err != nil {
		return models.Collection{}, mapError(err)
	}

	collection, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Collection])
	if err != nil {
		return models.Collection{}, mapError(err)
	}
	return collection, nil
}

// CollectionRecipe is one (collection_id, recipe_id) pair from the
// collection_recipes junction table.
type CollectionRecipe struct {
	CollectionID int64
	RecipeID     int64
}

// ListCollectionRecipesForUser returns every (collection_id, recipe_id) pair
// for collections owned by userID — used to pre-check "already in this
// collection" state across the recipes list/detail page's "add to
// collection" widgets without one query per row.
func (s *Store) ListCollectionRecipesForUser(ctx context.Context, userID int64) ([]CollectionRecipe, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cr.collection_id, cr.recipe_id
		FROM   collection_recipes cr
		JOIN   collections c ON c.id = cr.collection_id
		WHERE  c.user_id = $1`, userID)
	if err != nil {
		return nil, mapError(err)
	}

	pairs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CollectionRecipe, error) {
		var p CollectionRecipe
		err := row.Scan(&p.CollectionID, &p.RecipeID)
		return p, err
	})
	if err != nil {
		return nil, mapError(err)
	}
	return pairs, nil
}

// ToggleCollectionRecipe adds recipeID to collectionID if it isn't already
// there, or removes it if it is, and reports the resulting membership state
// (true = now a member). Callers are responsible for checking the
// collection belongs to whoever is making the request — this just performs
// the toggle.
func (s *Store) ToggleCollectionRecipe(ctx context.Context, collectionID, recipeID int64) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, mapError(err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		DELETE FROM collection_recipes WHERE collection_id = $1 AND recipe_id = $2`,
		collectionID, recipeID)
	if err != nil {
		return false, mapError(err)
	}

	added := tag.RowsAffected() == 0
	if added {
		if _, err := tx.Exec(ctx, `
			INSERT INTO collection_recipes (collection_id, recipe_id)
			VALUES ($1, $2)`, collectionID, recipeID); err != nil {
			return false, mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, mapError(err)
	}
	return added, nil
}
