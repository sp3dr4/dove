package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLRepository struct {
	db *sqlx.DB
}

func NewURLRepository(db *sqlx.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, clicks, created_at)
		VALUES (:short_code, :original_url, :clicks, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, url)
	if err != nil {
		return domain.ErrShortCodeExists
	}

	return nil
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

func (r *URLRepository) IncrementClicks(ctx context.Context, shortCode string) error {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code = $1`

	result, err := r.db.ExecContext(ctx, query, shortCode)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrURLNotFound
	}

	return nil
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
