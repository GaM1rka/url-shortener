package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository"
)

const maxCreateAttempts = 16

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
	if err := ctx.Err(); err != nil {
		return domain.Link{}, false, err
	}
	if !isValidOriginalURL(originalURL) {
		return domain.Link{}, false, domain.ErrInvalidURL
	}

	existing, err := s.repo.GetByOriginal(ctx, originalURL)
	switch {
	case err == nil:
		return existing, false, nil
	case !errors.Is(err, domain.ErrNotFound):
		return domain.Link{}, false, fmt.Errorf("find link by original URL: %w", err)
	}

	for attempt := 0; attempt < maxCreateAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return domain.Link{}, false, err
		}

		shortCode, err := s.generator.Generate()
		if err != nil {
			return domain.Link{}, false, fmt.Errorf("generate short code: %w", err)
		}
		if !isValidShortCode(shortCode) {
			return domain.Link{}, false, fmt.Errorf("%w: generator returned %q", domain.ErrInvalidShortCode, shortCode)
		}

		link := domain.Link{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
		}
		err = s.repo.Create(ctx, link)
		switch {
		case err == nil:
			return link, true, nil

		case errors.Is(err, domain.ErrOriginalURLExists):
			existing, getErr := s.repo.GetByOriginal(ctx, originalURL)
			if getErr != nil {
				return domain.Link{}, false, fmt.Errorf("get concurrently created link: %w", getErr)
			}
			return existing, false, nil

		case errors.Is(err, domain.ErrShortCodeExists):
			continue

		default:
			return domain.Link{}, false, fmt.Errorf("store link: %w", err)
		}
	}

	return domain.Link{}, false, fmt.Errorf("generate unique short code: exhausted %d attempts", maxCreateAttempts)
}

func (s *ShortenerService) GetByShort(ctx context.Context, shortCode string) (domain.Link, error) {
	if err := ctx.Err(); err != nil {
		return domain.Link{}, err
	}
	if !isValidShortCode(shortCode) {
		return domain.Link{}, domain.ErrInvalidShortCode
	}

	link, err := s.repo.GetByShort(ctx, shortCode)
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link by short code: %w", err)
	}

	return link, nil
}

func isValidOriginalURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") &&
		parsed.IsAbs() && parsed.Host != "" && parsed.Hostname() != ""
}

func isValidShortCode(value string) bool {
	if len(value) != shortCodeLength {
		return false
	}

	for i := 0; i < len(value); i++ {
		char := value[i]
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return false
		}
	}

	return true
}
