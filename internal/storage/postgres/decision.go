package storage

import (
	"context"
	"errors"
	"fmt"

	"tender/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InsertDecision inserts decision.
func (s *Storage) InsertDecision(ctx context.Context, decision models.Decision) error {
	const op = "storage.Postgres.InsertDecision"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	if _, err := w.Exec(ctx, `
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM decision
				WHERE user_id=$1 AND bid_id=$2
			) THEN
				INSERT INTO decision(user_id,bid_id,decision)
				VALUES($1,$2,$3)
			ELSE
				UPDATE decision
				SET decision=$3
				WHERE user_id=$1 AND bid_id=$2
			END IF;
		END $$
	`, decision.UserId, decision.BidId, decision.Decision); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Decisions returns all decisions for bid id.
func (s *Storage) Decisions(ctx context.Context, bidId uuid.UUID) ([]models.Decision, error) {
	const op = "storage.Postgres.Decision"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var d models.Decision
	d.BidId = bidId
	decisions := make([]models.Decision, 0)

	rows, err := w.Query(ctx, `
		SELECT user_id, decision
		FROM decision
		WHERE bid_id=$1
	`, bidId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for rows.Next() {
		if err := rows.Scan(&d.UserId, &d.Decision); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		decisions = append(decisions, d)
	}

	return decisions, nil
}
