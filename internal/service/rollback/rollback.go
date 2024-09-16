package service

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

type Rollback struct {
	log             *slog.Logger
	rollbackStorage RollbackStorage
}

func New(
	log *slog.Logger,
	rollbackStorage RollbackStorage,
) *Rollback {
	return &Rollback{
		log:             log,
		rollbackStorage: rollbackStorage,
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name RollbackStorage
type RollbackStorage interface {
	SaveTender(ctx context.Context, tender models.Tender) error
	SaveBid(ctx context.Context, bid models.Bid) error
	RecoverTender(ctx context.Context, tenderId uuid.UUID, version int32) (models.Tender, error)
	RecoverBid(ctx context.Context, bidId uuid.UUID, version int32) (models.Bid, error)
}

// SaveTender saves outdated tender.
func (r *Rollback) SaveTender(ctx context.Context, tender models.Tender) error {
	const op = "Rollback.SaveTender"

	log := r.log.With(
		slog.String("op", op),
		slog.String("id", tender.Id.String()),
	)

	// Save tender.
	if err := r.rollbackStorage.SaveTender(ctx, tender); err != nil {
		log.Error("failed to save tender", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// SwapTender saves outdated tender and restores old tender.
func (r *Rollback) SwapTender(ctx context.Context, tenderId uuid.UUID, version int32, outdatedTedner models.Tender) (models.Tender, error) {
	const op = "Rollback.SwapTender"

	log := r.log.With(
		slog.String("op", op),
		slog.String("id", tenderId.String()),
		slog.Int("version", int(version)),
	)

	// Save outdated tender.
	if err := r.rollbackStorage.SaveTender(ctx, outdatedTedner); err != nil {
		log.Error("failed to save outdated tender", sl.Err(err))
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	// recover old tender.
	oldTender, err := r.rollbackStorage.RecoverTender(ctx, tenderId, version)
	if err != nil {
		if errors.Is(err, storage.ErrVersionNotFound) {
			log.Warn("version not found")
			return models.Tender{}, service.ErrVersionNotFound
		}
		log.Error("failed to restore tender", sl.Err(err))
		return models.Tender{}, fmt.Errorf("%s: %w", op, err)
	}

	return oldTender, nil
}

// SaveBid saves outdated bid.
func (r *Rollback) SaveBid(ctx context.Context, bid models.Bid) error {
	const op = "Rollback.SaveBid"

	log := r.log.With(
		slog.String("op", op),
		slog.String("id", bid.Id.String()),
	)

	// Save bid.
	if err := r.rollbackStorage.SaveBid(ctx, bid); err != nil {
		log.Error("failed to save bid", sl.Err(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// SwapBid saves outdated bid and restores old bid.
func (r *Rollback) SwapBid(ctx context.Context, bidId uuid.UUID, version int32, outdatedBid models.Bid) (models.Bid, error) {
	const op = "Rollback.SwapBid"

	log := r.log.With(
		slog.String("op", op),
		slog.String("id", bidId.String()),
		slog.Int("version", int(version)),
	)

	// Save outdated tender.
	if err := r.rollbackStorage.SaveBid(ctx, outdatedBid); err != nil {
		log.Error("failed to save outdated tender", sl.Err(err))
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	// Recover old bid.
	oldBid, err := r.rollbackStorage.RecoverBid(ctx, bidId, version)
	if err != nil {
		if errors.Is(err, storage.ErrVersionNotFound) {
			log.Warn("version not found")
			return models.Bid{}, service.ErrVersionNotFound
		}
		log.Error("failed to restore bid", sl.Err(err))
		return models.Bid{}, fmt.Errorf("%s: %w", op, err)
	}

	return oldBid, nil
}
