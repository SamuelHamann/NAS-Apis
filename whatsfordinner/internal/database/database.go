// Package database manages the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/config"
)

// NewPool creates a pgx connection pool from the configuration and verifies
// connectivity with a ping. The caller is responsible for closing the pool.
func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
	}

	if cfg.DBMaxConns > 0 {
		poolCfg.MaxConns = cfg.DBMaxConns
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.DBConnTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
