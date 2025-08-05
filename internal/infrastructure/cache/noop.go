package cache

import (
	"context"
	"time"

	"github.com/sp3dr4/dove/internal/domain"
)

// NoOpCache is a no-operation cache implementation that does nothing
// Used when caching is disabled
type NoOpCache struct{}

func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(_ context.Context, _ string) (*domain.URL, error) {
	// Always return cache miss
	return nil, nil
}

func (c *NoOpCache) Set(_ context.Context, _ *domain.URL, _ time.Duration) error {
	// Do nothing
	return nil
}

func (c *NoOpCache) Delete(_ context.Context, _ string) error {
	// Do nothing
	return nil
}

func (c *NoOpCache) Ping(_ context.Context) error {
	// Always available
	return nil
}
