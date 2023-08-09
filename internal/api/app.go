package api

import (
	"github.com/go-chi/chi/v5"

	"git.rpjosh.de/RPJosh/go-logger"
	"git.rpjosh.de/ncDocConverter/internal/models"
)

type Api struct {
	Logger *logger.Logger
	Config *models.WebConfig
}

func (api *Api) SetupServer(router *chi.Mux) {
	api.routes(router)
}
