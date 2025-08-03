package fx

import (
	"go.uber.org/fx"

	httpFX "github.com/sp3dr4/dove/internal/fx/http"
)

// HTTPServerModules combines all modules needed for HTTP server entrypoint
var HTTPServerModules = fx.Options(
	CoreModules,
	httpFX.HTTPModule,
	httpFX.HTTPLifecycleModule,
)
