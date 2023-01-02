package api

import (
	"github.com/go-chi/chi/v5"

	"rpjosh.de/ncDocConverter/internal/models"
	"rpjosh.de/ncDocConverter/pkg/logger"
)

type Api struct {
	Logger 	*logger.Logger
	Config 	*models.WebConfig
}

func (api *Api) SetupServer(router *chi.Mux) {
	api.routes(router)
}