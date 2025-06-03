package main

import (
	"github.com/getevo/docify"
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/restify"
	"mediax/apps/mediax"
)

func main() {
	evo.Setup()

	var apps = application.GetInstance()
	// Register all application modules
	apps.Register( // Authentication follows
		mediax.App{},
		restify.App{},
		docify.App{},
	)

	// Start the application
	evo.Run()
}
