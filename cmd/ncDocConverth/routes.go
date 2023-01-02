package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"rpjosh.de/ncDocConverter/internal/api"
)

func (app *WebApplication) routes() http.Handler {
	api := api.Api{Logger: app.logger, Config: app.config}

	router := chi.NewRouter()
	router.Use(middleware.RealIP, app.recoverPanic, app.logRequest, secureHeaders)

	api.SetupServer(router)

	return router
}
