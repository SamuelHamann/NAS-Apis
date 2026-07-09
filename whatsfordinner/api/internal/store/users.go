package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/models"
)

// ListUsers returns every user ordered by username. There is no pagination:
// the list is expected to stay small (a household's worth of people) since
// it doubles as the sign-in picker rendered on every visit to /login.
func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, username, created_at, updated_at
		FROM users
		ORDER BY username`)
	if err != nil {
		return nil, mapError(err)
	}

	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, mapError(err)
	}
	return users, nil
}

// GetUser returns a single user by ID.
func (s *Store) GetUser(ctx context.Context, id int64) (models.User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, username, created_at, updated_at
		FROM users
		WHERE id = $1`, id)
	if err != nil {
		return models.User{}, mapError(err)
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return models.User{}, mapError(err)
	}
	return user, nil
}

// CreateUser inserts a new user and returns the stored row.
func (s *Store) CreateUser(ctx context.Context, username string) (models.User, error) {
	rows, err := s.pool.Query(ctx, `
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id, username, created_at, updated_at`, strings.TrimSpace(username))
	if err != nil {
		return models.User{}, mapError(err)
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return models.User{}, mapError(err)
	}
	return user, nil
}

// UpdateUser renames an existing user and returns the stored row.
func (s *Store) UpdateUser(ctx context.Context, id int64, username string) (models.User, error) {
	rows, err := s.pool.Query(ctx, `
		UPDATE users
		SET username = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, username, created_at, updated_at`, id, strings.TrimSpace(username))
	if err != nil {
		return models.User{}, mapError(err)
	}

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return models.User{}, mapError(err)
	}
	return user, nil
}

// DeleteUser removes a user by ID.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return mapError(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
