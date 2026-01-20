package handlers

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
)

type Handlers struct {
	BuildHandler      *BuildHandler
	DeploymentHandler *DeploymentHandler
	WebhookHandler    *WebhookHandler
}

func NewHandlers(services *services.Services) *Handlers {
	buildHandler := NewBuildHandler(&BuildHandlerConfig{
		services: services,
	})
	deploymentHandler := NewDeploymentHandler(&DeploymentHandlerConfig{
		services: services,
	})
	WebhookHandler := NewWebhookHandler(services, "123")
	return &Handlers{
		BuildHandler:      buildHandler,
		DeploymentHandler: deploymentHandler,
		WebhookHandler:    WebhookHandler,
	}
}
