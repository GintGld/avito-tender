package storage

import (
	"context"
	"errors"
	"fmt"

	"tender/internal/models"
	"tender/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// SaveTender saves outdated tender to rollback table.
func (s *Storage) SaveTender(ctx context.Context, tender models.Tender) error {
	const op = "storage.Postgres.SaveTender"

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
		INSERT INTO rollback_tender(id, organization_id, name, description, type, status, version, created_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	`, tender.Id, tender.OrgId, tender.Name, tender.Desc, tender.ServiceType, tender.Status, tender.Version, tender.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// SaveBid saves outdated bid to rollback table.
func (s *Storage) SaveBid(ctx context.Context, bid models.Bid) error {
	const op = "storage.Postgres.SaveBid"

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
		INSERT INTO rollback_bid(id, tender_id, name, description, status, author_type, author_id, version, created_at)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, bid.Id, bid.TenderId, bid.Name, bid.Desc, bid.Status, bid.AuthorType, bid.AuthorId, bid.Version, bid.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// RecoverTender returns old tender.
func (s *Storage) RecoverTender(ctx context.Context, tenderId uuid.UUID, version int32) (models.Tender, error) {
	const op = "storage.Postgres.RecoverTender"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return models.Tender{}, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var tender models.Tender

	if err := w.QueryRow(ctx, `
		SELECT id, organization_id, name, description, type, status, version, created_at
		FROM rollback_tender
		WHERE id=$1 AND version=$2
	`, tenderId, version).
		Scan(&tender.Id, &tender.OrgId, &tender.Name, &tender.Desc, &tender.ServiceType, &tender.Status, &tender.Version, &tender.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Tender{}, storage.ErrVersionNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Tender{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return tender, nil
}

// RecoverBid returns old bid.
func (s *Storage) RecoverBid(ctx context.Context, bidId uuid.UUID, version int32) (models.Bid, error) {
	const op = "storage.Postgres.RecoverBid"

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
		SELECT id, tender_id, name, description, status, author_type, author_id, version, created_at
		FROM rollback_bid
		WHERE id=$1 AND version=$2
	`, bidId, version).
		Scan(&bid.Id, &bid.TenderId, &bid.Name, &bid.Desc, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Bid{}, storage.ErrVersionNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Bid{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid, nil
}
