package integration

import (
	"context"
	"database/sql"
	"fmt"
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
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sp3dr4/dove/internal/application"
	postgresRepo "github.com/sp3dr4/dove/internal/infrastructure/postgres"
)

var (
	sharedContainer *postgresContainer.PostgresContainer
	sharedDB        *sqlx.DB
	containerOnce   sync.Once
	cleanupOnce     sync.Once
)

// TestEnvironment holds the test setup
type TestEnvironment struct {
	DB      *sqlx.DB
	Service *application.URLService
}

// SetupTestEnvironment creates a PostgreSQL container (shared), runs migrations, and returns a configured URLService
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	containerOnce.Do(func() {
		ctx := context.Background()

		container, err := postgresContainer.Run(ctx,
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
		sharedContainer = container

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("failed to get connection string: %v", err)
		}

		db, err := sqlx.Connect("postgres", connStr)
		if err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
		sharedDB = db

		if err := runMigrations(db.DB); err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}
	})

	cleanDatabase(t, sharedDB)

	repo := postgresRepo.NewURLRepository(sharedDB)
	service := application.NewURLService(repo)

	return &TestEnvironment{
		DB:      sharedDB,
		Service: service,
	}
}

// CleanupSharedResources should be called once at the end of all tests
func CleanupSharedResources() {
	cleanupOnce.Do(func() {
		ctx := context.Background()
		if sharedDB != nil {
			_ = sharedDB.Close()
		}
		if sharedContainer != nil {
			_ = sharedContainer.Terminate(ctx)
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

// TestMain handles setup and teardown for the entire test suite
func TestMain(m *testing.M) {
	code := m.Run()

	CleanupSharedResources()

	// Exit with the same code as the tests
	os.Exit(code)
}
