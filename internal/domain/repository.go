package domain

import "context"

type URLRepository interface {
	Create(ctx context.Context, url *URL) (*URL, error)
	FindByShortCode(ctx context.Context, shortCode string) (*URL, error)
	IncrementClicks(ctx context.Context, shortCode string) (*URL, error)
	Exists(ctx context.Context, shortCode string) (bool, error)
	Close() error
	HealthCheck(ctx context.Context) error
}
