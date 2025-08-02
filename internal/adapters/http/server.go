package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpswagger "github.com/swaggo/http-swagger"
)

func NewRouter(handlers *Handlers) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handlers.HandleHealth)
	r.Get("/swagger/*", httpswagger.Handler(
		httpswagger.URL("http://localhost:8080/swagger/doc.json"),
	))
	r.Get("/redoc", handleRedoc)

	r.Post("/shorten", handlers.HandleShorten)
	r.Get("/{shortCode}", handlers.HandleRedirect)

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
