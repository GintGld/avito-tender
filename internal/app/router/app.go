package app

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	bidCtr "tender/internal/controller/bid"
	pingCtr "tender/internal/controller/ping"
	tenderCtr "tender/internal/controller/tender"

	bidSrv "tender/internal/service/bid"
	rollbackSrv "tender/internal/service/rollback"
	tenderSrv "tender/internal/service/tender"
	userSrv "tender/internal/service/user"
)

type App struct {
	log      *slog.Logger
	addr     string
	fiberApp *fiber.App
}

func New(
	log *slog.Logger,
	addr string,
	openapiPath string,
	Timeout time.Duration,
	idleTimeout time.Duration,
	userStorage userSrv.EmployeeStorage,
	tenderStorage tenderSrv.TenderStorage,
	bidStorage bidSrv.BidStorage,
	rollbackStorage rollbackSrv.RollbackStorage,
) *App {
	// Initialize services.
	user := userSrv.New(
		log,
		userStorage,
	)
	rollback := rollbackSrv.New(
		log,
		rollbackStorage,
	)
	tender := tenderSrv.New(
		log,
		user,
		rollback,
		tenderStorage,
	)
	bid := bidSrv.New(
		log,
		user,
		tender,
		rollback,
		bidStorage,
	)

	// Initialize fiber router.
	fiberApp := fiber.New(fiber.Config{
		IdleTimeout: idleTimeout,
		JSONDecoder: decode,
	})

	// Mount controllers.
	fiberApp.Mount("/api/ping", pingCtr.New(Timeout))
	fiberApp.Mount("/api/tenders", tenderCtr.New(Timeout, tender))
	fiberApp.Mount("/api/bids", bidCtr.New(Timeout, bid))

	// Handler for openapi specification.
	fiberApp.Get("/api/openapi", func(c *fiber.Ctx) error {
		return c.SendFile(openapiPath)
	})

	return &App{
		log:      log,
		addr:     addr,
		fiberApp: fiberApp,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	return a.fiberApp.Listen(a.addr)
}

func (a *App) Stop() error {
	return a.fiberApp.Shutdown()
}

// JSON decoder function for fiber app.
func decode(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
