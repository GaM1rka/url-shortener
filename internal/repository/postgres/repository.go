package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository"
	"github.com/GaM1rka/url-shortener/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

const startupTimeout = 30 * time.Second

var _ repository.LinkRepository = (*Repository)(nil)

type Repository struct {
	pool *pgxpool.Pool
}

func New(dsn string) (*Repository, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	if err := runMigrations(ctx, poolConfig); err != nil {
		return nil, fmt.Errorf("run PostgreSQL migrations: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) GetByShort(ctx context.Context, shortCode string) (domain.Link, error) {
	var link domain.Link
	err := r.pool.QueryRow(ctx, getByShortQuery, shortCode).Scan(
		&link.ShortCode,
		&link.OriginalURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", err)
	}

	return link, nil
}

func (r *Repository) GetByOriginal(ctx context.Context, originalURL string) (domain.Link, error) {
	var link domain.Link
	err := r.pool.QueryRow(ctx, getByOriginalQuery, originalURL).Scan(
		&link.ShortCode,
		&link.OriginalURL,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Link{}, fmt.Errorf("get link by original URL: %w", domain.ErrNotFound)
	}
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link by original URL: %w", err)
	}

	return link, nil
}

func (r *Repository) Create(ctx context.Context, link domain.Link) error {
	result, err := r.pool.Exec(ctx, createLinkQuery, link.ShortCode, link.OriginalURL)
	if err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	if result.RowsAffected() == 1 {
		return nil
	}

	var originalURLExists bool
	var shortCodeExists bool
	err = r.pool.QueryRow(
		ctx,
		getConflictQuery,
		link.OriginalURL,
		link.ShortCode,
	).Scan(&originalURLExists, &shortCodeExists)
	if err != nil {
		return fmt.Errorf("classify link conflict: %w", err)
	}

	if originalURLExists {
		return fmt.Errorf("create link: %w", domain.ErrOriginalURLExists)
	}
	if shortCodeExists {
		return fmt.Errorf("create link: %w", domain.ErrShortCodeExists)
	}

	return errors.New("create link: insert was skipped without a unique conflict")
}

func (r *Repository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

func runMigrations(ctx context.Context, poolConfig *pgxpool.Config) error {
	db := stdlib.OpenDB(*poolConfig.ConnConfig)
	defer db.Close()

	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrations.FS,
		goose.WithSessionLocker(sessionLocker),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
