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
	"time"

	"github.com/GaM1rka/url-shortener/internal/config"
	"github.com/GaM1rka/url-shortener/internal/repository"
	"github.com/GaM1rka/url-shortener/internal/repository/memory"
	"github.com/GaM1rka/url-shortener/internal/repository/postgres"
	"github.com/GaM1rka/url-shortener/internal/service"
	transporthttp "github.com/GaM1rka/url-shortener/internal/transport/http"
)

type App struct {
	server          *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
	closeRepository func()
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo, closeRepository, err := buildRepository(cfg)
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
		Addr:              cfg.HTTP.Address,
		Handler:           handler.Routes(),
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return &App{
		server:          server,
		shutdownTimeout: cfg.HTTP.ShutdownTimeout,
		logger:          logger,
		closeRepository: closeRepository,
	}, nil
}

func buildRepository(cfg *config.Config) (repository.LinkRepository, func(), error) {
	switch cfg.StorageType {
	case config.StorageMemory:
		return memory.New(), func() {}, nil

	case config.StoragePostgres:
		repo, err := postgres.New(cfg.PostgresDSN)
		if err != nil {
			return nil, nil, err
		}
		return repo, repo.Close, nil

	default:
		return nil, nil, fmt.Errorf("unsupported storage type: %q", cfg.StorageType)
	}
}

func (a *App) Run() error {
	if a == nil || a.server == nil {
		return errors.New("application is not initialized")
	}
	if a.closeRepository != nil {
		defer a.closeRepository()
	}

	errCh := make(chan error, 1)

	go func() {
		a.logger.Info(
			"HTTP server started",
			"address", a.server.Addr,
		)
		errCh <- a.server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil

	case sig := <-sigCh:
		a.logger.Info("shutdown signal received",
			"signal", sig.String(),
		)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		_ = a.server.Close()
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	a.logger.Info("graceful shutdown completed")
	return nil
}
