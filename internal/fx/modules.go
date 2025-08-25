package fx

import (
	"go.uber.org/fx"

	"github.com/sp3dr4/dove/config"
	"github.com/sp3dr4/dove/internal/application"
)

// ConfigModule provides configuration-related dependencies
var ConfigModule = fx.Module("config",
	fx.Provide(config.Load),
)

// InfrastructureModule provides infrastructure-related dependencies
var InfrastructureModule = fx.Module("infrastructure",
	fx.Provide(ProvideLogger),
	fx.Provide(ProvideRepository),
	fx.Provide(ProvideRedisClient),
	fx.Provide(ProvideCache),
	fx.Provide(ProvideCacheTTL),
)

// ApplicationModule provides application service dependencies
var ApplicationModule = fx.Module("application",
	fx.Provide(application.NewURLService),
)

// MetricsModule provides metrics-related dependencies
var MetricsModule = fx.Module("metrics",
	fx.Provide(ProvideMetricsRegistry),
)

// CoreLifecycleModule provides core lifecycle management (shared by all entrypoints)
var CoreLifecycleModule = fx.Module("core-lifecycle",
	fx.Invoke(RegisterRepositoryHooks),
	fx.Invoke(RegisterCacheHooks),
)

// CoreModules combines the core modules shared by all entrypoints
var CoreModules = fx.Options(
	ConfigModule,
	InfrastructureModule,
	ApplicationModule,
	MetricsModule,
	CoreLifecycleModule,
)
