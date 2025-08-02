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
}

func NewURL(shortCode, originalURL string) (*URL, error) {
	if shortCode == "" {
		return nil, ErrInvalidShortCode
	}
	if originalURL == "" {
		return nil, ErrInvalidURL
	}

	return &URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		Clicks:      0,
		CreatedAt:   time.Now(),
	}, nil
}

func (u *URL) IncrementClicks() {
	u.Clicks++
}
