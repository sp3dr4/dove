// Package main implements a simple URL shortener service.
//
//	@title			Dove URL Shortener API
//	@version		1.0
//	@description	A fast and simple URL shortener service
//	@host			localhost:8080
//	@BasePath		/
//	@schemes		http https
package main

import (
	"time"

	"go.uber.org/fx"

	_ "github.com/sp3dr4/dove/docs"
	fxProviders "github.com/sp3dr4/dove/internal/fx"
)

func main() {
	app := fx.New(
		fxProviders.HTTPServerModules,
		fx.StopTimeout(30*time.Second),
	)

	app.Run()
}
