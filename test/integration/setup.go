package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sp3dr4/dove/internal/application"
	postgresRepo "github.com/sp3dr4/dove/internal/infrastructure/postgres"
	redisCache "github.com/sp3dr4/dove/internal/infrastructure/redis"
)

var (
	sharedPgContainer    *postgresContainer.PostgresContainer
	sharedRedisContainer *redisContainer.RedisContainer
	sharedDB             *sqlx.DB
	sharedRedisClient    *redis.Client
	containerOnce        sync.Once
	cleanupOnce          sync.Once
)

// TestEnvironment holds the test setup
type TestEnvironment struct {
	DB          *sqlx.DB
	RedisClient *redis.Client
	Service     *application.URLService
}

// SetupTestEnvironment creates PostgreSQL and Redis containers (shared), runs migrations, and returns a configured URLService
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	containerOnce.Do(func() {
		ctx := context.Background()

		pgContainer, err := postgresContainer.Run(ctx,
			"postgres:16-alpine",
			postgresContainer.WithDatabase("dove_test"),
			postgresContainer.WithUsername("test"),
			postgresContainer.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second)),
		)
		if err != nil {
			t.Fatalf("failed to start postgres container: %v", err)
		}
		sharedPgContainer = pgContainer

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("failed to get connection string: %v", err)
		}

		db, err := sqlx.Connect("postgres", connStr)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		sharedDB = db

		if migErr := runMigrations(db.DB); migErr != nil {
			t.Fatalf("failed to run migrations: %v", migErr)
		}

		redisC, err := redisContainer.Run(ctx,
			"redis:7-alpine",
			testcontainers.WithWaitStrategy(
				wait.ForLog("Ready to accept connections").
					WithStartupTimeout(30*time.Second)),
		)
		if err != nil {
			t.Fatalf("failed to start redis container: %v", err)
		}
		sharedRedisContainer = redisC

		redisConnStr, err := redisC.ConnectionString(ctx)
		if err != nil {
			t.Fatalf("failed to get redis connection string: %v", err)
		}

		redisOpt, err := redis.ParseURL(redisConnStr)
		if err != nil {
			t.Fatalf("failed to parse redis URL: %v", err)
		}

		sharedRedisClient = redis.NewClient(redisOpt)

		if err := sharedRedisClient.Ping(ctx).Err(); err != nil {
			t.Fatalf("failed to ping redis: %v", err)
		}
	})

	cleanDatabase(t, sharedDB)
	cleanRedisCache(t, sharedRedisClient)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := postgresRepo.NewURLRepository(sharedDB, logger)
	cache := redisCache.NewRedisCache(sharedRedisClient, logger)
	service := application.NewURLService(repo, cache, 10*time.Minute, logger)

	return &TestEnvironment{
		DB:          sharedDB,
		RedisClient: sharedRedisClient,
		Service:     service,
	}
}

// CleanupSharedResources should be called once at the end of all tests
func CleanupSharedResources() {
	cleanupOnce.Do(func() {
		ctx := context.Background()
		if sharedDB != nil {
			_ = sharedDB.Close()
		}
		if sharedRedisClient != nil {
			_ = sharedRedisClient.Close()
		}
		if sharedPgContainer != nil {
			_ = sharedPgContainer.Terminate(ctx)
		}
		if sharedRedisContainer != nil {
			_ = sharedRedisContainer.Terminate(ctx)
		}
	})
}

// cleanDatabase truncates all tables to ensure test isolation
func cleanDatabase(t *testing.T, db *sqlx.DB) {
	_, err := db.Exec("TRUNCATE TABLE urls RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to clean database: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationsPath, err := filepath.Abs("../../migrations/postgres")
	if err != nil {
		return fmt.Errorf("failed to get migrations path: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// cleanRedisCache flushes all Redis keys to ensure test isolation
func cleanRedisCache(t *testing.T, client *redis.Client) {
	ctx := context.Background()
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("failed to flush Redis: %v", err)
	}
}

// TestMain handles setup and teardown for the entire test suite
func TestMain(m *testing.M) {
	code := m.Run()

	CleanupSharedResources()

	// Exit with the same code as the tests
	os.Exit(code)
}
