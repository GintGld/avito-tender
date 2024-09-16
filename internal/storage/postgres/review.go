package storage

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"tender/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InsertReview inserts review. Returns its id.
func (s *Storage) InsertReview(ctx context.Context, review models.Review) (uuid.UUID, error) {
	const op = "storage.Postgres.InsertReview"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return uuid.Nil, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var id uuid.UUID

	if err := w.QueryRow(ctx, `
		INSERT INTO review(bid_id, description, author)
		VALUES($1, $2, $3)
		RETURNING id
	`, review.BidId, review.Desc, review.AuthorName).
		Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return uuid.Nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// Reviews returns review by their author and tender.
func (s *Storage) Reviews(ctx context.Context, tenderId uuid.UUID, author string, limit, offset int32) ([]models.Review, error) {
	const op = "storage.Postgres.Reviews"

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

	rows, err := w.Query(ctx, `
		SELECT id, bid_id, description, author, created_at
		FROM review
		WHERE 
			bid_id IN (
				SELECT id
				FROM bid
				WHERE tender_id=$1
			) AND
			author=$2

	`, tenderId, author)
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

	var review models.Review
	reviews := make([]models.Review, 0, limit)

	for rows.Next() {
		if err := rows.Scan(&review.Id, &review.BidId, &review.Desc, &review.AuthorName, &review.CreatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		reviews = append(reviews, review)
	}

	return slices.Clip(reviews), nil
}
