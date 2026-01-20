package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/api/handlers"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func InitRouter(services *services.Services) *gin.Engine {
	router := gin.New()
	handlers := handlers.NewHandlers(services)

	// SetupBuildRoutes(router, services)
	SetupDeploymentRoutes(router, handlers)
	SetupWebhookRoutes(router, handlers)
	return router
}
