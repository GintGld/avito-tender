package storage

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"tender/internal/models"
	"tender/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InsertBid insert bid. Returns initialized bid.
func (s *Storage) InsertBid(ctx context.Context, bid models.Bid) (models.Bid, error) {
	const op = "storage.Postgres.InsertBid"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return models.Bid{}, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	if err := w.QueryRow(ctx, `
		INSERT INTO bid(tender_id, name, description, status, author_type, author_id, version)
		VALUES($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, bid.TenderId, bid.Name, bid.Desc, bid.Status, bid.AuthorType, bid.AuthorId, bid.Version).
		Scan(&bid.Id, &bid.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Bid{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid, nil
}

// Bid returns Bid by its id.
func (s *Storage) Bid(ctx context.Context, bidId uuid.UUID) (models.Bid, error) {
	const op = "storage.Postgres.Bid"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return models.Bid{}, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var bid models.Bid

	if err := w.QueryRow(ctx, `SELECT id, tender_id, name, description, status, author_type, author_id, version, created_at FROM bid WHERE id=$1`, bidId).
		Scan(&bid.Id, &bid.TenderId, &bid.Name, &bid.Desc, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Bid{}, storage.ErrBidNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Bid{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid, nil
}

// UpdateBid updates bid.
func (s *Storage) UpdateBid(ctx context.Context, bid models.Bid) error {
	const op = "storage.Postgres.UpdateBid"

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
		UPDATE bid
		SET name=$2,description=$3,status=$4,author_type=$5,author_id=$6,version=$7
		WHERE id=$1
	`, bid.Id, bid.Name, bid.Desc, bid.Status, bid.AuthorType, bid.AuthorId, bid.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrBidNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// TenderBids returns published bids related to tender.
func (s *Storage) TenderBids(ctx context.Context, tenderId uuid.UUID, limit, offset int32) ([]models.Bid, error) {
	const op = "storage.Postgres.TenderBids"

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
		SELECT id, tender_id, name, description, status, author_type, author_id, version, created_at
		FROM bid
		WHERE
			tender_id=$1
			AND
			status='Published'
		ORDER BY name ASC
		LIMIT $2
		OFFSET $3
	`, tenderId, limit, offset)
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

	var bid models.Bid
	bids := make([]models.Bid, 0, limit)

	for rows.Next() {
		if err := rows.Scan(&bid.Id, &bid.TenderId, &bid.Name, &bid.Desc, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		bids = append(bids, bid)
	}

	return slices.Clip(bids), nil
}

// UserBids returns user's bids.
func (s *Storage) UserBids(ctx context.Context, username string, limit, offset int32) ([]models.Bid, error) {
	const op = "storage.Postgres.UserBids"

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
		SELECT id, tender_id, name, description, status, author_type, author_id, version, created_at
		FROM bid
		WHERE
			author_type='User'
			AND
			author_id IN (
				SELECT id
				FROM employee
				WHERE username=$1
			)
		ORDER BY name ASC
		LIMIT $2
		OFFSET $3
	`, username, limit, offset)
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

	var bid models.Bid
	bids := make([]models.Bid, 0, limit)

	for rows.Next() {
		if err := rows.Scan(&bid.Id, &bid.TenderId, &bid.Name, &bid.Desc, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		bids = append(bids, bid)
	}

	return slices.Clip(bids), nil
}

// BidSetStatus updates bid status.
func (s *Storage) BidSetStatus(ctx context.Context, bidId uuid.UUID, status models.BidStatus) (models.Bid, error) {
	const op = "storage.Postgres.BidSetStatus"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return models.Bid{}, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var bid models.Bid

	if err := w.QueryRow(ctx, `
		UPDATE bid
		SET status=$2
		WHERE id=$1
		RETURNING id, tender_id, name, description, status, author_type, author_id, version, created_at
	`, bidId, status).
		Scan(&bid.Id, &bid.TenderId, &bid.Name, &bid.Desc, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Bid{}, storage.ErrBidNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Bid{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid, nil
}
