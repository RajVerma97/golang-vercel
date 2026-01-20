package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/api/requests"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	webhookSecret string
	services      *services.Services
}

func NewWebhookHandler(services *services.Services, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		webhookSecret: webhookSecret,
		services:      services,
	}
}

type GitHubPushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
	// 1. Verify signature if secret is configured
	if h.webhookSecret != "" {
		signature := c.GetHeader("X-Hub-Signature-256")
		body, _ := io.ReadAll(c.Request.Body)

		if !h.verifySignature(body, signature) {
			logger.Error("GitHub Webhook: Invalid Signature detected!", nil,
				zap.String("received_sig", signature),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		// Re-populate body for binding
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	// 2. Parse GitHub payload
	var payload GitHubPushPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 3. Extract branch from ref (refs/heads/main -> main)
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	// 4. Create deploy request
	request := requests.DeployRequest{
		RepoURL:    payload.Repository.CloneURL,
		Branch:     &branch,
		CommitHash: &payload.After,
	}

	// 5. Validate
	if err := request.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Debug("", zap.Any("github_webhook_requst", request))
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
func (h *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	if h.webhookSecret == "" {
		logger.Warn("GitHub Webhook: Secret not configured, skipping verification")
		return true // Or false depending on your security policy
	}

	if signature == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	// Remove the "sha256=" prefix
	actualSig := signature[7:]
	sigBytes, err := hex.DecodeString(actualSig)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(payload)
	expectedMac := mac.Sum(nil)

	return hmac.Equal(sigBytes, expectedMac)
}
