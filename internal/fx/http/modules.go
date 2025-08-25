package http

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"

	"github.com/sp3dr4/dove/config"
	httpAdapter "github.com/sp3dr4/dove/internal/adapters/http"
	"github.com/sp3dr4/dove/internal/pkg/metrics"
)

// ProvideRouter creates a chi router with all dependencies
func ProvideRouter(handlers *httpAdapter.Handlers, logger *slog.Logger, cfg *config.Config, metricsRegistry metrics.Registry) chi.Router {
	return httpAdapter.NewRouter(handlers, logger, cfg, metricsRegistry)
}

// HTTPModule provides HTTP-related dependencies
var HTTPModule = fx.Module("http",
	fx.Provide(ProvideHandlers),
	fx.Provide(ProvideRouter),
	fx.Provide(ProvideHTTPServer),
)

// HTTPLifecycleModule provides HTTP server lifecycle management
var HTTPLifecycleModule = fx.Module("http-lifecycle",
	fx.Invoke(RegisterHTTPServerHooks),
)
