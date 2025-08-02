// Package main implements a simple URL shortener service.
//
//	@title			Dove URL Shortener API
//	@version		1.0
//	@description	A fast and simple URL shortener service
//	@host			localhost:8080
//	@BasePath		/
//	@schemes		http https
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/sp3dr4/dove/config"
	_ "github.com/sp3dr4/dove/docs"
	httpAdapter "github.com/sp3dr4/dove/internal/adapters/http"
	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/domain"
	memoryRepo "github.com/sp3dr4/dove/internal/infrastructure/memory"
	sqliteRepo "github.com/sp3dr4/dove/internal/infrastructure/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	repo, err := setupRepository(cfg)
	if err != nil {
		slog.Error("Failed to setup repository", "error", err)
		os.Exit(1)
	}

	service := application.NewURLService(repo)
	handlers := httpAdapter.NewHandlers(service, cfg.App.BaseURL)
	router := httpAdapter.NewRouter(handlers)

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	if timeout, err := time.ParseDuration(cfg.Server.ReadTimeout); err == nil {
		srv.ReadTimeout = timeout
	}
	if timeout, err := time.ParseDuration(cfg.Server.WriteTimeout); err == nil {
		srv.WriteTimeout = timeout
	}
	if timeout, err := time.ParseDuration(cfg.Server.IdleTimeout); err == nil {
		srv.IdleTimeout = timeout
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Starting server",
			"port", cfg.Server.Port,
			"database", cfg.Database.Type,
			"base_url", cfg.App.BaseURL,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exited")
}

func setupRepository(cfg *config.Config) (domain.URLRepository, error) {
	switch cfg.Database.Type {
	case "memory":
		slog.Info("Using in-memory repository")
		return memoryRepo.NewURLRepository(), nil

	case "sqlite":
		dbURL := cfg.GetDatabaseURL()
		slog.Info("Using SQLite repository", "path", dbURL)

		// Create data directory if it doesn't exist
		if err := os.MkdirAll("./data", 0750); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		db, err := sqlx.Connect("sqlite3", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}

		if err := runMigrations(db, "sqlite3"); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		return sqliteRepo.NewURLRepository(db), nil

	case "postgres":
		dbURL := cfg.GetDatabaseURL()
		slog.Info("Using PostgreSQL repository", "url", dbURL)

		db, err := sqlx.Connect("postgres", dbURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}

		if err := runMigrations(db, "postgres"); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		return sqliteRepo.NewURLRepository(db), nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}

func runMigrations(db interface{}, driverName string) error {
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

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
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
