package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sp3dr4/dove/config"
	httpAdapter "github.com/sp3dr4/dove/internal/adapters/http"
	"github.com/sp3dr4/dove/internal/application"
	"github.com/sp3dr4/dove/internal/domain"
	"github.com/sp3dr4/dove/internal/server"
)

// HTTPServer implements the generic Server interface for HTTP
type HTTPServer struct {
	server *http.Server
}

// Start starts the HTTP server
func (s *HTTPServer) Start(ctx context.Context) error {
	go func() {
		_ = s.server.ListenAndServe()
	}()
	return nil
}

// Stop stops the HTTP server gracefully
func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Addr returns the server address
func (s *HTTPServer) Addr() string {
	return s.server.Addr
}

// ProvideHTTPServer creates an HTTP server that implements the Server interface
func ProvideHTTPServer(cfg *config.Config, router chi.Router) server.Server {
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

	return &HTTPServer{server: srv}
}

// ProvideHandlers creates HTTP handlers with proper dependencies
func ProvideHandlers(service *application.URLService, cfg *config.Config, repo domain.URLRepository) *httpAdapter.Handlers {
	return httpAdapter.NewHandlers(service, cfg.App.BaseURL, repo)
}
