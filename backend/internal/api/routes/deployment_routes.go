package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/api/handlers"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func SetupDeploymentRoutes(r *gin.Engine, services *services.Services) {
	handlers := handlers.NewHandlers(services)
	r.POST("/deploy", handlers.DeploymentHandler.HandleDeployment)
}
