package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func InitRouter(services *services.Services) *gin.Engine {
	router := gin.New()
	SetupBuildRoutes(router, services)
	return router
}
