package handlers

import (
	"errors"
	"net/http"

	appErrors "github.com/RajVerma97/golang-vercel/backend/internal/api/errors"

	"github.com/gin-gonic/gin"
)

// Response represents a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// SuccessResponse sends a standardized success response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// ErrorResponse sends a standardized error response
func ErrorResponse(c *gin.Context, err error) {
	var appErr *appErrors.AppError
	if ok := errors.As(err, &appErr); ok {
		c.JSON(appErr.HTTPCode, Response{
			Success: false,
			Error:   appErr,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error:   err.Error(),
	})
}
