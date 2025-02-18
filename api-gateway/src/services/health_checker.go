// services/health_checker.go

package services

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthChecker handles health checking of registered services
type HealthChecker struct {
	logger     *zap.Logger
	httpClient *http.Client
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(logger *zap.Logger) *HealthChecker {
	return &HealthChecker{
		logger: logger,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				MaxConnsPerHost:     10,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

// StartChecks begins periodic health checking of services
func (hc *HealthChecker) StartChecks(ctx context.Context, interval time.Duration, registry *ServiceRegistry) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				hc.checkServices(registry)
			}
		}
	}()
}

// checkServices performs health checks on all registered services
func (hc *HealthChecker) checkServices(registry *ServiceRegistry) {
	services := registry.ListServices()

	for _, service := range services {
		go hc.checkService(service, registry)
	}
}

// checkService performs a health check on a single service
func (hc *HealthChecker) checkService(service *ServiceInstance, registry *ServiceRegistry) {
	startTime := time.Now()

	healthURL := service.BaseURL + service.HealthURL
	req, err := http.NewRequest(http.MethodGet, healthURL, nil)
	if err != nil {
		hc.logger.Error("failed to create health check request",
			zap.String("service", service.Name),
			zap.Error(err),
		)
		registry.UpdateServiceHealth(service.Name, false, 0)
		return
	}

	resp, err := hc.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		hc.logger.Warn("health check failed",
			zap.String("service", service.Name),
			zap.Error(err),
			zap.Duration("response_time", responseTime),
		)
		registry.UpdateServiceHealth(service.Name, false, responseTime)
		return
	}
	defer resp.Body.Close()

	isHealthy := resp.StatusCode == http.StatusOK
	registry.UpdateServiceHealth(service.Name, isHealthy, responseTime)

	if !isHealthy {
		hc.logger.Warn("service reported unhealthy status",
			zap.String("service", service.Name),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("response_time", responseTime),
		)
	} else {
		hc.logger.Debug("health check successful",
			zap.String("service", service.Name),
			zap.Duration("response_time", responseTime),
		)
	}
}

// CheckServiceHealth performs an immediate health check on a specific service
func (hc *HealthChecker) CheckServiceHealth(service *ServiceInstance, registry *ServiceRegistry) bool {
	startTime := time.Now()

	healthURL := service.BaseURL + service.HealthURL
	resp, err := hc.httpClient.Get(healthURL)
	responseTime := time.Since(startTime)

	if err != nil {
		hc.logger.Warn("immediate health check failed",
			zap.String("service", service.Name),
			zap.Error(err),
		)
		registry.UpdateServiceHealth(service.Name, false, responseTime)
		return false
	}
	defer resp.Body.Close()

	isHealthy := resp.StatusCode == http.StatusOK
	registry.UpdateServiceHealth(service.Name, isHealthy, responseTime)

	return isHealthy
}
