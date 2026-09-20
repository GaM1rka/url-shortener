package repository

import (
	"context"

	"github.com/GaM1rka/url-shortener/internal/domain"
)

type LinkRepository interface {
	GetByShort(
		ctx context.Context,
		shortCode string,
	) (domain.Link, error)

	GetByOriginal(
		ctx context.Context,
		originalURL string,
	) (domain.Link, error)

	Create(
		ctx context.Context,
		link domain.Link,
	) error
}