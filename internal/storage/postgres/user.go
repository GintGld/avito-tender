package storage

import (
	"context"
	"errors"
	"fmt"
	"tender/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// VerifyUser checks if username is in table.
func (s *Storage) VerifyUser(ctx context.Context, username string) (bool, error) {
	const op = "storage.Postgres.VerifyUser"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var exists bool

	if err := w.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM employee WHERE username=$1)", username).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return false, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// VerifyUserId checks if user id is in table.
func (s *Storage) VerifyUserId(ctx context.Context, userId uuid.UUID) (bool, error) {
	const op = "storage.Postgres.VerifyUserId"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var exists bool

	if err := w.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM employee WHERE id=$1)", userId).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return false, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// VerifyUserId checks if user id is in table.
func (s *Storage) VerifyOrgId(ctx context.Context, orgId uuid.UUID) (bool, error) {
	const op = "storage.Postgres.VerifyOrgId"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var exists bool

	if err := w.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM organization WHERE id=$1)", orgId).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return false, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// UserId returns user's id by its name.
func (s *Storage) UserId(ctx context.Context, username string) (uuid.UUID, error) {
	const op = "storage.Postgres.UserId"

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

	if err := w.QueryRow(ctx, "SELECT id from employee where username=$1", username).Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return uuid.Nil, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// VerifyUserPermission check if username is related to organization.
func (s *Storage) VerifyUserPermission(ctx context.Context, username string, orgId uuid.UUID) (bool, error) {
	const op = "storage.Postgres.VerifyUserPermission"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return false, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var exists bool

	if err := w.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM organization_responsible
			WHERE organization_id=$1 AND
				user_id=(
					SELECT id from employee
					WHERE username=$2
				)
		)
		`, orgId, username).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return false, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return exists, nil
}

// OrgSize returns # of org employees.
func (s *Storage) OrgSize(ctx context.Context, orgId uuid.UUID) (int64, error) {
	const op = "storage.Postgres.OrgSize"

	// Get worker
	var w worker
	if w = s.tx(ctx); w == nil {
		conn, err := s.conn(ctx)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}
		defer conn.Release()
		w = conn
	}

	var size int64

	if err := w.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM employee e
		JOIN organization_responsible r ON e.id = r.user_id
		JOIN organization o ON o.id = r.organization_id
		WHERE o.id = $1
	`, orgId).
		Scan(&size); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storage.ErrOrgNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return 0, fmt.Errorf("%s pgx error: [%s] %s", op, pgErr.Code, pgErr.Message)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return size, nil
}
