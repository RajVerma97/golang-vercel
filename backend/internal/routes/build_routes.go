package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/handlers"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func SetupBuildRoutes(r *gin.Engine, services *services.Services) {
	handlers := handlers.NewHandlers(services)
	r.POST("/deploy", handlers.BuildHandler.HandleBuild)
}
