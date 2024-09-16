package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func New(
	ErrTimeout time.Duration,
) *fiber.App {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).Send([]byte("OK"))
	})

	return app
}
