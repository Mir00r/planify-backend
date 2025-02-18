// services/metrics.go

package services

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/zap"
)

// MetricsService handles collection and reporting of metrics
type MetricsService struct {
	logger *zap.Logger
	mu     sync.RWMutex

	// Prometheus metrics
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	errorCounter    *prometheus.CounterVec
	serviceHealth   *prometheus.GaugeVec
}

// NewMetricsService creates a new metrics service
func NewMetricsService(logger *zap.Logger) *MetricsService {
	ms := &MetricsService{
		logger: logger,
	}

	ms.initializeMetrics()
	return ms
}

// initializeMetrics sets up all Prometheus metrics
func (ms *MetricsService) initializeMetrics() {
	ms.requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_requests_total",
			Help: "Total number of requests processed by the API Gateway",
		},
		[]string{"service", "method", "status"},
	)

	ms.requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_request_duration_seconds",
			Help:    "Duration of requests processed by the API Gateway",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // from 1ms to ~1s
		},
		[]string{"service", "method"},
	)

	ms.responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_response_size_bytes",
			Help:    "Size of responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), // from 100B to ~10MB
		},
		[]string{"service"},
	)

	ms.errorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"service", "error_type"},
	)

	ms.serviceHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_gateway_service_health",
			Help: "Health status of backend services (0=unhealthy, 1=healthy)",
		},
		[]string{"service"},
	)
}

// RecordRequest records metrics for a request
func (ms *MetricsService) RecordRequest(service, method string, statusCode int, duration time.Duration, responseSize int64) {
	status := ms.statusCodeToString(statusCode)

	ms.requestCounter.WithLabelValues(service, method, status).Inc()
	ms.requestDuration.WithLabelValues(service, method).Observe(duration.Seconds())
	ms.responseSize.WithLabelValues(service).Observe(float64(responseSize))

	if statusCode >= 500 {
		ms.errorCounter.WithLabelValues(service, "server_error").Inc()
	} else if statusCode >= 400 {
		ms.errorCounter.WithLabelValues(service, "client_error").Inc()
	}
}

// UpdateServiceHealth updates the health status of a service
func (ms *MetricsService) UpdateServiceHealth(service string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	ms.serviceHealth.WithLabelValues(service).Set(value)
}

// RecordError records a service error
func (ms *MetricsService) RecordError(service, errorType string) {
	ms.errorCounter.WithLabelValues(service, errorType).Inc()
}

// statusCodeToString converts HTTP status code to a string category
func (ms *MetricsService) statusCodeToString(statusCode int) string {
	switch {
	case statusCode < 200:
		return "informational"
	case statusCode < 300:
		return "success"
	case statusCode < 400:
		return "redirection"
	case statusCode < 500:
		return "client_error"
	default:
		return "server_error"
	}
}

// getMetricValue safely extracts the value from a Prometheus metric
func (ms *MetricsService) getMetricValue(metric prometheus.Metric) float64 {
	var metricDto dto.Metric
	err := metric.Write(&metricDto)
	if err != nil {
		ms.logger.Error("failed to get metric value", zap.Error(err))
		return 0
	}
	if metricDto.Counter != nil {
		return metricDto.Counter.GetValue()
	}
	return 0
}

// GetStatusMetrics returns current status metrics for a service
func (ms *MetricsService) GetStatusMetrics(service string) map[string]float64 {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	metrics := make(map[string]float64)

	// Collect metrics from Prometheus
	metrics["requests"] = ms.getRequestCount(service)
	metrics["errors"] = ms.getErrorCount(service)
	metrics["success_rate"] = ms.calculateSuccessRate(service)

	return metrics
}

// getRequestCount returns the total request count for a service
func (ms *MetricsService) getRequestCount(service string) float64 {
	var total float64

	// Get the metric for successful requests
	if metric, err := ms.requestCounter.GetMetricWithLabelValues(service, "GET", "success"); err == nil {
		total += ms.getMetricValue(metric)
	}

	// Add other HTTP methods
	methods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		if metric, err := ms.requestCounter.GetMetricWithLabelValues(service, method, "success"); err == nil {
			total += ms.getMetricValue(metric)
		}
	}

	return total
}

// getErrorCount returns the total error count for a service
func (ms *MetricsService) getErrorCount(service string) float64 {
	var total float64
	metrics, err := ms.errorCounter.GetMetricWithLabelValues(service, "server_error")
	if err == nil {
		total += ms.getMetricValue(metrics)
	}
	metrics, err = ms.errorCounter.GetMetricWithLabelValues(service, "client_error")
	if err == nil {
		total += ms.getMetricValue(metrics)
	}
	return total
}

// calculateSuccessRate calculates the success rate for a service
func (ms *MetricsService) calculateSuccessRate(service string) float64 {
	totalRequests := ms.getRequestCount(service)
	if totalRequests == 0 {
		return 100.0 // No requests means no failures
	}

	totalErrors := ms.getErrorCount(service)
	return (1 - (totalErrors / totalRequests)) * 100
}
