package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/GaM1rka/url-shortener/internal/domain"
	"github.com/GaM1rka/url-shortener/internal/repository/memory"
)

type sequenceGenerator struct {
	mu    sync.Mutex
	codes []string
	next  int
	err   error
}

func (g *sequenceGenerator) Generate() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.err != nil {
		return "", g.err
	}
	if len(g.codes) == 0 {
		code := fmt.Sprintf("%010d", g.next)
		g.next++
		return code, nil
	}

	code := g.codes[g.next%len(g.codes)]
	g.next++
	return code, nil
}

func (g *sequenceGenerator) calls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.next
}

func TestShortenerServiceCreate(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	generator := &sequenceGenerator{codes: []string{"Ab3_dE9xY0"}}
	shortener := NewShortenerService(repo, generator)

	link, created, err := shortener.Create(context.Background(), "https://example.com/article")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created {
		t.Fatal("Create() created = false, want true")
	}
	want := domain.Link{
		ShortCode:   "Ab3_dE9xY0",
		OriginalURL: "https://example.com/article",
	}
	if link != want {
		t.Fatalf("Create() link = %+v, want %+v", link, want)
	}

	stored, err := repo.GetByShort(context.Background(), link.ShortCode)
	if err != nil {
		t.Fatalf("GetByShort() error = %v", err)
	}
	if stored != want {
		t.Fatalf("stored link = %+v, want %+v", stored, want)
	}
}

func TestShortenerServiceReturnsExistingLink(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	want := domain.Link{
		ShortCode:   "Ab3_dE9xY0",
		OriginalURL: "https://example.com/existing",
	}
	if err := repo.Create(context.Background(), want); err != nil {
		t.Fatalf("repository Create() error = %v", err)
	}

	generator := &sequenceGenerator{}
	shortener := NewShortenerService(repo, generator)
	link, created, err := shortener.Create(context.Background(), want.OriginalURL)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created {
		t.Fatal("Create() created = true, want false")
	}
	if link != want {
		t.Fatalf("Create() link = %+v, want %+v", link, want)
	}
	if calls := generator.calls(); calls != 0 {
		t.Fatalf("generator calls = %d, want 0", calls)
	}
}

func TestShortenerServiceRetriesShortCodeCollision(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	if err := repo.Create(context.Background(), domain.Link{
		ShortCode:   "AAAAAAAAAA",
		OriginalURL: "https://example.com/occupied",
	}); err != nil {
		t.Fatalf("repository Create() error = %v", err)
	}

	generator := &sequenceGenerator{codes: []string{"AAAAAAAAAA", "BBBBBBBBBB"}}
	shortener := NewShortenerService(repo, generator)
	link, created, err := shortener.Create(context.Background(), "https://example.com/new")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created {
		t.Fatal("Create() created = false, want true")
	}
	if link.ShortCode != "BBBBBBBBBB" {
		t.Fatalf("Create() short code = %q, want %q", link.ShortCode, "BBBBBBBBBB")
	}
	if calls := generator.calls(); calls != 2 {
		t.Fatalf("generator calls = %d, want 2", calls)
	}
}

func TestShortenerServiceConcurrentCreateSameURL(t *testing.T) {
	const requests = 100

	repo := memory.New()
	shortener := NewShortenerService(repo, &sequenceGenerator{})
	originalURL := "https://example.com/shared"

	type result struct {
		link    domain.Link
		created bool
		err     error
	}
	results := make(chan result, requests)

	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			link, created, err := shortener.Create(context.Background(), originalURL)
			results <- result{link: link, created: created, err: err}
		}()
	}
	wg.Wait()
	close(results)

	createdCount := 0
	var shortCode string
	for result := range results {
		if result.err != nil {
			t.Fatalf("Create() error = %v", result.err)
		}
		if result.created {
			createdCount++
		}
		if shortCode == "" {
			shortCode = result.link.ShortCode
		}
		if result.link.ShortCode != shortCode || result.link.OriginalURL != originalURL {
			t.Fatalf("Create() link = %+v, want URL %q and code %q", result.link, originalURL, shortCode)
		}
	}
	if createdCount != 1 {
		t.Fatalf("new links = %d, want 1", createdCount)
	}
}

func TestShortenerServiceRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	shortener := NewShortenerService(memory.New(), &sequenceGenerator{})
	invalidURLs := []string{
		"",
		"example.com",
		"ftp://example.com",
		"https:///missing-host",
		"://broken",
	}

	for _, originalURL := range invalidURLs {
		originalURL := originalURL
		t.Run(originalURL, func(t *testing.T) {
			_, _, err := shortener.Create(context.Background(), originalURL)
			if !errors.Is(err, domain.ErrInvalidURL) {
				t.Fatalf("Create() error = %v, want ErrInvalidURL", err)
			}
		})
	}
}

func TestShortenerServiceRejectsInvalidGeneratedCode(t *testing.T) {
	t.Parallel()

	shortener := NewShortenerService(
		memory.New(),
		&sequenceGenerator{codes: []string{"too-short"}},
	)

	_, _, err := shortener.Create(context.Background(), "https://example.com")
	if !errors.Is(err, domain.ErrInvalidShortCode) {
		t.Fatalf("Create() error = %v, want ErrInvalidShortCode", err)
	}
}

func TestShortenerServiceStopsAfterRepeatedCollisions(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	if err := repo.Create(context.Background(), domain.Link{
		ShortCode:   "AAAAAAAAAA",
		OriginalURL: "https://example.com/occupied",
	}); err != nil {
		t.Fatalf("repository Create() error = %v", err)
	}

	generator := &sequenceGenerator{codes: []string{"AAAAAAAAAA"}}
	shortener := NewShortenerService(repo, generator)
	_, _, err := shortener.Create(context.Background(), "https://example.com/new")
	if err == nil {
		t.Fatal("Create() error = nil after repeated collisions")
	}
	if calls := generator.calls(); calls != maxCreateAttempts {
		t.Fatalf("generator calls = %d, want %d", calls, maxCreateAttempts)
	}
}

func TestShortenerServiceGeneratorError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("random source failed")
	shortener := NewShortenerService(
		memory.New(),
		&sequenceGenerator{err: wantErr},
	)

	_, _, err := shortener.Create(context.Background(), "https://example.com")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want wrapped generator error", err)
	}
}

func TestShortenerServiceGetByShort(t *testing.T) {
	t.Parallel()

	repo := memory.New()
	want := domain.Link{
		ShortCode:   "Ab3_dE9xY0",
		OriginalURL: "https://example.com",
	}
	if err := repo.Create(context.Background(), want); err != nil {
		t.Fatalf("repository Create() error = %v", err)
	}

	shortener := NewShortenerService(repo, &sequenceGenerator{})
	got, err := shortener.GetByShort(context.Background(), want.ShortCode)
	if err != nil {
		t.Fatalf("GetByShort() error = %v", err)
	}
	if got != want {
		t.Fatalf("GetByShort() = %+v, want %+v", got, want)
	}

	_, err = shortener.GetByShort(context.Background(), "invalid")
	if !errors.Is(err, domain.ErrInvalidShortCode) {
		t.Fatalf("GetByShort() error = %v, want ErrInvalidShortCode", err)
	}

	_, err = shortener.GetByShort(context.Background(), "1234567890")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByShort() error = %v, want ErrNotFound", err)
	}
}

func TestShortenerServiceHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	shortener := NewShortenerService(memory.New(), &sequenceGenerator{})

	if _, _, err := shortener.Create(ctx, "https://example.com"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if _, err := shortener.GetByShort(ctx, "Ab3_dE9xY0"); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByShort() error = %v, want context.Canceled", err)
	}
}
