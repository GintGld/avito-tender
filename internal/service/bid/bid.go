package bid

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

type Bid struct {
	log         *slog.Logger
	userSrv     UserService
	tenderSrv   TenderService
	rollbackSrv RollbackService
	bidStorage  BidStorage
}

func New(
	log *slog.Logger,
	userSrv UserService,
	tenderSrv TenderService,
	rollbackSrv RollbackService,
	bidStorage BidStorage,
) *Bid {
	return &Bid{
		log:         log,
		userSrv:     userSrv,
		tenderSrv:   tenderSrv,
		rollbackSrv: rollbackSrv,
		bidStorage:  bidStorage,
	}
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name UserService
type UserService interface {
	Validate(ctx context.Context, username string) error
	ValidateUserId(ctx context.Context, userId uuid.UUID) error
	ValidateOrgId(ctx context.Context, orgId uuid.UUID) error
	UserId(ctx context.Context, username string) (uuid.UUID, error)
	Permission(ctx context.Context, username string, orgId uuid.UUID) error
	OrgSize(ctx context.Context, orgId uuid.UUID) (int64, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name TenderService
type TenderService interface {
	Tender(ctx context.Context, id uuid.UUID) (models.Tender, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name RollbackService
type RollbackService interface {
	SaveBid(ctx context.Context, bid models.Bid) error
	// Save outdated bid and recover old bid.
	SwapBid(ctx context.Context, bidId uuid.UUID, version int32, outdatedBid models.Bid) (models.Bid, error)
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name BidStorage
type BidStorage interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	InsertBid(ctx context.Context, bid models.Bid) (models.Bid, error)
	Bid(ctx context.Context, bidId uuid.UUID) (models.Bid, error)
	UpdateBid(ctx context.Context, bid models.Bid) error
	TenderBids(ctx context.Context, tenderId uuid.UUID, limit, offset int32) ([]models.Bid, error)
	UserBids(ctx context.Context, username string, limit, offset int32) ([]models.Bid, error)
	BidSetStatus(ctx context.Context, bidId uuid.UUID, status models.BidStatus) (models.Bid, error)

	InsertReview(ctx context.Context, review models.Review) (uuid.UUID, error)
	Reviews(ctx context.Context, tenderId uuid.UUID, author string, limit, offset int32) ([]models.Review, error)

	InsertDecision(ctx context.Context, decision models.Decision) error
	Decisions(ctx context.Context, bidId uuid.UUID) ([]models.Decision, error)
}

const (
	QUORUM_SIZE = 3
)

// New inserts new bid.
func (b *Bid) New(ctx context.Context, bidNew models.BidNew) (models.BidOut, error) {
	const op = "Bid.New"

	log := b.log.With(
		slog.String("op", op),
		slog.String("creator", bidNew.AuthorId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Create bid with version=1.
	bid := bidNew.ToBid()

	// Check if user/org exists.
	switch bidNew.AuthorType {
	case models.User:
		if err := b.userSrv.ValidateUserId(ctx, bidNew.AuthorId); err != nil {
			if errors.Is(err, service.ErrUserNotFound) {
				log.Warn("user not found")
				return models.BidOut{}, service.ErrUserNotFound
			}
			log.Error("failed to verify user", sl.Err(err))
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
	case models.Organization:
		if err := b.userSrv.ValidateOrgId(ctx, bidNew.AuthorId); err != nil {
			if errors.Is(err, service.ErrOrganizationNotFound) {
				log.Warn("organization not found")
				return models.BidOut{}, service.ErrOrganizationNotFound
			}
			log.Error("failed to verify organization", sl.Err(err))
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	// Insert bid.
	bid, err = b.bidStorage.InsertBid(ctx, bid)
	if err != nil {
		log.Error("failed to insert bid", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid.ToOut(), nil
}

// SubmitDecision submits decision.
// Closes bid.
func (b *Bid) SubmitDecision(ctx context.Context, username string, bidId uuid.UUID, decision models.DecisionType) (models.BidOut, error) {
	const op = "Bid.SubmitDecision"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("bid id", bidId.String()),
		slog.String("decision", string(decision)),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get bid.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get bid's tender
	tender, err := b.tenderSrv.Tender(ctx, bid.TenderId)
	if err != nil {
		if errors.Is(err, service.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to modify tender info.
	if err := b.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("user not allowed")
			return models.BidOut{}, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check permission", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get user id.
	userId, err := b.userSrv.UserId(ctx, username)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, service.ErrUserNotFound
		}
		log.Error("failed to get user id")
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save decision.
	if err := b.bidStorage.InsertDecision(ctx, models.Decision{
		UserId:   userId,
		BidId:    bid.Id,
		Decision: decision,
	}); err != nil {
		log.Error("failed to insert decision")
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get all decisions for bid.
	decisions, err := b.bidStorage.Decisions(ctx, bidId)
	if err != nil {
		log.Error("failed to get bid's decision", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get organization size.
	orgSize, err := b.userSrv.OrgSize(ctx, tender.OrgId)
	if err != nil {
		if errors.Is(err, service.ErrOrganizationNotFound) {
			log.Warn("org not found")
			return models.BidOut{}, service.ErrOrganizationNotFound
		}
		log.Error("failed to get org size", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Determine minimum required approves.
	required_approves := min(orgSize, QUORUM_SIZE)

	// Summary decision.
	summary := models.DecisionType("null")
	approve_counter := 0
loop:
	for _, d := range decisions {
		switch d.Decision {
		case models.Approved:
			approve_counter++
		case models.Rejected:
			summary = models.Rejected
			break loop
		}
	}
	if summary != models.Rejected && approve_counter >= int(required_approves) {
		summary = models.Approved
	}

	// check if decision wac conclusive or not.
	if summary == models.DecisionType("null") {
		log.Info("inconclusive decision")
		return bid.ToOut(), nil
	}
	log.Info("conclusive decision", slog.String("decision", string(summary)))

	// Set bid status to cancel if it was rejected or approved by quorum.
	bid.Status = models.BidCanceled
	if err := b.bidStorage.UpdateBid(ctx, bid); err != nil {
		log.Error("failed to update bid status")
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid.ToOut(), nil
}

// List returns bids related to tender.
func (b *Bid) List(ctx context.Context, username string, tenderId uuid.UUID, limit, offset int32) ([]models.BidOut, error) {
	const op = "Bid.List"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return nil, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Check if tender exists.
	if _, err := b.tenderSrv.Tender(ctx, tenderId); err != nil {
		if errors.Is(err, service.ErrTenderNotFound) {
			log.Warn("tender not found")
			return nil, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get tender's bids.
	res, err := b.bidStorage.TenderBids(ctx, tenderId, limit, offset)
	if err != nil {
		log.Error("failed to get tender's bids", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Convert slice elements.
	out := make([]models.BidOut, 0, len(res))
	for i := range res {
		out = append(out, res[i].ToOut())
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, nil
}

// My returns user's bids.
func (b *Bid) My(ctx context.Context, username string, limit, offset int32) ([]models.BidOut, error) {
	const op = "Bid.My"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return nil, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get user's bids.
	res, err := b.bidStorage.UserBids(ctx, username, limit, offset)
	if err != nil {
		log.Error("failed to get tenders", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Convert slice elements.
	out := make([]models.BidOut, 0, len(res))
	for i := range res {
		out = append(out, res[i].ToOut())
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, nil
}

// BidStatus return bid status.
func (b *Bid) Status(ctx context.Context, username string, bidId uuid.UUID) (models.BidStatus, error) {
	const op = "Bid.BidStatus"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", bidId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return "", err
		}
		log.Error("failed to verify user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	// Get bid.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("bid not found")
			return "", service.ErrBidNotFound
		}
		log.Error("failed to get bid status", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	// Check if user/org is allowed to modify bid.
	switch bid.AuthorType {
	case models.User:
		userId, err := b.userSrv.UserId(ctx, username)
		if err != nil {
			log.Error("failed to get user's id", sl.Err(err))
			return "", fmt.Errorf("%s: %w", op, err)
		}
		if userId != bid.AuthorId {
			log.Warn("user not allowed to modify this bid")
			return "", service.ErrNotEnoughPrivileges
		}
	case models.Organization:
		if err := b.userSrv.Permission(ctx, username, bid.AuthorId); err != nil {
			if errors.Is(err, service.ErrNotEnoughPrivileges) {
				log.Warn("unallowed to modify")
				return "", service.ErrNotEnoughPrivileges
			}
			log.Error("failed to check user permission")
			return "", fmt.Errorf("%s: %w", op, err)
		}
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return bid.Status, nil
}

// BidSetStatus updates bid status.
func (b *Bid) SetStatus(ctx context.Context, username string, bidId uuid.UUID, status models.BidStatus) (models.BidOut, error) {
	const op = "Bid.BidSetStatus"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", bidId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get bid.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user/org is allowed to modify bid.
	switch bid.AuthorType {
	case models.User:
		userId, err := b.userSrv.UserId(ctx, username)
		if err != nil {
			log.Error("failed to get user's id", sl.Err(err))
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
		if userId != bid.AuthorId {
			log.Warn("user not allowed to modify this bid")
			return models.BidOut{}, service.ErrNotEnoughPrivileges
		}
	case models.Organization:
		if err := b.userSrv.Permission(ctx, username, bid.AuthorId); err != nil {
			if errors.Is(err, service.ErrNotEnoughPrivileges) {
				log.Warn("unallowed to modify")
				return models.BidOut{}, service.ErrNotEnoughPrivileges
			}
			log.Error("failed to check user permission")
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	// Update tender status.
	bid, err = b.bidStorage.BidSetStatus(ctx, bidId, status)
	if err != nil {
		log.Error("failed to update bid status", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid.ToOut(), nil
}

// Edit edits bid.
func (b *Bid) Edit(ctx context.Context, username string, bidId uuid.UUID, patch models.BidPatch) (models.BidOut, error) {
	const op = "Bid.Edit"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", bidId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get tender.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user/org is allowed to modify bid.
	switch bid.AuthorType {
	case models.User:
		userId, err := b.userSrv.UserId(ctx, username)
		if err != nil {
			log.Error("failed to get user's id", sl.Err(err))
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
		if userId != bid.AuthorId {
			log.Warn("user not allowed to modify this bid")
			return models.BidOut{}, service.ErrNotEnoughPrivileges
		}
	case models.Organization:
		if err := b.userSrv.Permission(ctx, username, bid.AuthorId); err != nil {
			if errors.Is(err, service.ErrNotEnoughPrivileges) {
				log.Warn("unallowed to modify")
				return models.BidOut{}, service.ErrNotEnoughPrivileges
			}
			log.Error("failed to check user permission")
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	// Apply patch.
	newBid := bid
	newBid.Patch(patch)
	newBid.Version += 1

	// Update bid.
	if err := b.bidStorage.UpdateBid(ctx, newBid); err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("bid not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to updated bid", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save old version of bid.
	if err := b.rollbackSrv.SaveBid(ctx, bid); err != nil {
		log.Error("failed to insert bid", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return newBid.ToOut(), nil
}

// Rollback rollbacks old version of bid.
func (b *Bid) Rollback(ctx context.Context, username string, bidId uuid.UUID, version int32) (models.BidOut, error) {
	const op = "Tender.Rollback"

	log := b.log.With(
		slog.String("op", op),
		slog.String("user", username),
		slog.String("id", bidId.String()),
		slog.Int("version", int(version)),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get actual tender.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user/org is allowed to modify bid.
	switch bid.AuthorType {
	case models.User:
		userId, err := b.userSrv.UserId(ctx, username)
		if err != nil {
			log.Error("failed to get user's id", sl.Err(err))
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
		if userId != bid.AuthorId {
			log.Warn("user not allowed to modify this bid")
			return models.BidOut{}, service.ErrNotEnoughPrivileges
		}
	case models.Organization:
		if err := b.userSrv.Permission(ctx, username, bid.AuthorId); err != nil {
			if errors.Is(err, service.ErrNotEnoughPrivileges) {
				log.Warn("unallowed to modify")
				return models.BidOut{}, service.ErrNotEnoughPrivileges
			}
			log.Error("failed to check user permission")
			return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	// Save outdated tender and recover old tender.
	recoveredBid, err := b.rollbackSrv.SwapBid(ctx, bidId, version, bid)
	if err != nil {
		if errors.Is(err, service.ErrVersionNotFound) {
			log.Warn("version not found")
			return models.BidOut{}, service.ErrVersionNotFound
		}
		log.Error("failed to recover old version", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Save recovered tender.
	recoveredBid.Version = bid.Version + 1
	recoveredBid.Status = bid.Status
	newTender, err := b.bidStorage.InsertBid(ctx, recoveredBid)
	if err != nil {
		log.Error("failed to insert tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return newTender.ToOut(), nil
}

// Reviews returns
func (b *Bid) Reviews(ctx context.Context, requester, author string, tenderId uuid.UUID, limit, offset int32) ([]models.ReviewOut, error) {
	const op = "Bid.Reviews"

	log := b.log.With(
		slog.String("op", op),
		slog.String("requester", requester),
		slog.String("author", author),
		slog.String("tender id", tenderId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if requester exists
	if err := b.userSrv.Validate(ctx, requester); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return nil, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	// Check if author exists
	if err := b.userSrv.Validate(ctx, author); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return nil, service.ErrAuthorNotFound
		}
		log.Error("failed to verify user", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get bid's tender.
	tender, err := b.tenderSrv.Tender(ctx, tenderId)
	if err != nil {
		if errors.Is(err, service.ErrTenderNotFound) {
			log.Warn("tender not found")
			return nil, service.ErrTenderNotFound
		}
		log.Error("failed to get tender")
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to view tender's feedbacks.
	if err := b.userSrv.Permission(ctx, requester, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return nil, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission")
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Get reviews.
	res, err := b.bidStorage.Reviews(ctx, tenderId, author, limit, offset)
	if err != nil {
		log.Error("failed to get reviews", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Convert slice's elements.
	out := make([]models.ReviewOut, 0, len(res))
	for i := range res {
		out = append(out, res[i].ToOut())
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, nil
}

// Feedback creates feedback for a bid.
// If user is not allowed returnes error.
func (b *Bid) Feedback(ctx context.Context, username string, bidId uuid.UUID, feedback string) (models.BidOut, error) {
	const op = "Bid.Feedback"

	log := b.log.With(
		slog.String("op", op),
		slog.String("username", username),
		slog.String("id", bidId.String()),
	)

	ctx, err := b.bidStorage.Begin(ctx)
	if err != nil {
		log.Error("failed to start tx", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}
	defer func() {
		if err := b.bidStorage.Rollback(ctx); err != nil {
			log.Error("failed to rollback", sl.Err(err))
		}
	}()

	// Check if user exists
	if err := b.userSrv.Validate(ctx, username); err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			log.Warn("user not found")
			return models.BidOut{}, err
		}
		log.Error("failed to verify user", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Get bid.
	bid, err := b.bidStorage.Bid(ctx, bidId)
	if err != nil {
		if errors.Is(err, storage.ErrBidNotFound) {
			log.Warn("bid not found")
			return models.BidOut{}, service.ErrBidNotFound
		}
		log.Error("failed to get bid", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if tender exists.
	tender, err := b.tenderSrv.Tender(ctx, bid.TenderId)
	if err != nil {
		if errors.Is(err, service.ErrTenderNotFound) {
			log.Warn("tender not found")
			return models.BidOut{}, service.ErrTenderNotFound
		}
		log.Error("failed to get tender", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Check if user is allowed to modify tender.
	if err := b.userSrv.Permission(ctx, username, tender.OrgId); err != nil {
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			log.Warn("unallowed to modify")
			return models.BidOut{}, service.ErrNotEnoughPrivileges
		}
		log.Error("failed to check user permission")
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	// Create review.
	var review models.Review
	review.BidId = bid.Id
	review.Desc = feedback
	review.AuthorName = username

	// Insert review.
	if _, err := b.bidStorage.InsertReview(ctx, review); err != nil {
		log.Error("failed to insert review", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := b.bidStorage.Commit(ctx); err != nil {
		log.Error("failed to commit", sl.Err(err))
		return models.BidOut{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid.ToOut(), nil
}
