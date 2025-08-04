package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLRepository struct {
	db     *sqlx.DB
	logger *slog.Logger
}

func NewURLRepository(db *sqlx.DB, logger *slog.Logger) *URLRepository {
	return &URLRepository{db: db, logger: logger}
}

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) (*domain.URL, error) {
	query := `
		INSERT INTO urls (short_code, original_url, clicks, created_at, updated_at)
		VALUES (:short_code, :original_url, :clicks, :created_at, :updated_at)
	`

	result, err := r.db.NamedExecContext(ctx, query, url)
	if err != nil {
		return nil, domain.ErrShortCodeExists
	}

	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	createdURL := &domain.URL{
		ID:          id,
		ShortCode:   url.ShortCode,
		OriginalURL: url.OriginalURL,
		Clicks:      url.Clicks,
		CreatedAt:   url.CreatedAt,
		UpdatedAt:   url.UpdatedAt,
	}

	return createdURL, nil
}

func (r *URLRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	var url domain.URL
	query := `SELECT * FROM urls WHERE short_code = $1`

	err := r.db.GetContext(ctx, &url, query, shortCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, err
	}

	return &url, nil
}

func (r *URLRepository) IncrementClicks(ctx context.Context, shortCode string) (*domain.URL, error) {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code = $1`

	result, err := r.db.ExecContext(ctx, query, shortCode)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, domain.ErrURLNotFound
	}

	// Fetch the updated record
	return r.FindByShortCode(ctx, shortCode)
}

func (r *URLRepository) Exists(ctx context.Context, shortCode string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	err := r.db.GetContext(ctx, &exists, query, shortCode)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *URLRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *URLRepository) HealthCheck(ctx context.Context) error {
	if r.db == nil {
		return errors.New("database connection is nil")
	}
	return r.db.PingContext(ctx)
}
