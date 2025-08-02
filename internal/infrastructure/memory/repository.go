package memory

import (
	"context"
	"sync"

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

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.urls[url.ShortCode]; exists {
		return domain.ErrShortCodeExists
	}

	r.urls[url.ShortCode] = url
	return nil
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

func (r *URLRepository) IncrementClicks(ctx context.Context, shortCode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	url, exists := r.urls[shortCode]
	if !exists {
		return domain.ErrURLNotFound
	}

	url.Clicks++
	return nil
}

func (r *URLRepository) Exists(ctx context.Context, shortCode string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.urls[shortCode]
	return exists, nil
}
