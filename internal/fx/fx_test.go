package fx

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/domain"
	httpFX "github.com/sp3dr4/dove/internal/fx/http"
	cacheImpl "github.com/sp3dr4/dove/internal/infrastructure/cache"
)

func TestFXIntegration(t *testing.T) {
	// Test that all dependencies can be wired correctly
	app := fxtest.New(t,
		// Use test configuration
		fx.Provide(func() (*config.Config, error) {
			return &config.Config{
				Server: config.ServerConfig{
					Port:         "8080",
					ReadTimeout:  "15s",
					WriteTimeout: "15s",
					IdleTimeout:  "60s",
				},
				Database: config.DatabaseConfig{
					Type: "memory",
				},
				App: config.AppConfig{
					BaseURL:         "http://localhost:8080",
					ShortCodeLength: 6,
				},
				Cache: config.CacheConfig{
					Enabled: false, // Disable cache for tests
				},
			}, nil
		}),

		// Use the same providers as the main app
		InfrastructureModule,
		ApplicationModule,
		httpFX.HTTPModule,

		// Test that we can get the service
		fx.Invoke(func(service *application.URLService, repo domain.URLRepository) {
			require.NotNil(t, service)
			require.NotNil(t, repo)

			// Test basic functionality
			ctx := context.Background()
			req := application.CreateURLRequest{
				URL: "https://example.com",
			}

			resp, err := service.CreateShortURL(ctx, req, "http://localhost:8080")
			require.NoError(t, err)
			assert.Equal(t, "https://example.com", resp.OriginalURL)
			assert.NotEmpty(t, resp.ShortCode)
		}),
	)

	// Start and stop the app to ensure lifecycle works
	app.RequireStart()
	app.RequireStop()
}

func TestFXModules(t *testing.T) {
	// Test that individual modules can be loaded
	tests := []struct {
		name         string
		module       fx.Option
		needsConfig  bool
		needsRepo    bool
		needsService bool
	}{
		{"InfrastructureModule", InfrastructureModule, true, false, false},
		{"ApplicationModule", ApplicationModule, false, true, false},
		{"HTTPModule", httpFX.HTTPModule, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := []fx.Option{tt.module}

			// Add dependencies based on what the module needs
			if tt.needsConfig {
				options = append(options, fx.Provide(func() (*config.Config, error) {
					return &config.Config{
						Database: config.DatabaseConfig{Type: "memory"},
						App:      config.AppConfig{BaseURL: "http://localhost:8080"},
						Server:   config.ServerConfig{Port: "8080"},
					}, nil
				}))
			}

			if tt.needsRepo {
				options = append(options, fx.Provide(func() domain.URLRepository {
					return &mockRepository{}
				}))
			}

			if tt.needsService {
				options = append(options, fx.Provide(func(repo domain.URLRepository) *application.URLService {
					logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
					noopCache := cacheImpl.NewNoOpCache()
					return application.NewURLService(repo, noopCache, 10*time.Minute, logger)
				}))
			}

			// Create a minimal app with just the module
			app := fxtest.New(t, options...)

			// Should be able to start and stop without errors
			app.RequireStart()
			app.RequireStop()
		})
	}
}

func TestConfigModule(t *testing.T) {
	// Test ConfigModule separately since it provides config
	app := fxtest.New(t, ConfigModule)
	app.RequireStart()
	app.RequireStop()
}
func TestProviderFunctions(t *testing.T) {
	t.Run("ProvideLogger", func(t *testing.T) {
		cfg := &config.Config{
			Logging: config.LoggingConfig{Level: "info"},
		}
		logger := ProvideLogger(cfg)
		assert.NotNil(t, logger)
	})

	t.Run("ProvideRepository", func(t *testing.T) {
		cfg := &config.Config{
			Database: config.DatabaseConfig{Type: "memory"},
			Logging:  config.LoggingConfig{Level: "info"},
		}
		logger := ProvideLogger(cfg)

		repo, err := ProvideRepository(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, repo)

		// Test that it implements the interface
		_ = repo
	})

	t.Run("ProvideHTTPServer", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port:         "8080",
				ReadTimeout:  "15s",
				WriteTimeout: "15s",
				IdleTimeout:  "60s",
			},
		}

		// Create a chi router for testing
		router := chi.NewRouter()

		server := httpFX.ProvideHTTPServer(cfg, router)
		assert.NotNil(t, server)
		assert.Equal(t, ":8080", server.Addr())
	})
}

// mockRepository is a simple mock repository for testing
type mockRepository struct{}

func (m *mockRepository) Create(ctx context.Context, url *domain.URL) (*domain.URL, error) {
	return url, nil
}

func (m *mockRepository) FindByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	return &domain.URL{ShortCode: shortCode, OriginalURL: "https://example.com"}, nil
}

func (m *mockRepository) IncrementClicks(ctx context.Context, shortCode string) (*domain.URL, error) {
	return &domain.URL{ShortCode: shortCode, OriginalURL: "https://example.com", Clicks: 1}, nil
}

func (m *mockRepository) Exists(ctx context.Context, shortCode string) (bool, error) {
	return false, nil
}

func (m *mockRepository) Close() error {
	return nil
}

func (m *mockRepository) HealthCheck(ctx context.Context) error {
	return nil
}
