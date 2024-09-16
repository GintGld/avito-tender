package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey string

const (
	Begin txKey = "storage.Postgres.tx"
)

type worker interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Begin starts transaction.
func (s *Storage) Begin(ctx context.Context) (context.Context, error) {
	const op = "storage.Postgres.Begin"

	if s.tx(ctx) != nil {
		return ctx, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return context.Background(), fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return context.Background(), fmt.Errorf("%s: %w", op, err)
	}

	ctx = s.setTx(ctx, tx)

	return ctx, nil
}

// Commit commits tx saved in context.
func (s *Storage) Commit(ctx context.Context) error {
	const op = "storage.Postgres.Commit"

	tx := s.tx(ctx)

	if err := tx.Commit(ctx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Rollback rolls back tx saved in context.
func (s *Storage) Rollback(ctx context.Context) error {
	const op = "storage.Postgres.Rollback"

	tx := s.tx(ctx)

	if err := tx.Rollback(ctx); err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrTxClosed) {
			return nil
		}
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// setTx links tx to given context.
func (s *Storage) setTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, Begin, tx)
}

// tx extracts tx from context.
// If Begin was not been called panics.
func (s *Storage) tx(ctx context.Context) pgx.Tx {
	const op = "storage.Postgres.tx"

	val := ctx.Value(Begin)
	if val == nil {
		return nil
	}

	tx, ok := val.(pgx.Tx)
	if !ok {
		panic(fmt.Errorf("%s: can't cast context value to pdx.Tx", op))
	}

	return tx
}

// conn returns new conn
func (s *Storage) conn(ctx context.Context) (*pgxpool.Conn, error) {
	const op = "storage.Postgres.conn"

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return conn, err
}
