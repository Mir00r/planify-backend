package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Error     string      `json:"error,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, status int, message string, opts ...ErrorOption) {
	response := ErrorResponse{
		Status:    status,
		Message:   message,
		RequestID: c.GetString("RequestID"), // Assuming request ID middleware is in place
	}

	// Apply any optional configurations
	for _, opt := range opts {
		opt(&response)
	}

	c.JSON(status, response)
}

// ErrorOption defines a function type for configuring error responses
type ErrorOption func(*ErrorResponse)

// WithError adds error details to the response
func WithError(err error) ErrorOption {
	return func(r *ErrorResponse) {
		if err != nil {
			r.Error = err.Error()
		}
	}
}

// WithDetails adds additional details to the response
func WithDetails(details interface{}) ErrorOption {
	return func(r *ErrorResponse) {
		r.Details = details
	}
}

// Common error response helpers
func RespondWithBadRequest(c *gin.Context, message string, opts ...ErrorOption) {
	RespondWithError(c, http.StatusBadRequest, message, opts...)
}

func RespondWithUnauthorized(c *gin.Context, message string, opts ...ErrorOption) {
	if message == "" {
		message = "Unauthorized"
	}
	RespondWithError(c, http.StatusUnauthorized, message, opts...)
}

func RespondWithForbidden(c *gin.Context, message string, opts ...ErrorOption) {
	if message == "" {
		message = "Forbidden"
	}
	RespondWithError(c, http.StatusForbidden, message, opts...)
}

func RespondWithNotFound(c *gin.Context, message string, opts ...ErrorOption) {
	if message == "" {
		message = "Resource not found"
	}
	RespondWithError(c, http.StatusNotFound, message, opts...)
}

func RespondWithInternalError(c *gin.Context, message string, opts ...ErrorOption) {
	if message == "" {
		message = "Internal server error"
	}
	RespondWithError(c, http.StatusInternalServerError, message, opts...)
}

// IsHopByHopHeader determines if a header is hop-by-hop
func IsHopByHopHeader(header string) bool {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	return hopByHopHeaders[header]
}
