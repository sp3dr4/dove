package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpswagger "github.com/swaggo/http-swagger"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/pkg/metrics"
)

func NewRouter(handlers *Handlers, logger *slog.Logger, cfg *config.Config, metricsRegistry metrics.Registry) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(LoggingMiddleware(logger))
	r.Use(metrics.PrometheusMiddleware(metricsRegistry))
	r.Use(middleware.Recoverer)

	r.Get("/health", handlers.HandleHealth)
	r.Get("/ready", handlers.HandleReady)

	if cfg.Metrics.Enabled {
		r.Handle(cfg.Metrics.Path, metricsRegistry.GetHandler())
	}

	r.Get("/swagger/*", httpswagger.Handler(
		httpswagger.URL("http://localhost:8080/swagger/doc.json"),
	))
	r.Get("/redoc", handleRedoc)

	r.Post("/shorten", handlers.HandleShorten)

	r.Get("/{shortCode}", handlers.HandleRedirect)
	r.Head("/{shortCode}", handlers.HandleRedirect)

	return r
}

func handleRedoc(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	redocHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Dove API Documentation - Redoc</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <redoc spec-url='/swagger/doc.json'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
	_, _ = w.Write([]byte(redocHTML))
}
