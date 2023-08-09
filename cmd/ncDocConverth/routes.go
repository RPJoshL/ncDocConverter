package main

import (
	"net/http"

	"git.rpjosh.de/ncDocConverter/internal/api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *WebApplication) routes() http.Handler {
	api := api.Api{Logger: app.logger, Config: app.config}

	router := chi.NewRouter()
	router.Use(middleware.RealIP, app.recoverPanic, app.logRequest, secureHeaders)

	api.SetupServer(router)

	return router
}
