package domain

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations
type Cache interface {
	// Get retrieves a URL from cache by its short code
	Get(ctx context.Context, shortCode string) (*URL, error)

	// Set stores a URL in cache with the specified TTL
	Set(ctx context.Context, url *URL, ttl time.Duration) error

	// Delete removes a URL from cache
	Delete(ctx context.Context, shortCode string) error

	// Ping checks if the cache is available
	Ping(ctx context.Context) error
}
