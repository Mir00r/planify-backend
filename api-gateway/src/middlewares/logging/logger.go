// middleware/logging/logger.go

package logging

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger implements request logging middleware
type RequestLogger struct {
	logger *zap.Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger *zap.Logger) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// LogRequest logs incoming HTTP requests
func (rl *RequestLogger) LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details after processing
		latency := time.Since(start)

		// Get error if one occurred
		var errorMessage string
		if len(c.Errors) > 0 {
			errorMessage = c.Errors.String()
		}

		// Log request details
		rl.logger.Info("request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("error", errorMessage),
			zap.Int("bytes_out", c.Writer.Size()),
		)
	}
}

// LogError logs error details
func (rl *RequestLogger) LogError() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log detailed error information if any occurred
		for _, err := range c.Errors {
			fields := []zap.Field{
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Error(err.Err),
			}

			// Safely add meta information if it exists
			if err.Meta != nil {
				fields = append(fields, zap.Any("meta", err.Meta))
			}

			// Add error type as a string
			//fields = append(fields, zap.String("error_type", err.Type.Error()))

			rl.logger.Error("request error", fields...)
		}
	}
}

// CustomError represents a structured error with metadata
type CustomError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface
func (e *CustomError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}
