package fx

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/fx"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/domain"
	memoryRepo "github.com/sp3dr4/dove/internal/infrastructure/memory"
	postgresRepo "github.com/sp3dr4/dove/internal/infrastructure/postgres"
	sqliteRepo "github.com/sp3dr4/dove/internal/infrastructure/sqlite"
)

// ProvideLogger creates and configures the application logger
func ProvideLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return logger
}

// ProvideRepository creates the appropriate repository based on configuration
func ProvideRepository(cfg *config.Config, logger *slog.Logger) (domain.URLRepository, error) {
	switch cfg.Database.Type {
	case "memory":
		logger.Info("Using in-memory repository")
		return memoryRepo.NewURLRepository(), nil

	case "sqlite":
		dbURL := cfg.GetDatabaseURL()
		logger.Info("Using SQLite repository", "path", dbURL)

		// Create data directory if it doesn't exist
		if err := os.MkdirAll("./data", 0750); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		db, err := sqlx.Connect("sqlite3", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}

		if err := runMigrations(db, "sqlite3", "sqlite"); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		return sqliteRepo.NewURLRepository(db), nil

	case "postgres":
		dbURL := cfg.GetDatabaseURL()
		logger.Info("Using PostgreSQL repository", "url", dbURL)

		db, err := sqlx.Connect("postgres", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}

		if err := runMigrations(db, "postgres", "postgres"); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		return postgresRepo.NewURLRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}

// runMigrations runs database migrations
func runMigrations(db interface{}, driverName, migrationDir string) error {
	var driver database.Driver
	var err error

	sqlDB, ok := db.(*sqlx.DB)
	if ok {
		db = sqlDB.DB
	}

	sqliteDB, ok := db.(*sql.DB)
	if !ok {
		return fmt.Errorf("expected *sql.DB, got %T", db)
	}

	switch driverName {
	case "sqlite3":
		driver, err = sqlite3.WithInstance(sqliteDB, &sqlite3.Config{})
	case "postgres":
		driver, err = postgres.WithInstance(sqliteDB, &postgres.Config{})
	default:
		return fmt.Errorf("unsupported driver: %s", driverName)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationPath := fmt.Sprintf("file://migrations/%s", migrationDir)
	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		driverName,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("Migrations completed successfully")
	return nil
}

// RepositoryParams holds the parameters needed for repository lifecycle management
type RepositoryParams struct {
	fx.In

	Repository domain.URLRepository
	Logger     *slog.Logger
}

// RegisterRepositoryHooks registers repository lifecycle hooks with FX
func RegisterRepositoryHooks(lc fx.Lifecycle, params RepositoryParams) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if err := params.Repository.Close(); err != nil {
				params.Logger.Error("Failed to close repository resources", "error", err)
				return err
			}
			params.Logger.Info("Repository resources closed successfully")
			return nil
		},
	})
}
