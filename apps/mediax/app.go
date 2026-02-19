package mediax

import (
	"github.com/getevo/evo/v2"
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/db"
	"github.com/getevo/restify"
	"mediax/apps/media"
)

type App struct {
}

func (a App) Register() error {
	restify.SetPrefix("/admin")
	db.UseModel(media.Project{}, media.Storage{}, media.Origin{}, media.VideoProfile{})
	return nil
}

func (a App) Router() error {
	var controller Controller
	evo.Get("/health", controller.Health)
	evo.Post("/admin/reload", controller.Reload)
	evo.Get("/prometheus/metrics", controller.PrometheusMetrics)
	evo.Get("/*", controller.ServeMedia)
	return nil
}

func (a App) WhenReady() error {
	InitializeConfig()
	startEvictionLoop()
	return nil
}

func (a App) Priority() application.Priority {
	return application.LOWEST
}

func (a App) Name() string {
	return "mediax"
}
