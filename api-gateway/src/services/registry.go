// services/registry.go

package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	"go.uber.org/zap"
)

// ServiceInstance represents a registered service instance
type ServiceInstance struct {
	Name         string
	BaseURL      string
	HealthURL    string
	IsHealthy    bool
	LastChecked  time.Time
	ResponseTime time.Duration
	ErrorCount   int64
	SuccessCount int64
}

// ServiceRegistry manages service registration and discovery
type ServiceRegistry struct {
	services map[string]*ServiceInstance
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(config *config.ServicesConfig, logger *zap.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*ServiceInstance),
		logger:   logger,
	}
}

// RegisterService registers a new service instance
func (sr *ServiceRegistry) RegisterService(name string, instance *ServiceInstance) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}

	sr.services[name] = instance
	sr.logger.Info("service registered",
		zap.String("name", name),
		zap.String("url", instance.BaseURL),
	)

	return nil
}

// GetService retrieves a service instance by name
func (sr *ServiceRegistry) GetService(name string) (*ServiceInstance, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, exists := sr.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// UpdateServiceHealth updates the health status of a service
func (sr *ServiceRegistry) UpdateServiceHealth(name string, isHealthy bool, responseTime time.Duration) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if service, exists := sr.services[name]; exists {
		service.IsHealthy = isHealthy
		service.LastChecked = time.Now()
		service.ResponseTime = responseTime

		if isHealthy {
			service.SuccessCount++
		} else {
			service.ErrorCount++
		}
	}
}

// ListServices returns all registered services
func (sr *ServiceRegistry) ListServices() []*ServiceInstance {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	services := make([]*ServiceInstance, 0, len(sr.services))
	for _, service := range sr.services {
		services = append(services, service)
	}

	return services
}

// DeregisterService removes a service from the registry
func (sr *ServiceRegistry) DeregisterService(name string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}

	delete(sr.services, name)
	sr.logger.Info("service deregistered", zap.String("name", name))

	return nil
}
