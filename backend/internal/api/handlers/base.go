package handlers

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/helpers"
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
	webhookHandler := NewWebhookHandler(services, helpers.GetEnv("GITHUB_WEBHOOK_SECRET", ""))

	return &Handlers{
		BuildHandler:      buildHandler,
		DeploymentHandler: deploymentHandler,
		WebhookHandler:    webhookHandler,
	}
}
