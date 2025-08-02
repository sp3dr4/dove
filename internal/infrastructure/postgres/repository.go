package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/sp3dr4/dove/internal/domain"
)

type URLRepository struct {
	db *sqlx.DB
}

func NewURLRepository(db *sqlx.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(ctx context.Context, url *domain.URL) (*domain.URL, error) {
	query := `
		INSERT INTO urls (short_code, original_url, clicks, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, short_code, original_url, clicks, created_at, updated_at
	`

	var result domain.URL
	err := r.db.QueryRowContext(ctx, query, url.ShortCode, url.OriginalURL, url.Clicks, url.CreatedAt).
		Scan(&result.ID, &result.ShortCode, &result.OriginalURL, &result.Clicks, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, r.handlePostgreSQLError(err, "create URL")
	}

	slog.Debug("URL created successfully", "short_code", result.ShortCode, "id", result.ID)
	return &result, nil
}

func (r *URLRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	var url domain.URL
	query := `SELECT id, short_code, original_url, clicks, created_at FROM urls WHERE short_code = $1`

	err := r.db.GetContext(ctx, &url, query, shortCode)
	if err != nil {
		return nil, r.handlePostgreSQLError(err, "find URL by short code")
	}

	return &url, nil
}

func (r *URLRepository) IncrementClicks(ctx context.Context, shortCode string) (*domain.URL, error) {
	query := `
		UPDATE urls 
		SET clicks = clicks + 1 
		WHERE short_code = $1
		RETURNING id, short_code, original_url, clicks, created_at, updated_at
	`

	var url domain.URL
	err := r.db.QueryRowContext(ctx, query, shortCode).
		Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.Clicks, &url.CreatedAt, &url.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, r.handlePostgreSQLError(err, "increment clicks")
	}

	slog.Debug("Clicks incremented", "short_code", shortCode, "new_count", url.Clicks)
	return &url, nil
}

func (r *URLRepository) Exists(ctx context.Context, shortCode string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	err := r.db.GetContext(ctx, &exists, query, shortCode)
	if err != nil {
		return false, r.handlePostgreSQLError(err, "check URL existence")
	}

	return exists, nil
}

// handlePostgreSQLError converts PostgreSQL-specific errors to domain errors
func (r *URLRepository) handlePostgreSQLError(err error, operation string) error {
	if pqErr, ok := err.(*pq.Error); ok {
		slog.Error("PostgreSQL error",
			"operation", operation,
			"code", pqErr.Code,
			"message", pqErr.Message,
			"detail", pqErr.Detail,
		)

		switch pqErr.Code {
		case "23505": // unique_violation
			if pqErr.Constraint == "urls_short_code_key" {
				return domain.ErrShortCodeExists
			}
			return fmt.Errorf("unique constraint violation: %s", pqErr.Detail)
		case "23502": // not_null_violation
			return fmt.Errorf("required field missing: %s", pqErr.Column)
		case "23514": // check_violation
			return fmt.Errorf("check constraint violation: %s", pqErr.Detail)
		case "08000", "08003", "08006": // connection errors
			return fmt.Errorf("database connection error: %s", pqErr.Message)
		default:
			return fmt.Errorf("database error [%s]: %s", pqErr.Code, pqErr.Message)
		}
	}

	// Handle standard SQL errors
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrURLNotFound
	}

	return err
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
