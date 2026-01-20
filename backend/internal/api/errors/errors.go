package errors

import (
	"net/http"
)

type ErrorType string

const (
	ErrorTypeInternal   ErrorType = "INTERNAL"
	ErrorTypeValidation ErrorType = "VALIDATION"
	ErrorTypeNotFound   ErrorType = "NOT_FOUND"
	ErrorTypeAuth       ErrorType = "AUTHENTICATION"
	ErrorTypeForbidden  ErrorType = "FORBIDDEN"
	ErrorTypeBadRequest ErrorType = "BAD_REQUEST"
)

type AppError struct {
	Type     ErrorType           `json:"-"`
	Code     string              `json:"code"`
	Message  string              `json:"message"`
	Detail   string              `json:"detail,omitempty"`
	Stack    string              `json:"stack,omitempty"`
	HTTPCode int                 `json:"-"`
	Raw      error               `json:"-"`
	Metadata interface{}         `json:"metadata,omitempty"`
	Errors   map[string][]string `json:"errors,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// NewError creates a new AppError
func NewError(errType ErrorType, code string, message string) *AppError {
	return &AppError{
		Type:     errType,
		Code:     code,
		Message:  message,
		HTTPCode: getHTTPCode(errType),
	}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:     ErrorTypeNotFound,
		Code:     "NOT_FOUND",
		Message:  message,
		HTTPCode: http.StatusNotFound,
	}
}

func NewValidationError(errors map[string][]string) *AppError {
	errorMessage := "The given data was invalid"
	for k := range errors {
		errorMessage = errors[k][0]
		break
	}

	return &AppError{
		Type:     ErrorTypeValidation,
		Message:  errorMessage,
		HTTPCode: http.StatusUnprocessableEntity,
		Errors:   errors,
	}
}

func NewValidationErrorWithMetadata(code string, message string, metadata interface{}) *AppError {
	return &AppError{
		Type:     ErrorTypeValidation,
		Code:     code,
		Message:  message,
		HTTPCode: http.StatusBadRequest,
		Metadata: metadata,
	}
}

func NewAuthenticationError(message string) *AppError {
	return &AppError{
		Type:     ErrorTypeAuth,
		Code:     "UNAUTHORIZED",
		Message:  message,
		HTTPCode: http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{
		Type:     ErrorTypeForbidden,
		Code:     "FORBIDDEN",
		Message:  message,
		HTTPCode: http.StatusForbidden,
	}
}

func NewInternalError(err error) *AppError {
	return &AppError{
		Type:     ErrorTypeInternal,
		Code:     "INTERNAL_ERROR",
		Message:  "An internal error occurred",
		Detail:   err.Error(),
		HTTPCode: http.StatusInternalServerError,
		Raw:      err,
	}
}

func NewBadRequestError(message string) *AppError {
	return &AppError{
		Type:     ErrorTypeBadRequest,
		Code:     "BAD_REQUEST",
		Message:  message,
		HTTPCode: http.StatusBadRequest,
	}
}

// getHTTPCode maps error types to HTTP status codes
func getHTTPCode(errType ErrorType) int {
	switch errType {
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeAuth:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeBadRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
