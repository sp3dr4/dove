package fx

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/domain"
	cacheImpl "github.com/sp3dr4/dove/internal/infrastructure/cache"
	memoryRepo "github.com/sp3dr4/dove/internal/infrastructure/memory"
	postgresRepo "github.com/sp3dr4/dove/internal/infrastructure/postgres"
	redisCache "github.com/sp3dr4/dove/internal/infrastructure/redis"
	sqliteRepo "github.com/sp3dr4/dove/internal/infrastructure/sqlite"
)

// ProvideLogger creates and configures the application logger
func ProvideLogger(cfg *config.Config) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.Logging.Level),
	}))
	slog.SetDefault(logger)
	return logger
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ProvideRepository creates the appropriate repository based on configuration
func ProvideRepository(cfg *config.Config, logger *slog.Logger) (domain.URLRepository, error) {
	switch cfg.Database.Type {
	case "memory":
		logger.Info("Using in-memory repository")
		return memoryRepo.NewURLRepository(logger), nil

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

		return sqliteRepo.NewURLRepository(db, logger), nil

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

		return postgresRepo.NewURLRepository(db, logger), nil

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

// ProvideRedisClient creates a Redis client
func ProvideRedisClient(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	if !cfg.Cache.Enabled {
		return nil, nil
	}

	opt, err := redis.ParseURL(cfg.Cache.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with config values
	if cfg.Cache.Redis.Password != "" {
		opt.Password = cfg.Cache.Redis.Password
	}
	opt.DB = cfg.Cache.Redis.DB
	opt.PoolSize = cfg.Cache.Redis.PoolSize
	opt.MinIdleConns = cfg.Cache.Redis.MinIdleConns
	opt.MaxRetries = cfg.Cache.Redis.MaxRetries

	if cfg.Cache.Redis.ReadTimeout != "" {
		readTimeout, err := time.ParseDuration(cfg.Cache.Redis.ReadTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid read timeout: %w", err)
		}
		opt.ReadTimeout = readTimeout
	}

	if cfg.Cache.Redis.WriteTimeout != "" {
		writeTimeout, err := time.ParseDuration(cfg.Cache.Redis.WriteTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid write timeout: %w", err)
		}
		opt.WriteTimeout = writeTimeout
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Failed to connect to Redis, caching will be disabled", "error", err)
		return nil, nil
	}

	logger.Info("Connected to Redis", "url", cfg.Cache.Redis.URL)
	return client, nil
}

// ProvideCache creates the appropriate cache implementation
func ProvideCache(cfg *config.Config, client *redis.Client, logger *slog.Logger) domain.Cache {
	if !cfg.Cache.Enabled || client == nil {
		logger.Info("Caching disabled")
		return cacheImpl.NewNoOpCache()
	}

	logger.Info("Using Redis cache", "ttl", cfg.Cache.TTL)
	return redisCache.NewRedisCache(client, logger)
}

// ProvideCacheTTL provides the cache TTL duration
func ProvideCacheTTL(cfg *config.Config) (time.Duration, error) {
	if cfg.Cache.TTL == "" {
		return 10 * time.Minute, nil // Default to 10 minutes
	}

	ttl, err := time.ParseDuration(cfg.Cache.TTL)
	if err != nil {
		return 0, fmt.Errorf("invalid cache TTL: %w", err)
	}

	return ttl, nil
}

// CacheParams holds the parameters needed for cache lifecycle management
type CacheParams struct {
	fx.In

	Client *redis.Client `optional:"true"`
	Logger *slog.Logger
}

// RegisterCacheHooks registers cache lifecycle hooks with FX
func RegisterCacheHooks(lc fx.Lifecycle, params CacheParams) {
	if params.Client != nil {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				if err := params.Client.Close(); err != nil {
					params.Logger.Error("Failed to close Redis connection", "error", err)
					return err
				}
				params.Logger.Info("Redis connection closed successfully")
				return nil
			},
		})
	}
}
