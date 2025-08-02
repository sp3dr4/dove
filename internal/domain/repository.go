package domain

import "context"

type URLRepository interface {
	Create(ctx context.Context, url *URL) error
	FindByShortCode(ctx context.Context, shortCode string) (*URL, error)
	IncrementClicks(ctx context.Context, shortCode string) error
	Exists(ctx context.Context, shortCode string) (bool, error)
}
