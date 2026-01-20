package handlers

import (
	"net/http"

	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type BuildHandlerConfig struct {
	services *services.Services
}
type BuildHandler struct {
	services *services.Services
}

func NewBuildHandler(config *BuildHandlerConfig) *BuildHandler {
	return &BuildHandler{services: config.services}
}

func (h *BuildHandler) HandleBuild(c *gin.Context) {
	response := gin.H{
		"msg": "hello",
	}
	c.JSON(http.StatusOK, response)
}
