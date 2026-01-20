package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

func SetupDeploymentRoutes(r *gin.Engine, handlers *handlers.Handlers) {
	r.POST("/deploy", handlers.DeploymentHandler.HandleDeployment)
}
