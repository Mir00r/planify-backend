// config/config.go

package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config holds all configuration for our application
type Config struct {
	Server   ServerConfig
	Services ServicesConfig
	Auth     AuthConfig
	Redis    RedisConfig
}

// ServerConfig holds all server-related configuration
type ServerConfig struct {
	Port                int
	ReadTimeoutSecs     int
	WriteTimeoutSecs    int
	ShutdownTimeoutSecs int
}

// ServicesConfig holds configuration for downstream services
type ServicesConfig struct {
	UserService         ServiceConfig
	NotificationService ServiceConfig
	AppointmentService  ServiceConfig
	HealthCheckInterval int // Time in seconds between health checks
}

// ServiceConfig holds configuration for a single service
type ServiceConfig struct {
	BaseURL     string
	Timeout     int
	RetryCount  int
	HealthCheck string
}

// AuthConfig holds authentication-related configuration
type AuthConfig struct {
	JWTSecret       string
	TokenExpirySecs int
	IssuerURL       string
}

// RedisConfig holds Redis-related configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// LoadConfig loads configuration from files and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default configurations
	setDefaults(v)

	// Read the config file
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Enable environment variable override
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.readTimeoutSecs", 30)
	v.SetDefault("server.writeTimeoutSecs", 30)
	v.SetDefault("server.shutdownTimeoutSecs", 30)

	// Service defaults
	v.SetDefault("services.userService.timeout", 5)
	v.SetDefault("services.userService.retryCount", 3)

	// Auth defaults
	v.SetDefault("auth.tokenExpirySecs", 3600)

	// Add health check interval default
	v.SetDefault("services.healthCheckInterval", 30) // Check every 30 seconds by default
}
