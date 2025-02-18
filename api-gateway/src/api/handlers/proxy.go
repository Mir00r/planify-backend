package handlers

import (
	"bytes"
	_ "context"
	"fmt"
	"io"
	"net/http"
	_ "net/url"
	"strings"
	"time"

	"github.com/Mir00r/api-gateway/src/api-gateway/src/services"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProxyHandler handles proxying requests to backend services
type ProxyHandler struct {
	serviceRegistry *services.ServiceRegistry
	logger          *zap.Logger
	httpClient      *http.Client
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(serviceRegistry *services.ServiceRegistry, logger *zap.Logger) *ProxyHandler {
	return &ProxyHandler{
		serviceRegistry: serviceRegistry,
		logger:          logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProxyRequest handles proxying requests to the appropriate service
func (h *ProxyHandler) ProxyRequest(c *gin.Context) {
	// Extract service name from the URL path
	serviceName, path := h.extractServiceInfo(c.Request.URL.Path)

	// Get service instance from registry
	service, err := h.serviceRegistry.GetService(serviceName)
	if err != nil {
		h.logger.Error("service not found",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		utils.RespondWithError(c, http.StatusNotFound, "Service not found")
		return
	}

	// Check if service is healthy
	if !service.IsHealthy {
		h.logger.Warn("service unhealthy",
			zap.String("service", serviceName),
		)
		utils.RespondWithError(c, http.StatusServiceUnavailable, "Service unavailable")
		return
	}

	// Create target URL
	targetURL := fmt.Sprintf("%s%s", service.BaseURL, path)
	if c.Request.URL.RawQuery != "" {
		targetURL = fmt.Sprintf("%s?%s", targetURL, c.Request.URL.RawQuery)
	}

	// Create proxy request
	proxyReq, err := h.createProxyRequest(c, targetURL)
	if err != nil {
		h.logger.Error("failed to create proxy request",
			zap.Error(err),
			zap.String("target", targetURL),
		)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to proxy request")
		return
	}

	// Execute proxy request
	resp, err := h.httpClient.Do(proxyReq)
	if err != nil {
		h.logger.Error("proxy request failed",
			zap.Error(err),
			zap.String("target", targetURL),
		)
		utils.RespondWithError(c, http.StatusBadGateway, "Failed to reach service")
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	h.copyHeaders(c, resp.Header)

	// Copy response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("failed to read response body",
			zap.Error(err),
			zap.String("target", targetURL),
		)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to process response")
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// extractServiceInfo extracts service name and path from the URL
func (h *ProxyHandler) extractServiceInfo(fullPath string) (string, string) {
	parts := strings.SplitN(strings.TrimPrefix(fullPath, "/"), "/", 2)
	if len(parts) < 2 {
		return parts[0], "/"
	}
	return parts[0], "/" + parts[1]
}

// createProxyRequest creates a new HTTP request for proxying
func (h *ProxyHandler) createProxyRequest(c *gin.Context, targetURL string) (*http.Request, error) {
	var bodyReader io.Reader
	if c.Request.Body != nil {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(body)
	}

	proxyReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		targetURL,
		bodyReader,
	)
	if err != nil {
		return nil, err
	}

	// Copy headers
	h.copyRequestHeaders(proxyReq, c.Request)

	// Add proxy-specific headers
	proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())
	proxyReq.Header.Set("X-Original-URI", c.Request.RequestURI)

	return proxyReq, nil
}

// copyRequestHeaders copies headers from original request to proxy request
func (h *ProxyHandler) copyRequestHeaders(dst *http.Request, src *http.Request) {
	for key, values := range src.Header {
		// Skip hop-by-hop headers
		if utils.IsHopByHopHeader(key) {
			continue
		}
		dst.Header[key] = values
	}
}

// copyHeaders copies response headers to the client response
func (h *ProxyHandler) copyHeaders(c *gin.Context, headers http.Header) {
	for key, values := range headers {
		// Skip hop-by-hop headers
		if utils.IsHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			c.Header(key, value)
		}
	}
}
