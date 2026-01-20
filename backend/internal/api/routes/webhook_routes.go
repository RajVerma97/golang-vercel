package routes

import (
	"github.com/RajVerma97/golang-vercel/backend/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

func SetupWebhookRoutes(r *gin.Engine, handlers *handlers.Handlers) {
	r.POST("/webhook/github", handlers.WebhookHandler.HandleGitHubWebhook)
}
