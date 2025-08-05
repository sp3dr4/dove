package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLService struct {
	repo     domain.URLRepository
	cache    domain.Cache
	cacheTTL time.Duration
	validate *validator.Validate
	logger   *slog.Logger
}

func NewURLService(repo domain.URLRepository, cache domain.Cache, cacheTTL time.Duration, logger *slog.Logger) *URLService {
	return &URLService{
		repo:     repo,
		cache:    cache,
		cacheTTL: cacheTTL,
		validate: validator.New(),
		logger:   logger,
	}
}

type CreateURLRequest struct {
	URL         string `json:"url" validate:"required,url"`
	CustomAlias string `json:"customAlias,omitempty" validate:"omitempty,alphanum,min=3,max=20"`
}

type URLResponse struct {
	ID          int64     `json:"id"`
	ShortURL    string    `json:"shortUrl"`
	ShortCode   string    `json:"shortCode"`
	OriginalURL string    `json:"originalUrl"`
	Clicks      int       `json:"clicks"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (s *URLService) CreateShortURL(ctx context.Context, req CreateURLRequest, baseURL string) (*URLResponse, error) {
	if err := s.validate.Struct(req); err != nil {
		return nil, err
	}

	shortCode := req.CustomAlias
	if shortCode == "" {
		shortCode = generateShortCode()
	}

	exists, err := s.repo.Exists(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrShortCodeExists
	}

	url, err := domain.NewURL(shortCode, req.URL)
	if err != nil {
		return nil, err
	}

	createdURL, err := s.repo.Create(ctx, url)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, createdURL, s.cacheTTL); err != nil {
		s.logger.Warn("Failed to cache new URL", "short_code", createdURL.ShortCode, "error", err)
	}

	return &URLResponse{
		ID:          createdURL.ID,
		ShortURL:    baseURL + "/" + shortCode,
		ShortCode:   shortCode,
		OriginalURL: createdURL.OriginalURL,
		Clicks:      createdURL.Clicks,
		CreatedAt:   createdURL.CreatedAt,
		UpdatedAt:   createdURL.UpdatedAt,
	}, nil
}

func (s *URLService) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	cachedURL, err := s.cache.Get(ctx, shortCode)
	if err != nil {
		s.logger.Warn("Cache error during get", "short_code", shortCode, "error", err)
	}

	// Cache hit
	if cachedURL != nil {
		s.logger.Debug("Cache hit", "short_code", shortCode)
		return cachedURL, nil
	}

	// Cache miss
	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, url, s.cacheTTL); err != nil {
		s.logger.Warn("Failed to cache URL", "short_code", shortCode, "error", err)
	}

	return url, nil
}

func (s *URLService) IncrementClicks(ctx context.Context, shortCode string) (*domain.URL, error) {
	url, err := s.repo.IncrementClicks(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, url, s.cacheTTL); err != nil {
		s.logger.Warn("Failed to update cache after incrementing clicks", "short_code", shortCode, "error", err)
	}

	return url, nil
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
