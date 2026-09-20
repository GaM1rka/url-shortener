package postgres

import (
	"context"

	"github.com/GaM1rka/url-shortener/internal/domain"
)

type Repository struct {
}

func New(dsn string) (*Repository, error) {
	return &Repository{}, nil
}

func (r *Repository) GetByShort(ctx context.Context, shortCode string) (domain.Link, error) {
	panic("not implemented")
}

func (r *Repository) GetByOriginal(ctx context.Context, originalURL string) (domain.Link, error) {
	panic("not implemented")
}

func (r *Repository) Create(ctx context.Context, link domain.Link) error {
	return nil
}