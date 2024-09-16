package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

// New returns new storage instance.
// If error occurs error is returned.
func New(dbURL string) (*Storage, error) {
	const op = "storage.postgres.New"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		pool: pool,
	}, nil
}

// Stop stops underlying pgx pool.
func (s *Storage) Stop() {
	s.pool.Close()
}
