package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"tender/internal/lib/logger/sl"
	"tender/internal/service"
	"tender/internal/storage"

	"github.com/google/uuid"
)

type User struct {
	log             *slog.Logger
	employeeStorage EmployeeStorage
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name EmployeeStorage
type EmployeeStorage interface {
	VerifyUser(ctx context.Context, username string) (bool, error)
	VerifyUserId(ctx context.Context, userId uuid.UUID) (bool, error)
	VerifyOrgId(ctx context.Context, userId uuid.UUID) (bool, error)
	UserId(ctx context.Context, username string) (uuid.UUID, error)
	VerifyUserPermission(ctx context.Context, username string, orgId uuid.UUID) (bool, error)
	OrgSize(ctx context.Context, orgId uuid.UUID) (int64, error)
}

func New(
	log *slog.Logger,
	employeeStorage EmployeeStorage,
) *User {
	return &User{
		log:             log,
		employeeStorage: employeeStorage,
	}
}

// Validate checks if user exists.
func (u *User) Validate(ctx context.Context, username string) error {
	const op = "User.Validate"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", username),
	)

	// Check if user exists.
	userOk, err := u.employeeStorage.VerifyUser(ctx, username)
	if err != nil {
		log.Error("failed to verify user", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	if !userOk {
		log.Warn("user not found")
		return service.ErrUserNotFound
	}

	return nil
}

func (u *User) ValidateUserId(ctx context.Context, userId uuid.UUID) error {
	const op = "User.ValidateUserId"

	log := u.log.With(
		slog.String("op", op),
		slog.String("user id", userId.String()),
	)

	// Check if user exists.
	userOk, err := u.employeeStorage.VerifyUserId(ctx, userId)
	if err != nil {
		log.Error("failed to verify user", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	if !userOk {
		log.Warn("user not found")
		return service.ErrUserNotFound
	}

	return nil
}

func (u *User) ValidateOrgId(ctx context.Context, orgId uuid.UUID) error {
	const op = "User.ValidateOrgId"

	log := u.log.With(
		slog.String("op", op),
		slog.String("user id", orgId.String()),
	)

	// Check if org exists.
	userOk, err := u.employeeStorage.VerifyOrgId(ctx, orgId)
	if err != nil {
		log.Error("failed to verify organization", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	if !userOk {
		log.Warn("org not found")
		return service.ErrOrganizationNotFound
	}

	return nil
}

func (u *User) UserId(ctx context.Context, username string) (uuid.UUID, error) {
	const op = "User.UserId"

	log := u.log.With(
		slog.String("op", op),
		slog.String("user name", username),
	)

	// Get user's id.
	id, err := u.employeeStorage.UserId(ctx, username)
	if err != nil {
		log.Error("failed to verify organization", sl.Err(err))
		return uuid.Nil, fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

// Permission checks if user is allowed to modilfy organization's tenders.
//
// Should be called with existing username.
func (u *User) Permission(ctx context.Context, username string, orgId uuid.UUID) error {
	const op = "User.Permission"

	log := u.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("organization id", orgId.String()),
	)

	// Check if user has permissions to update tender status.
	permOk, err := u.employeeStorage.VerifyUserPermission(ctx, username, orgId)
	if err != nil {
		log.Error("failed to verify user permissions", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}
	if !permOk {
		log.Warn("user can't update tender")
		return service.ErrNotEnoughPrivileges
	}

	return nil
}

// OrgSize returns # of employees in org.
func (u *User) OrgSize(ctx context.Context, orgId uuid.UUID) (int64, error) {
	const op = "User.OrgSize"

	log := u.log.With(
		slog.String("op", op),
		slog.String("organization id", orgId.String()),
	)

	size, err := u.employeeStorage.OrgSize(ctx, orgId)
	if err != nil {
		if errors.Is(err, storage.ErrOrgNotFound) {
			log.Warn("org not found")
			return 0, service.ErrOrganizationNotFound
		}
		log.Error("failed to get org size", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return size, nil
}
