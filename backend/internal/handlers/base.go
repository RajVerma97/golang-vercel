package handlers

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
)

type Handlers struct {
	BuildHandler *BuildHandler
}

func NewHandlers(services *services.Services) *Handlers {
	buildHandler := NewBuildHandler(&BuildHandlerConfig{
		services: services,
	})
	return &Handlers{
		BuildHandler: buildHandler,
	}
}
