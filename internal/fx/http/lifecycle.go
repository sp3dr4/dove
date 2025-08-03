package http

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/server"
)

// ServerParams holds the parameters needed for HTTP server lifecycle management
type ServerParams struct {
	fx.In

	Server server.Server
	Config *config.Config
	Logger *slog.Logger
}

// RegisterHTTPServerHooks registers HTTP server lifecycle hooks with FX
func RegisterHTTPServerHooks(lc fx.Lifecycle, params ServerParams) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			params.Logger.Info("Starting HTTP server",
				"addr", params.Server.Addr(),
				"database", params.Config.Database.Type,
				"base_url", params.Config.App.BaseURL,
			)
			return params.Server.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			params.Logger.Info("Shutting down HTTP server...")
			if err := params.Server.Stop(ctx); err != nil {
				params.Logger.Error("Failed to shutdown HTTP server", "error", err)
				return err
			}
			params.Logger.Info("HTTP server shutdown completed")
			return nil
		},
	})
}
