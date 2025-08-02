package application

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLService struct {
	repo     domain.URLRepository
	validate *validator.Validate
}

func NewURLService(repo domain.URLRepository) *URLService {
	return &URLService{
		repo:     repo,
		validate: validator.New(),
	}
}

type CreateURLRequest struct {
	URL         string `json:"url" validate:"required,url"`
	CustomAlias string `json:"customAlias,omitempty" validate:"omitempty,alphanum,min=3,max=20"`
}

type URLResponse struct {
	ShortURL    string    `json:"shortUrl"`
	ShortCode   string    `json:"shortCode"`
	OriginalURL string    `json:"originalUrl"`
	CreatedAt   time.Time `json:"createdAt"`
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

	if err := s.repo.Create(ctx, url); err != nil {
		return nil, err
	}

	return &URLResponse{
		ShortURL:    baseURL + "/" + shortCode,
		ShortCode:   shortCode,
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
	}, nil
}

func (s *URLService) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	return s.repo.FindByShortCode(ctx, shortCode)
}

func (s *URLService) IncrementClicks(ctx context.Context, shortCode string) error {
	return s.repo.IncrementClicks(ctx, shortCode)
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
