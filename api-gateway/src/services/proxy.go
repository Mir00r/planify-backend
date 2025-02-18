// services/proxy.go

package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	"go.uber.org/zap"
)

// ProxyService handles proxying requests to backend services
type ProxyService struct {
	client    *http.Client
	discovery *ServiceDiscovery
	logger    *zap.Logger
	config    *config.ServicesConfig
}

// ProxyRequest contains information about the request to be proxied
type ProxyRequest struct {
	Method      string
	Path        string
	Body        io.Reader
	Headers     http.Header
	ServiceName string
	Timeout     time.Duration
	RetryCount  int
	Context     context.Context
}

// ProxyResponse contains the response from the proxied request
type ProxyResponse struct {
	StatusCode   int
	Headers      http.Header
	Body         []byte
	Error        error
	ResponseTime time.Duration
}

// NewProxyService creates a new proxy service
func NewProxyService(cfg *config.ServicesConfig, discovery *ServiceDiscovery, logger *zap.Logger) *ProxyService {
	return &ProxyService{
		client: &http.Client{
			Timeout: time.Second * 30,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				MaxConnsPerHost:     100,
				MaxIdleConnsPerHost: 100,
			},
		},
		discovery: discovery,
		logger:    logger,
		config:    cfg,
	}
}

// ProxyRequest handles proxying a request to a backend service
func (p *ProxyService) ProxyRequest(req *ProxyRequest) (*ProxyResponse, error) {
	startTime := time.Now()

	// Get service information
	service, err := p.discovery.GetService(req.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("service discovery error: %w", err)
	}

	// Build target URL
	targetURL, err := url.Parse(fmt.Sprintf("%s%s", service.URL, req.Path))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Create proxied request
	proxyReq, err := http.NewRequestWithContext(
		req.Context,
		req.Method,
		targetURL.String(),
		req.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	p.copyHeaders(proxyReq, req.Headers)

	// Execute request with retry
	response, err := p.executeWithRetry(proxyReq, req.RetryCount)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Read response body
	responseBody, err := io.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	responseTime := time.Since(startTime)

	// Log request details
	p.logger.Info("proxy request completed",
		zap.String("service", req.ServiceName),
		zap.String("method", req.Method),
		zap.String("path", req.Path),
		zap.Int("status", response.StatusCode),
		zap.Duration("response_time", responseTime),
	)

	return &ProxyResponse{
		StatusCode:   response.StatusCode,
		Headers:      response.Header,
		Body:         responseBody,
		ResponseTime: responseTime,
	}, nil
}

// executeWithRetry executes a request with retry logic
func (p *ProxyService) executeWithRetry(req *http.Request, retryCount int) (*http.Response, error) {
	var lastErr error

	for i := 0; i <= retryCount; i++ {
		// Clone request body for retry
		var bodyReader io.Reader
		if req.Body != nil {
			bodyData, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyData))
			bodyReader = bytes.NewReader(bodyData)
		}

		response, err := p.client.Do(req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		p.logger.Warn("request failed, retrying",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		// Reset request body for retry
		if bodyReader != nil {
			req.Body = io.NopCloser(bodyReader)
		}

		// Wait before retry
		if i < retryCount {
			time.Sleep(time.Second * time.Duration(i+1))
		}
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// copyHeaders copies headers from source to destination
func (p *ProxyService) copyHeaders(dst *http.Request, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}
}
