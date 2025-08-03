package http

import (
	"go.uber.org/fx"

	httpAdapter "github.com/sp3dr4/dove/internal/adapters/http"
)

// HTTPModule provides HTTP-related dependencies
var HTTPModule = fx.Module("http",
	fx.Provide(ProvideHandlers),
	fx.Provide(httpAdapter.NewRouter),
	fx.Provide(ProvideHTTPServer),
)

// HTTPLifecycleModule provides HTTP server lifecycle management
var HTTPLifecycleModule = fx.Module("http-lifecycle",
	fx.Invoke(RegisterHTTPServerHooks),
)
