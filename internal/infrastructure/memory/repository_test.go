package memory

import (
	"context"
	"testing"

	"github.com/sp3dr4/dove/internal/domain"
)

func TestMemoryRepository_Create(t *testing.T) {
	repo := NewURLRepository()
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		Clicks:      0,
	}

	err := repo.Create(ctx, url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Try to create duplicate
	err = repo.Create(ctx, url)
	if err != domain.ErrShortCodeExists {
		t.Fatalf("expected ErrShortCodeExists, got %v", err)
	}
}

func TestMemoryRepository_FindByShortCode(t *testing.T) {
	repo := NewURLRepository()
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		Clicks:      0,
	}

	if err := repo.Create(ctx, url); err != nil {
		t.Fatalf("failed to create URL: %v", err)
	}

	// Find existing
	found, err := repo.FindByShortCode(ctx, "test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.OriginalURL != url.OriginalURL {
		t.Fatalf("expected %s, got %s", url.OriginalURL, found.OriginalURL)
	}

	// Find non-existing
	_, err = repo.FindByShortCode(ctx, "notfound")
	if err != domain.ErrURLNotFound {
		t.Fatalf("expected ErrURLNotFound, got %v", err)
	}
}

func TestMemoryRepository_IncrementClicks(t *testing.T) {
	repo := NewURLRepository()
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		Clicks:      0,
	}

	if err := repo.Create(ctx, url); err != nil {
		t.Fatalf("failed to create URL: %v", err)
	}

	err := repo.IncrementClicks(ctx, "test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, _ := repo.FindByShortCode(ctx, "test123")
	if found.Clicks != 1 {
		t.Fatalf("expected clicks to be 1, got %d", found.Clicks)
	}

	// Non-existing
	err = repo.IncrementClicks(ctx, "notfound")
	if err != domain.ErrURLNotFound {
		t.Fatalf("expected ErrURLNotFound, got %v", err)
	}
}

func TestMemoryRepository_Exists(t *testing.T) {
	repo := NewURLRepository()
	ctx := context.Background()

	url := &domain.URL{
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		Clicks:      0,
	}

	if err := repo.Create(ctx, url); err != nil {
		t.Fatalf("failed to create URL: %v", err)
	}

	exists, err := repo.Exists(ctx, "test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected URL to exist")
	}

	exists, err = repo.Exists(ctx, "notfound")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected URL to not exist")
	}
}
