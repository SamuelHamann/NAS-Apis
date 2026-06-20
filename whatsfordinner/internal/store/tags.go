package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ListTags returns a page of tags ordered by name.
func (s *Store) ListTags(ctx context.Context, limit, offset int) ([]models.Tag, error) {
	limit, offset = clampPage(limit, offset)

	rows, err := s.pool.Query(ctx, `
		SELECT id, name
		FROM tags
		ORDER BY name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, mapError(err)
	}

	tags, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Tag])
	if err != nil {
		return nil, mapError(err)
	}
	return tags, nil
}

// GetTag returns a single tag by ID.
func (s *Store) GetTag(ctx context.Context, id int64) (models.Tag, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM tags WHERE id = $1`, id)
	if err != nil {
		return models.Tag{}, mapError(err)
	}

	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
	if err != nil {
		return models.Tag{}, mapError(err)
	}
	return tag, nil
}

// CreateTag inserts a new tag and returns the stored row.
func (s *Store) CreateTag(ctx context.Context, name string) (models.Tag, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO tags (name)
		VALUES ($1)
		RETURNING id, name`, name)
	if err != nil {
		return models.Tag{}, mapError(err)
	}

	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
	if err != nil {
		return models.Tag{}, mapError(err)
	}
	return tag, nil
}

// UpdateTag renames an existing tag and returns the stored row.
func (s *Store) UpdateTag(ctx context.Context, id int64, name string) (models.Tag, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE tags
		SET name = $2
		WHERE id = $1
		RETURNING id, name`, id, name)
	if err != nil {
		return models.Tag{}, mapError(err)
	}

	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Tag])
	if err != nil {
		return models.Tag{}, mapError(err)
	}
	return tag, nil
}

// DeleteTag removes a tag by ID.
func (s *Store) DeleteTag(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
