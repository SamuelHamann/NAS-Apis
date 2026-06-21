package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ListFoodLocations returns a page of food locations ordered by name.
func (s *Store) ListFoodLocations(ctx context.Context, limit, offset int) ([]models.FoodLocation, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, created_at
		FROM food_locations
		ORDER BY name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	locations, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.FoodLocation])
	if err != nil {
		return nil, mapError(err)
	}
	return locations, nil
}

// GetFoodLocation returns a single food location by ID.
func (s *Store) GetFoodLocation(ctx context.Context, id int64) (models.FoodLocation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, created_at
		FROM food_locations
		WHERE id = $1`, id)
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}

	location, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.FoodLocation])
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}
	return location, nil
}

// CreateFoodLocation inserts a new food location and returns the stored row.
func (s *Store) CreateFoodLocation(ctx context.Context, name string) (models.FoodLocation, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO food_locations (name)
		VALUES ($1)
		RETURNING id, name, created_at`, name)
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}

	location, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.FoodLocation])
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}
	return location, nil
}

// UpdateFoodLocation renames an existing food location and returns the stored row.
func (s *Store) UpdateFoodLocation(ctx context.Context, id int64, name string) (models.FoodLocation, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE food_locations
		SET name = $2
		WHERE id = $1
		RETURNING id, name, created_at`, id, name)
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}

	location, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.FoodLocation])
	if err != nil {
		return models.FoodLocation{}, mapError(err)
	}
	return location, nil
}

// DeleteFoodLocation removes a food location by ID.
func (s *Store) DeleteFoodLocation(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM food_locations WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
