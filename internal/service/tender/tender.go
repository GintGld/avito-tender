package tender

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"tender/internal/lib/logger/sl"
	"tender/internal/models"
	"tender/internal/service"
	"tender/internal/storage"

	"github.com/google/uuid"
)

type Tender struct {
	log           *slog.Logger
	tenderStorage TenderStorage
	userSrv       UserService
	rollbackSrv   RollbackService
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name UserService
type UserService interface {
	Validate(ctx context.Context, username string) error
	Permission(ctx context.Context, username string, orgId uuid.UUID) error
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name RollbackService
type RollbackService interface {
	SaveTender(ctx context.Context, tender models.Tender) error
	// Save outdated tender and recover old tender.
	SwapTender(ctx context.Context, tenderId uuid.UUID, version int32, outdatedTedner models.Tender) (models.Tender, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name TenderStorage
type TenderStorage interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	InsertTender(ctx context.Context, tender models.Tender) (models.Tender, error)
	Tender(ctx context.Context, id uuid.UUID) (models.Tender, error)
	UpdateTender(ctx context.Context, tender models.Tender) error
	Tenders(ctx context.Context, limit, offset int32, services []models.ServiceType) ([]models.Tender, error)
	UserTenders(ctx context.Context, limit, offset int32, username string) ([]models.Tender, error)
	TenderSetStatus(ctx context.Context, tenderId uuid.UUID, status models.TenderStatus) (models.Tender, error)
}

func New(
	log *slog.Logger,
	userSrv UserService,
	rollback RollbackService,
	tenderStorage TenderStorage,
) *Tender {
	return &Tender{
		log:           log,
		tenderStorage: tenderStorage,
		userSrv:       userSrv,
		rollbackSrv:   rollback,
	}
}

// New adds new tender.
func (t *Tender) New(ctx context.Context, tenderNew models.TenderNew) (models.TenderOut, error) {
	const op = "Tender.New"

	log := t.log.With(
		slog.String("op", op),
		slog.String("username", tenderNew.CreatorUsername),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, tenderNew.CreatorUsername); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.TenderOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Create tender with version=1.
	tender := tenderNew.ToTender()

	// Insert tender.
	tender, err = t.tenderStorage.InsertTender(ctx, tender)
	if err != nil {
		log.Error("failed to insert tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}
	return tender.ToOut(), nil
}

// All returns all tenders.
func (t *Tender) All(ctx context.Context, limit, offset int32, services []models.ServiceType) ([]models.TenderOut, error) {
	const op = "Tender.All"

	log := t.log.With(slog.String("op", op))

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Get all tenders.
	res, err := t.tenderStorage.Tenders(ctx, limit, offset, services)
	if err != nil {
		log.Error("failed to get tenders", slog.Int("limit", int(limit)), slog.Int("offset", int(offset)), slog.Any("services", services), sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Convert slice elements.
	out := make([]models.TenderOut, 0, len(res))
	for i := range res {
		out = append(out, res[i].ToOut())
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, err
}

// My returns user's tenders.
func (t *Tender) My(ctx context.Context, limit, offset int32, username string) ([]models.TenderOut, error) {
	const op = "Tender.My"

	log := t.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return nil, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get user's tenders.
	res, err := t.tenderStorage.UserTenders(ctx, limit, offset, username)
	if err != nil {
		log.Error("failed to get tenders", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Convert slice elements.
	out := make([]models.TenderOut, 0, len(res))
	for i := range res {
		out = append(out, res[i].ToOut())
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, nil
}

// TenderStatus returns tender status.
func (t *Tender) Status(ctx context.Context, username string, tenderId uuid.UUID) (models.TenderStatus, error) {
	const op = "Tender.TenderStatus"

	log := t.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", tenderId.String()),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return "", err
		}
		log.Error("failed to verify user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	// Get tender
	tender, err := t.tenderStorage.Tender(ctx, tenderId)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return "", service.ErrTenderNotFound
		}
		log.Error("failed to get tendet status", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := t.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return "", service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission")
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return tender.Status, nil
}

// TenderSetStatus updates tender status.
// If it is not allowed for user returns error.
func (t *Tender) SetStatus(ctx context.Context, username string, tenderId uuid.UUID, status models.TenderStatus) (models.TenderOut, error) {
	const op = "Tender.TenderSetStatus"

	log := t.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", tenderId.String()),
		slog.String("new status", string(status)),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.TenderOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get tender.
	tender, err := t.tenderStorage.Tender(ctx, tenderId)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.TenderOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to modify tender.
	if err := t.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return models.TenderOut{}, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission")
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Update tender status.
	tender, err = t.tenderStorage.TenderSetStatus(ctx, tenderId, status)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Error("tender not found")
			return models.TenderOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to update tender status", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return tender.ToOut(), nil
}

// Edit updates tender.
// If it is not allowed for user returns error.
func (t *Tender) Edit(ctx context.Context, username string, tenderId uuid.UUID, patch models.TenderPatch) (models.TenderOut, error) {
	const op = "Tender.Edit"

	log := t.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", tenderId.String()),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.TenderOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get tender.
	tender, err := t.tenderStorage.Tender(ctx, tenderId)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.TenderOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to modify tender.
	if err := t.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return models.TenderOut{}, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Apply tender.
	newTender := tender
	newTender.Patch(patch)
	newTender.Version += 1

	// Update tender.
	if err := t.tenderStorage.UpdateTender(ctx, newTender); err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.TenderOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to updated tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save old version of tender.
	if err := t.rollbackSrv.SaveTender(ctx, tender); err != nil {
		log.Error("failed to insert tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return newTender.ToOut(), nil
}

// Rollback restores old tender version.
// If version doesn't exist returns error.
func (t *Tender) Rollback(ctx context.Context, username string, id uuid.UUID, version int32) (models.TenderOut, error) {
	const op = "Tender.Rollback"

	log := t.log.With(
		slog.String("op", op),
		slog.String("user", username),
		slog.String("id", id.String()),
		slog.Int("version", int(version)),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := t.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.TenderOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get actual tender.
	tender, err := t.tenderStorage.Tender(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.TenderOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to modify tender.
	if err := t.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return models.TenderOut{}, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission")
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save outdated tender and recover old tender.
	recoveredTender, err := t.rollbackSrv.SwapTender(ctx, id, version, tender)
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			log.Warn("version not found")
			return models.TenderOut{}, service.ErrVersionNotFound
		}
		log.Error("failed to recover old version", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save recovered tender.
	recoveredTender.Version = tender.Version + 1
	recoveredTender.Status = tender.Status
	newTender, err := t.tenderStorage.InsertTender(ctx, recoveredTender)
	if err != nil {
		log.Error("failed to insert tender", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.TenderOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return newTender.ToOut(), nil
}

// Tender return tender by its id.
func (t *Tender) Tender(ctx context.Context, tenderId uuid.UUID) (models.Tender, error) {
	const op = "Tender.Tender"

	log := t.log.With(
		slog.String("op", op),
		slog.String("id", tenderId.String()),
	)

	ctx, err := t.tenderStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := t.tenderStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	res, err := t.tenderStorage.Tender(ctx, tenderId)
	if err != nil {
		if errors.Is(err, storage.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.Tender{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := t.tenderStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return res, nil
}
