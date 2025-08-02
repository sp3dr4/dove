package memory

import (
	"context"
	"sync"
	"time"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLRepository struct {
	urls map[string]*domain.URL
	mu   sync.RWMutex
}

func NewURLRepository() *URLRepository {
	return &URLRepository{
		urls: make(map[string]*domain.URL),
	}
}

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) (*domain.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.urls[url.ShortCode]; exists {
		return nil, domain.ErrShortCodeExists
	}

	// Create a copy with a generated ID (simulate database behavior)
	createdURL := &domain.URL{
		ID:          int64(len(r.urls) + 1), // Simple ID generation
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalURL,
		Clicks:      url.Clicks,
		CreatedAt:   url.CreatedAt,
		UpdatedAt:   url.UpdatedAt,
	}

	r.urls[url.ShortCode] = createdURL
	return createdURL, nil
}

func (r *URLRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, exists := r.urls[shortCode]
	if !exists {
		return nil, domain.ErrURLNotFound
	}

	return url, nil
}

func (r *URLRepository) IncrementClicks(ctx context.Context, shortCode string) (*domain.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	url, exists := r.urls[shortCode]
	if !exists {
		return nil, domain.ErrURLNotFound
	}

	url.Clicks++
	url.UpdatedAt = time.Now()

	return url, nil
}

func (r *URLRepository) Exists(ctx context.Context, shortCode string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.urls[shortCode]
	return exists, nil
}

func (r *URLRepository) Close() error {
	return nil
}

func (r *URLRepository) HealthCheck(ctx context.Context) error {
	return nil
}
