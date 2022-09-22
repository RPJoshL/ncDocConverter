package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"rpjosh.de/ncDocConverter/internal/api"
	"rpjosh.de/ncDocConverter/internal/frontend"
)

func (app *WebApplication) routes() http.Handler {
	frontend := frontend.Frontend{Logger: app.logger, Config: app.config}
	api := api.Api{Logger: app.logger, Config: app.config}

	router := chi.NewRouter()
	router.Use(middleware.RealIP, app.recoverPanic, app.logRequest, secureHeaders)

	frontend.SetupServer(router)
	api.SetupServer(router)

	return router
}