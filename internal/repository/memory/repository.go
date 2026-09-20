package memory

import (
	"context"
	"sync"

	"github.com/GaM1rka/url-shortener/internal/domain"
)

type Repository struct {
	mu sync.RWMutex

	byShort    map[string]string
	byOriginal map[string]string
}

func New() *Repository {
	return &Repository{
		byShort:    make(map[string]string),
		byOriginal: make(map[string]string),
	}
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