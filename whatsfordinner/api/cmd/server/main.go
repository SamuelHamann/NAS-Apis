// Command server is the entry point for the whatsfordinner API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/config"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/database"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/server"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

// run wires the application together and blocks until the server is shut down.
// It is kept separate from main so the startup logic stays testable.
func run() error {
	// Best-effort load of a local .env file. In production the file is absent
	// and real environment variables are used instead.
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()

	pool, err := database.NewPool(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	logger.Info("connected to database")

	st := store.New(pool)
	srv := server.New(cfg, logger, st)

	// Background job resolving scanned receipt items against OpenFoodFacts
	// (see internal/worker). Stopped via workerCancel on shutdown, below.
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go worker.New(st, logger, cfg.QueueWorkerInterval, cfg.QueueWorkerBatchSize, cfg.OpenFoodFactsLocale).Run(workerCtx)

	httpServer := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      srv.Routes(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Run the server in its own goroutine so we can listen for OS signals.
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.Addr())
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Block until we receive a shutdown signal or the server fails.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	case sig := <-shutdown:
		logger.Info("shutdown started", "signal", sig.String())
		defer logger.Info("shutdown complete")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}
