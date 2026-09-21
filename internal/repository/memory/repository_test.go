package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/GaM1rka/url-shortener/internal/domain"
)

func TestRepositoryCreateAndGet(t *testing.T) {
	t.Parallel()

	repo := New()
	want := domain.Link{
		ShortCode:   "Ab3_dE9xY0",
		OriginalURL: "https://example.com/article?id=42",
	}

	if err := repo.Create(context.Background(), want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	byShort, err := repo.GetByShort(context.Background(), want.ShortCode)
	if err != nil {
		t.Fatalf("GetByShort() error = %v", err)
	}
	if byShort != want {
		t.Fatalf("GetByShort() = %+v, want %+v", byShort, want)
	}

	byOriginal, err := repo.GetByOriginal(context.Background(), want.OriginalURL)
	if err != nil {
		t.Fatalf("GetByOriginal() error = %v", err)
	}
	if byOriginal != want {
		t.Fatalf("GetByOriginal() = %+v, want %+v", byOriginal, want)
	}
}

func TestRepositoryNotFound(t *testing.T) {
	t.Parallel()

	repo := New()
	tests := []struct {
		name string
		get  func() error
	}{
		{
			name: "by short code",
			get: func() error {
				_, err := repo.GetByShort(context.Background(), "Ab3_dE9xY0")
				return err
			},
		},
		{
			name: "by original URL",
			get: func() error {
				_, err := repo.GetByOriginal(context.Background(), "https://example.com")
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.get(); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestRepositoryConflicts(t *testing.T) {
	t.Parallel()

	repo := New()
	stored := domain.Link{
		ShortCode:   "Ab3_dE9xY0",
		OriginalURL: "https://example.com/first",
	}
	if err := repo.Create(context.Background(), stored); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := repo.Create(context.Background(), domain.Link{
		ShortCode:   "1234567890",
		OriginalURL: stored.OriginalURL,
	})
	if !errors.Is(err, domain.ErrOriginalURLExists) {
		t.Fatalf("duplicate original URL error = %v, want ErrOriginalURLExists", err)
	}

	err = repo.Create(context.Background(), domain.Link{
		ShortCode:   stored.ShortCode,
		OriginalURL: "https://example.com/second",
	})
	if !errors.Is(err, domain.ErrShortCodeExists) {
		t.Fatalf("duplicate short code error = %v, want ErrShortCodeExists", err)
	}
}

func TestRepositoryConcurrentCreateSameOriginalURL(t *testing.T) {
	const requests = 100

	repo := New()
	ctx := context.Background()
	originalURL := "https://example.com/shared"

	var wg sync.WaitGroup
	errorsCh := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errorsCh <- repo.Create(ctx, domain.Link{
				ShortCode:   fmt.Sprintf("%010d", index),
				OriginalURL: originalURL,
			})
		}(i)
	}
	wg.Wait()
	close(errorsCh)

	created := 0
	for err := range errorsCh {
		switch {
		case err == nil:
			created++
		case errors.Is(err, domain.ErrOriginalURLExists):
		default:
			t.Fatalf("Create() unexpected error = %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("successful creates = %d, want 1", created)
	}

	if _, err := repo.GetByOriginal(ctx, originalURL); err != nil {
		t.Fatalf("GetByOriginal() error = %v", err)
	}
}

func TestRepositoryHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	repo := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := repo.Create(ctx, domain.Link{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetByShort(ctx, "Ab3_dE9xY0"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByShort() error = %v, want context.Canceled", err)
	}
	if _, err := repo.GetByOriginal(ctx, "https://example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByOriginal() error = %v, want context.Canceled", err)
	}
}
