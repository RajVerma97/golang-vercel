package handlers

import (
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/api/errors"
	"github.com/RajVerma97/golang-vercel/backend/internal/api/requests"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DeploymentHandlerConfig struct {
	services *services.Services
}
type DeploymentHandler struct {
	services *services.Services
}

func NewDeploymentHandler(config *DeploymentHandlerConfig) *DeploymentHandler {
	return &DeploymentHandler{services: config.services}
}

func (h *DeploymentHandler) HandleDeployment(c *gin.Context) {
	if h.services.RedisService == nil {
		logger.Error("redis servcie is nil", nil)
		return
	}
	// bind json
	var request requests.DeployRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		ErrorResponse(c, errors.NewBadRequestError("Invalid Request"))
		return
	}

	// validate
	if err := request.Validate(); err != nil {
		ErrorResponse(c, err)
		return
	}

	logger.Debug("", zap.Any("request", request))

	now := time.Now()
	err := h.services.RedisService.EnqueueBuild(c.Request.Context(), &dto.Build{
		RepoUrl:    request.RepoURL,
		Branch:     request.Branch,
		CommitHash: request.CommitHash,
		Status:     constants.BuildStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	if err != nil {
		ErrorResponse(c, err)
		return
	}

	SuccessResponse(c, true)
}
