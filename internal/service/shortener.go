package service

import (
	"context"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository"
)

type ShortenerService struct {
	repo      repository.LinkRepository
	generator CodeGenerator
}

func NewShortenerService(repo repository.LinkRepository, generator CodeGenerator) *ShortenerService {
	return &ShortenerService{
		repo:      repo,
		generator: generator,
	}
}

func (s *ShortenerService) Create(ctx context.Context, originalURL string) (domain.Link, bool, error) {
	panic("not implemented")
}

func (s *ShortenerService) GetByShort(ctx context.Context, shortCode string) (domain.Link, error) {
	return s.repo.GetByShort(ctx, shortCode)
}