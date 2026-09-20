package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GaM1rka/url-shortener/internal/config"
	"github.com/GaM1rka/url-shortener/internal/repository"
	"github.com/GaM1rka/url-shortener/internal/repository/memory"
	"github.com/GaM1rka/url-shortener/internal/repository/postgres"
	"github.com/GaM1rka/url-shortener/internal/service"
	transporthttp "github.com/GaM1rka/url-shortener/internal/transport/http"
)

type App struct {
	server          *http.Server
	shutdownTimeout config.Duration
	logger          *slog.Logger
}

func New(cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo, err := buildRepository(cfg)
	if err != nil {
		return nil, fmt.Errorf("build repository: %w", err)
	}

	generator := service.NewRandomGenerator()

	shortenerService := service.NewShortenerService(
		repo,
		generator,
	)

	handler := transporthttp.NewHandler(
		shortenerService,
		cfg.HTTP.BaseURL,
		logger,
	)

	server := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      handler.Routes(),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &App{
		server: server,
		logger: logger,
	}, nil
}

func buildRepository(cfg *config.Config) (repository.LinkRepository, error) {
	switch cfg.StorageType {
	case config.StorageMemory:
		return memory.New(), nil

	case config.StoragePostgres:
		return postgres.New(cfg.PostgresDSN)

	default:
		return nil, fmt.Errorf("unsupported storage type: %q", cfg.StorageType)
	}
}

func (a *App) Run() error {
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info("HTTP server started",
			"address", a.server.Addr,
		)

		errCh <- a.server.ListenAndServe()
	}().logger.Info("HTTP server started",
			"address", a.server.Addr,
		)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil

	case sig := <-sigCh:
		a.logger.Info("shutdown signal received",
			"signal", sig.String(),
		)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		config.RLogger.Printf("HTTP server shutdown error: %v", err)
	}

	a.logger.Info("Graceful shutdown cimpleted")
	return nil
}