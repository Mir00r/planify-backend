package src

import (
	"fmt"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/pkg/metrics"
	"github.com/Mir00r/api-gateway/src/api-gateway/src/services"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize metrics collector
	metrics.GetCollector()

	// Initialize service registry
	registry, err := services.NewRegistry(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize service registry", zap.Error(err))
	}

	// Initialize handlers
	handlers := handlers.NewHandlers(cfg, registry, logger)

	// Initialize router
	router := api.NewRouter(cfg, handlers)
	router.Setup()

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router.GetEngine(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSecs) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSecs) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ShutdownTimeoutSecs)*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// initLogger initializes the zap logger
func initLogger() (*zap.Logger, error) {
	env := os.Getenv("APP_ENV")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

// loadConfig loads the application configuration
func loadConfig() (*config.Config, error) {
	configPath := config.GetDefaultConfigPath()
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	loader := config.NewConfigLoader(configPath)
	return loader.Load()
}
