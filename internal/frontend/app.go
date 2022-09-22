package frontend

import (
	"text/template"

	"github.com/go-chi/chi/v5"
	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/pkg/logger"
)

// Contains the shared dependencies needed for the WebApplication
type Frontend struct {
	Logger        *logger.Logger
	Config        *models.WebConfig
	templateCache map[string]*template.Template
}

func (app *Frontend) SetupServer(router *chi.Mux) {
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Fatal("Failed to parse the templates", err)
	}
	app.templateCache = templateCache

	app.setServerConfiguration()

	app.routes(router)
}
