package app

import (
	"log/slog"
	"time"

	storage "tender/internal/app/postgres"
	router "tender/internal/app/router"
	"tender/internal/lib/logger/sl"
)

type App struct {
	Router  *router.App
	Storage *storage.Storage
}

func New(
	log *slog.Logger,
	addr string,
	openapiPath string,
	Timeout time.Duration,
	idleTimeout time.Duration,
	postgresURL string,
) *App {
	storage, err := storage.New(postgresURL)
	if err != nil {
		log.Error("failed to create storage", sl.Err(err))
		panic(err)
	}

	router := router.New(
		log,
		addr,
		openapiPath,
		Timeout,
		idleTimeout,
		storage.Postgres,
		storage.Postgres,
		storage.Postgres,
		storage.Postgres,
	)

	return &App{
		Router:  router,
		Storage: storage,
	}
}
