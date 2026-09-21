package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository"
)

var _ repository.LinkRepository = (*Repository)(nil)

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
	if err := ctx.Err(); err != nil {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", err)
	}

	originalURL, ok := r.byShort[shortCode]
	if !ok {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", domain.ErrNotFound)
	}

	return domain.Link{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
	}, nil
}

func (r *Repository) GetByOriginal(ctx context.Context, originalURL string) (domain.Link, error) {
	if err := ctx.Err(); err != nil {
		return domain.Link{}, fmt.Errorf("get link by original URL: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return domain.Link{}, fmt.Errorf("get link by original URL: %w", err)
	}

	shortCode, ok := r.byOriginal[originalURL]
	if !ok {
		return domain.Link{}, fmt.Errorf("get link by original URL: %w", domain.ErrNotFound)
	}

	return domain.Link{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
	}, nil
}

func (r *Repository) Create(ctx context.Context, link domain.Link) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create link: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create link: %w", err)
	}

	if _, exists := r.byOriginal[link.OriginalURL]; exists {
		return fmt.Errorf("create link: %w", domain.ErrOriginalURLExists)
	}
	if _, exists := r.byShort[link.ShortCode]; exists {
		return fmt.Errorf("create link: %w", domain.ErrShortCodeExists)
	}

	if r.byShort == nil {
		r.byShort = make(map[string]string)
	}
	if r.byOriginal == nil {
		r.byOriginal = make(map[string]string)
	}

	r.byShort[link.ShortCode] = link.OriginalURL
	r.byOriginal[link.OriginalURL] = link.ShortCode

	return nil
}
