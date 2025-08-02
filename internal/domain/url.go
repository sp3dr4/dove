package domain

import (
	"errors"
	"time"
)

var (
	ErrURLNotFound      = errors.New("url not found")
	ErrShortCodeExists  = errors.New("short code already exists")
	ErrInvalidURL       = errors.New("invalid url")
	ErrInvalidShortCode = errors.New("invalid short code")
)

type URL struct {
	ID          int64     `db:"id" json:"id"`
	ShortCode   string    `db:"short_code" json:"shortCode"`
	OriginalURL string    `db:"original_url" json:"originalUrl"`
	Clicks      int       `db:"clicks" json:"clicks"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

func NewURL(shortCode, originalURL string) (*URL, error) {
	if shortCode == "" {
		return nil, ErrInvalidShortCode
	}
	if originalURL == "" {
		return nil, ErrInvalidURL
	}

	now := time.Now()
	return &URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		Clicks:      0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (u *URL) IncrementClicks() {
	u.Clicks++
}
