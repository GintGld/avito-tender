package storage

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"tender/internal/models"
	"tender/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// InsertTedner inserts tender, returns initialized tender.
func (s *Storage) InsertTender(ctx context.Context, tender models.Tender) (models.Tender, error) {
	const op = "storage.postgres.InsertTender"

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

	if err := w.QueryRow(ctx, `
		INSERT INTO tender(organization_id, name, description, type, status, version)
		VALUES($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		tender.OrgId, tender.Name, tender.Desc, tender.ServiceType, tender.Status, tender.Version,
	).Scan(&tender.Id, &tender.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Tender{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return tender, nil
}

// Tedner returns tender by its id.
func (s *Storage) Tender(ctx context.Context, id uuid.UUID) (models.Tender, error) {
	const op = "storage.Postgres.Tender"

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

	if err := w.QueryRow(ctx, `SELECT id, organization_id, name, description, type, status, version, created_at FROM tender WHERE id=$1`, id).
		Scan(&tender.Id, &tender.OrgId, &tender.Name, &tender.Desc, &tender.ServiceType, &tender.Status, &tender.Version, &tender.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Tender{}, storage.ErrTenderNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Tender{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return tender, nil
}

// UpdateTender updates tender.
func (s *Storage) UpdateTender(ctx context.Context, tender models.Tender) error {
	const op = "storage.Postgres.UpdateTender"

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
		UPDATE tender
		SET organization_id=$2,name=$3,description=$4,type=$5,status=$6,version=$7
		WHERE id=$1
	`, tender.Id, tender.OrgId, tender.Name, tender.Desc, tender.ServiceType, tender.Status, tender.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrTenderNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Tenders returns tenders in alphabet order.
func (s *Storage) Tenders(ctx context.Context, limit, offset int32, services []models.ServiceType) ([]models.Tender, error) {
	const op = "storage.Postgres.Tenders"

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

	types := make([]string, 0, len(services))
	for _, s := range services {
		types = append(types, "'"+string(s)+"'")
	}

	typeCondition := ""
	if len(types) > 0 {
		typeCondition = fmt.Sprintf("AND type IN (%s)", strings.Join(types, ","))
	}

	rows, err := w.Query(ctx, fmt.Sprintf(`
		SELECT id, organization_id, name, description, type, status, version, created_at
		FROM tender
		WHERE status='Published' %s
		ORDER BY name ASC
		LIMIT $1
		OFFSET $2
	`, typeCondition), limit, offset)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrTenderNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var tender models.Tender
	tenders := make([]models.Tender, 0, limit)

	for rows.Next() {
		if err := rows.Scan(&tender.Id, &tender.OrgId, &tender.Name, &tender.Desc, &tender.ServiceType, &tender.Status, &tender.Version, &tender.CreatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		tenders = append(tenders, tender)
	}

	return slices.Clip(tenders), nil
}

// UserTenders returns tenders related to user.
func (s *Storage) UserTenders(ctx context.Context, limit, offset int32, username string) ([]models.Tender, error) {
	const op = "storage.Postgres.UserTenders"

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
		SELECT id, organization_id, name, description, type, status, version, created_at
		FROM tender
		WHERE organization_id=(
			SELECT id from employee
			WHERE username=$1
		)
		ORDER BY name ASC
		LIMIT $2
		OFFSET $3
	`, username, limit, offset)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrTenderNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var tender models.Tender
	tenders := make([]models.Tender, 0, limit)

	for rows.Next() {
		if err := rows.Scan(&tender.Id, &tender.OrgId, &tender.Name, &tender.Desc, &tender.ServiceType, &tender.Status, &tender.Version, &tender.CreatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
			}
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		tenders = append(tenders, tender)
	}

	return slices.Clip(tenders), nil
}

// TenderSetStatus updates tender status.
func (s *Storage) TenderSetStatus(ctx context.Context, tenderId uuid.UUID, status models.TenderStatus) (models.Tender, error) {
	const op = "storage.Postgres.TenderSetStatus"

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
		UPDATE tender
		SET status=$2
		WHERE id=$1
		RETURNING id, organization_id, name, description, type, status, version, created_at
	`, tenderId, status).
		Scan(&tender.Id, &tender.OrgId, &tender.Name, &tender.Desc, &tender.ServiceType, &tender.Status, &tender.Version, &tender.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Tender{}, storage.ErrTenderNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return models.Tender{}, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return tender, nil
}
