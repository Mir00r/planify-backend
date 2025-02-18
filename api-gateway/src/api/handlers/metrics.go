// api/handlers/metrics.go

package handlers

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"sync"
)

// MetricsHandler handles metrics collection and exposure
type MetricsHandler struct {
	logger  *zap.Logger
	metrics *MetricsCollector
}

// MetricsCollector holds all Prometheus metrics
type MetricsCollector struct {
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	activeRequests  *prometheus.GaugeVec
	errorCounter    *prometheus.CounterVec
	serviceHealth   *prometheus.GaugeVec
}

var (
	once      sync.Once
	collector *MetricsCollector
)

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(logger *zap.Logger) *MetricsHandler {
	once.Do(func() {
		collector = initializeMetrics()
	})

	return &MetricsHandler{
		logger:  logger,
		metrics: collector,
	}
}

// initializeMetrics initializes all Prometheus metrics
func initializeMetrics() *MetricsCollector {
	m := &MetricsCollector{
		requestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_total",
				Help: "Total number of requests processed by the API Gateway",
			},
			[]string{"method", "path", "service", "status"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"method", "path", "service"},
		),

		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_response_size_bytes",
				Help:    "Size of responses in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 2, 10),
			},
			[]string{"method", "path", "service"},
		),

		activeRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_active_requests",
				Help: "Number of active requests being processed",
			},
			[]string{"method", "service"},
		),

		errorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_errors_total",
				Help: "Total number of errors encountered",
			},
			[]string{"method", "path", "service", "error_type"},
		),

		serviceHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_service_health",
				Help: "Health status of backend services (1 = healthy, 0 = unhealthy)",
			},
			[]string{"service"},
		),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(
		m.requestCounter,
		m.requestDuration,
		m.responseSize,
		m.activeRequests,
		m.errorCounter,
		m.serviceHealth,
	)

	return m
}
