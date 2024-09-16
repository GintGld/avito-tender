package controller

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	valid "tender/internal/lib/validate"
	"tender/internal/models"
	"tender/internal/service"
)

func New(
	ErrTimeout time.Duration,
	bid Bid,
) *fiber.App {
	ctr := bidController{
		ErrTimeout: ErrTimeout,
		bid:        bid,
	}

	app := fiber.New()

	// Group 06/bids/new
	app.Post("/new", ctr.new)

	// Group 07/bids/decision
	app.Put("/:bidId/submit_decision", ctr.decision)

	// Group 08/bids/list
	app.Get("/:tenderId/list", ctr.list)
	app.Get("/my", ctr.my)

	// Group 09/bids/status
	app.Get("/:bidId/status", ctr.status)
	app.Put("/:bidId/status", ctr.statusUpd)

	// Group 10/bids/version
	app.Patch("/:bidId/edit", ctr.edit)
	app.Put("/:bidId/rollback/:version", ctr.rollback)

	// Group 11/bids/reviews
	app.Get("/:tenderId/reviews", ctr.reviews)
	app.Put("/:bidId/feedback", ctr.feedback)

	return app
}

type bidController struct {
	ErrTimeout time.Duration
	bid        Bid
}

type Bid interface {
	New(context.Context, models.BidNew) (models.BidOut, error)
	SubmitDecision(ctx context.Context, username string, bidId uuid.UUID, decision models.DecisionType) (models.BidOut, error)
	List(ctx context.Context, username string, tenderId uuid.UUID, limit, offset int32) ([]models.BidOut, error)
	My(ctx context.Context, username string, limit, offset int32) ([]models.BidOut, error)
	Status(ctx context.Context, username string, bidId uuid.UUID) (models.BidStatus, error)
	SetStatus(ctx context.Context, username string, bidId uuid.UUID, status models.BidStatus) (models.BidOut, error)
	Edit(ctx context.Context, username string, bidId uuid.UUID, patch models.BidPatch) (models.BidOut, error)
	Rollback(ctx context.Context, username string, bidId uuid.UUID, version int32) (models.BidOut, error)
	Reviews(ctx context.Context, requester, author string, tenderId uuid.UUID, limit, offset int32) ([]models.ReviewOut, error)
	Feedback(ctx context.Context, username string, bidId uuid.UUID, feedback string) (models.BidOut, error)
}

func (b *bidController) new(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	var bidNew models.BidNew

	if err := c.BodyParser(&bidNew); err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			if parseErr.UserCaused {
				return c.Status(fiber.StatusUnauthorized).JSON(parseErr.Response())
			}
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid json"))
	}

	res, err := b.bid.New(ctx, bidNew)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) decision(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	desicion, err := models.StrToDecision(c.Query("decision"))
	if err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	res, err := b.bid.SubmitDecision(ctx, username, bidId, desicion)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bid not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action for user"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) list(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	limit := int32(c.QueryInt("limit", 5))
	offset := int32(c.QueryInt("offset", 0))

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	res, err := b.bid.List(ctx, username, tenderId, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bids not found"))
		}
		if errors.Is(err, service.ErrTenderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("tender not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action for user"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) my(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	limit := int32(c.QueryInt("limit", 5))
	offset := int32(c.QueryInt("offset", 0))
	username := c.Query("username")

	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	res, err := b.bid.My(ctx, username, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) status(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	res, err := b.bid.Status(ctx, username, bidId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action for user"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) statusUpd(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	status, err := models.StrToBidStatus(c.Query("status"))
	if err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
	}

	res, err := b.bid.SetStatus(ctx, username, bidId, status)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bid not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action for user"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) edit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	var patch models.BidPatch

	if err := c.BodyParser(&patch); err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid json"))
	}

	res, err := b.bid.Edit(ctx, username, bidId, patch)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("bid not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action for user"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) rollback(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	versionInt64, err := strconv.ParseInt(c.Params("version"), 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid version"))
	}

	res, err := b.bid.Rollback(ctx, username, bidId, int32(versionInt64))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bid not found"))
		}
		if errors.Is(err, service.ErrVersionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("version not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) reviews(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	limit := int32(c.QueryInt("limit", 5))
	offset := int32(c.QueryInt("offset", 0))

	authorUsername := c.Query("authorUsername")
	if err := valid.Validate(authorUsername, "author username", 100); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp(err.Error()))
	}

	requesterUsername := c.Query("requesterUsername")
	if err := valid.Validate(requesterUsername, "requester username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	res, err := b.bid.Reviews(ctx, requesterUsername, authorUsername, tenderId, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrAuthorNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("author not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		if errors.Is(err, service.ErrTenderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("tender not found"))
		}
		if errors.Is(err, service.ErrReviewsNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("reviews not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (b *bidController) feedback(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.ErrTimeout)
	defer cancel()

	bidFeedback := c.Query("bidFeedback")
	if err := valid.Validate(bidFeedback, "bid feedback", 1000); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp(err.Error()))

	}
	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	bidId, err := uuid.Parse(c.Params("bidId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid bid id"))
	}

	res, err := b.bid.Feedback(ctx, username, bidId, bidFeedback)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		if errors.Is(err, service.ErrBidNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bid not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}
