package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"ruangku-backend/config"
)

const defaultDBURL = "postgres://ruangku:ruangku@localhost:5432/ruangku?sslmode=disable"

// NewPool membuka connection pool ke PostgreSQL dan memastikan database
// bisa dihubungi sebelum dikembalikan.
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, config.GetEnv("DB_URL", defaultDBURL))
	if err != nil {
		return nil, fmt.Errorf("membuat pool database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
