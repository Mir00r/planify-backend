// config/loader.go

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigLoader handles configuration loading and validation
type ConfigLoader struct {
	configPath string
	config     *Config
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader(configPath string) *ConfigLoader {
	return &ConfigLoader{
		configPath: configPath,
	}
}

// Load loads and validates the configuration
func (cl *ConfigLoader) Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s", cl.configPath)
	}

	// Load configuration
	config, err := LoadConfig(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cl.validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cl.config = config
	return config, nil
}

// validateConfig performs validation on the loaded configuration
func (cl *ConfigLoader) validateConfig(config *Config) error {
	// Validate Server Configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate Services Configuration
	if err := cl.validateServices(config.Services); err != nil {
		return fmt.Errorf("services validation failed: %w", err)
	}

	// Validate Auth Configuration
	if config.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	return nil
}

// validateServices validates service-specific configurations
func (cl *ConfigLoader) validateServices(services ServicesConfig) error {
	// Validate User Service
	if services.UserService.BaseURL == "" {
		return fmt.Errorf("user service BaseURL is required")
	}

	// Validate Notification Service
	if services.NotificationService.BaseURL == "" {
		return fmt.Errorf("notification service BaseURL is required")
	}

	// Validate Appointment Service
	if services.AppointmentService.BaseURL == "" {
		return fmt.Errorf("appointment service BaseURL is required")
	}

	return nil
}

// GetDefaultConfigPath returns the default configuration path
func GetDefaultConfigPath() string {
	// Check for environment-specific config
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	return filepath.Join("config", fmt.Sprintf("config.%s.yaml", env))
}
