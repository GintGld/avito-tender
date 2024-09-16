package controller

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	valid "tender/internal/lib/validate"
	"tender/internal/models"
	"tender/internal/service"
)

func New(
	Timeout time.Duration,
	tender Tender,
) *fiber.App {
	ctr := tenderController{
		Timeout: Timeout,
		tender:  tender,
	}

	app := fiber.New()

	// Group 02/tenders/new
	app.Post("/new", ctr.new)

	// Group 03/tenders/list
	app.Get("/", ctr.all)
	app.Get("/my", ctr.my)

	// Group 04/tenders/status
	app.Get("/:tenderId/status", ctr.status)
	app.Put("/:tenderId/status", ctr.statusUpd)

	// Group 05/tenders/version
	app.Patch("/:tenderId/edit", ctr.edit)
	app.Put("/:tenderId/rollback/:version", ctr.rollback)

	return app
}

type tenderController struct {
	Timeout time.Duration
	tender  Tender
}

//go:generate go run github.com/vektra/mockery/v2@v2.45.1 --name Tender
type Tender interface {
	New(context.Context, models.TenderNew) (models.TenderOut, error)
	All(ctx context.Context, limit, offset int32, services []models.ServiceType) ([]models.TenderOut, error)
	My(ctx context.Context, limit, offset int32, username string) ([]models.TenderOut, error)
	Status(ctx context.Context, username string, tenderId uuid.UUID) (models.TenderStatus, error)
	SetStatus(ctx context.Context, username string, tenderId uuid.UUID, status models.TenderStatus) (models.TenderOut, error)
	Edit(ctx context.Context, username string, tenderId uuid.UUID, patch models.TenderPatch) (models.TenderOut, error)
	Rollback(ctx context.Context, username string, tenderId uuid.UUID, version int32) (models.TenderOut, error)
}

// new creates new tender.
func (t *tenderController) new(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	var tenderNew models.TenderNew

	if err := c.BodyParser(&tenderNew); err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			if parseErr.UserCaused {
				return c.Status(fiber.StatusUnauthorized).JSON(parseErr.Response())
			}
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid json"))
	}

	res, err := t.tender.New(ctx, tenderNew)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// all returns all public tenders.
func (t *tenderController) all(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	var services []models.ServiceType
	if s := c.Query("service_type"); s != "" {
		splitted := strings.Split(s, ",")
		services = make([]models.ServiceType, 0, len(splitted))
		for _, el := range splitted {
			t, err := models.StrToServiceType(el)
			if err != nil {
				var parseErr *models.Error
				if errors.As(err, &parseErr) {
					return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
				}
			}
			services = append(services, t)
		}
	}

	limit := int32(c.QueryInt("limit", 5))
	offset := int32(c.QueryInt("offset", 0))

	res, err := t.tender.All(ctx, limit, offset, services)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// user returns all user's tenders.
func (t *tenderController) my(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	limit := int32(c.QueryInt("limit", 5))
	offset := int32(c.QueryInt("offset", 0))
	username := c.Query("username")

	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	res, err := t.tender.My(ctx, limit, offset, username)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		c.SendStatus(fiber.StatusInternalServerError)
	}

	if res == nil {
		res = []models.TenderOut{}
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// status returns tender's status.
func (t *tenderController) status(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	res, err := t.tender.Status(ctx, username, tenderId)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrTenderNotFound) {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("tender not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).SendString(string(res))
}

// status updates tender's status.
func (t *tenderController) statusUpd(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	status, err := models.StrToTenderStatus(c.Query("status"))
	if err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
	}

	res, err := t.tender.SetStatus(ctx, username, tenderId, status)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
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

// edit update tender.
func (t *tenderController) edit(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	var patch models.TenderPatch

	if err := c.BodyParser(&patch); err != nil {
		var parseErr *models.Error
		if errors.As(err, &parseErr) {
			return c.Status(fiber.StatusBadRequest).JSON(parseErr.Response())
		}
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid json"))
	}

	res, err := t.tender.Edit(ctx, username, tenderId, patch)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrTenderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("tender not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

// rollback rollbacks tender to previous version.
func (t *tenderController) rollback(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	username := c.Query("username")
	if err := valid.Validate(username, "username", 100); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp(err.Error()))
	}

	tenderId, err := uuid.Parse(c.Params("tenderId"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("invalid tender id"))
	}

	versionInt64, err := strconv.ParseInt(c.Params("version"), 10, 32)
	if err != nil {
		c.Status(fiber.StatusBadRequest).JSON(models.ErrorResp("invalid version"))
	}

	res, err := t.tender.Rollback(ctx, username, tenderId, int32(versionInt64))
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResp("user not found"))
		}
		if errors.Is(err, service.ErrNotEnoughPrivileges) {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResp("unallowed action"))
		}
		if errors.Is(err, service.ErrTenderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("tender not found"))
		}
		if errors.Is(err, service.ErrVersionNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(models.ErrorResp("bid not found"))
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Status(fiber.StatusOK).JSON(res)
}
