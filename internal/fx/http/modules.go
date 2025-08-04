package http

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"

	httpAdapter "github.com/sp3dr4/dove/internal/adapters/http"
)

// ProvideRouter creates a chi router with all dependencies
func ProvideRouter(handlers *httpAdapter.Handlers, logger *slog.Logger) chi.Router {
	return httpAdapter.NewRouter(handlers, logger)
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
