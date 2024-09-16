package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"tender/internal/app"
	"tender/internal/config"
	"tender/internal/lib/logger/slogpretty"
)

func main() {
	// Setup config.
	cfg := config.MustLoad()

	// Setup logger.
	log := setupLogger(cfg.PrettyLogger)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	// Initialize app.
	httpApplication := app.New(
		log,
		cfg.Addr,
		cfg.OpenapiPath,
		cfg.Timeout,
		cfg.IdleTimeout,
		cfg.PostgresConn,
	)

	// Run server.
	go httpApplication.Router.MustRun()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)

	<-stop

	// Stop application.
	httpApplication.Router.Stop()
	httpApplication.Storage.Postgres.Stop()
	log.Info("Gracefully stopped")
}

func setupLogger(prettyLogger bool) *slog.Logger {
	var log *slog.Logger

	if prettyLogger {
		log = setupPrettySlog()
	} else {
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
