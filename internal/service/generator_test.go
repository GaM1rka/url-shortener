package service

import (
	"sync"
	"testing"
)

func TestRandomGeneratorGenerate(t *testing.T) {
	t.Parallel()

	generator := NewRandomGenerator()
	for i := 0; i < 1_000; i++ {
		code, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if !isValidShortCode(code) {
			t.Fatalf("Generate() = %q, want a valid short code", code)
		}
	}
}

func TestRandomGeneratorConcurrentUse(t *testing.T) {
	t.Parallel()

	const goroutines = 100

	generator := NewRandomGenerator()
	results := make(chan string, goroutines)
	errorsCh := make(chan error, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, err := generator.Generate()
			if err != nil {
				errorsCh <- err
				return
			}
			results <- code
		}()
	}
	wg.Wait()
	close(results)
	close(errorsCh)

	for err := range errorsCh {
		t.Fatalf("Generate() error = %v", err)
	}

	seen := make(map[string]struct{}, goroutines)
	for code := range results {
		if !isValidShortCode(code) {
			t.Fatalf("Generate() = %q, want a valid short code", code)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("Generate() returned duplicate code %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) != goroutines {
		t.Fatalf("generated codes = %d, want %d", len(seen), goroutines)
	}
}
