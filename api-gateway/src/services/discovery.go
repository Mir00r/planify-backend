// services/discovery.go

package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	"go.uber.org/zap"
)

// ServiceDiscovery handles service discovery and health monitoring
type ServiceDiscovery struct {
	registry    *ServiceRegistry
	logger      *zap.Logger
	config      *config.ServicesConfig
	healthCheck *HealthChecker
	mu          sync.RWMutex
}

// ServiceInfo contains service metadata and health status
type ServiceInfo struct {
	Name         string
	URL          string
	Status       ServiceStatus
	LastChecked  time.Time
	ResponseTime time.Duration
	ErrorCount   int64
	SuccessCount int64
}

type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
)

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(cfg *config.ServicesConfig, logger *zap.Logger) *ServiceDiscovery {
	sd := &ServiceDiscovery{
		registry:    NewServiceRegistry(cfg, logger),
		logger:      logger,
		config:      cfg,
		healthCheck: NewHealthChecker(logger),
	}

	return sd
}

// Start begins the service discovery process
func (sd *ServiceDiscovery) Start(ctx context.Context) error {
	// Initialize service registry
	if err := sd.initializeServices(); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Start health checks
	sd.startHealthChecks(ctx)

	return nil
}

// initializeServices registers all configured services
func (sd *ServiceDiscovery) initializeServices() error {
	services := []struct {
		name   string
		config config.ServiceConfig
	}{
		{"user-service", sd.config.UserService},
		{"notification-service", sd.config.NotificationService},
		{"appointment-service", sd.config.AppointmentService},
	}

	for _, svc := range services {
		if err := sd.registerService(svc.name, svc.config); err != nil {
			return fmt.Errorf("failed to register service %s: %w", svc.name, err)
		}
	}

	return nil
}

// registerService registers a new service
func (sd *ServiceDiscovery) registerService(name string, cfg config.ServiceConfig) error {
	//serviceInfo := &ServiceInfo{
	//	Name:   name,
	//	URL:    cfg.BaseURL,
	//	Status: StatusUnknown,
	//}

	sd.mu.Lock()
	defer sd.mu.Unlock()

	if err := sd.registry.RegisterService(name, &ServiceInstance{
		Name:      name,
		BaseURL:   cfg.BaseURL,
		HealthURL: cfg.HealthCheck,
	}); err != nil {
		return err
	}

	sd.logger.Info("service registered",
		zap.String("name", name),
		zap.String("url", cfg.BaseURL),
	)

	return nil
}

// GetService returns information about a specific service
func (sd *ServiceDiscovery) GetService(name string) (*ServiceInfo, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	service, err := sd.registry.GetService(name)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	return &ServiceInfo{
		Name:         service.Name,
		URL:          service.BaseURL,
		Status:       sd.getServiceStatus(service),
		LastChecked:  service.LastChecked,
		ResponseTime: service.ResponseTime,
	}, nil
}

// startHealthChecks begins periodic health checking of services
func (sd *ServiceDiscovery) startHealthChecks(ctx context.Context) {
	interval := time.Duration(sd.config.HealthCheckInterval) * time.Second
	sd.healthCheck.StartChecks(ctx, interval, sd.registry)
}

// getServiceStatus determines the current status of a service
func (sd *ServiceDiscovery) getServiceStatus(svc *ServiceInstance) ServiceStatus {
	if !svc.IsHealthy {
		return StatusUnhealthy
	}
	if time.Since(svc.LastChecked) > time.Minute*5 {
		return StatusUnknown
	}
	return StatusHealthy
}
