// pkg/metrics/collector.go

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sync"
)

// Collector handles all metrics collection for the API Gateway
type Collector struct {
	// HTTP request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec

	// Circuit breaker metrics
	circuitBreakerState *prometheus.GaugeVec

	// Rate limiting metrics
	rateLimitHits *prometheus.CounterVec

	// Cache metrics
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec

	once sync.Once
}

var (
	instance *Collector
	mu       sync.Mutex
)

// GetCollector returns the singleton instance of the metrics collector
func GetCollector() *Collector {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		instance = &Collector{}
		instance.initialize()
	}
	return instance
}

// initialize sets up all prometheus metrics
func (c *Collector) initialize() {
	c.once.Do(func() {
		// HTTP request metrics
		c.requestsTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_requests_total",
				Help: "Total number of requests processed by the API Gateway",
			},
			[]string{"method", "path", "status", "service"},
		)

		c.requestDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "api_gateway_request_duration_seconds",
				Help:    "Duration of requests processed by the API Gateway",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "service"},
		)

		c.requestsInFlight = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_requests_in_flight",
				Help: "Current number of requests being processed",
			},
			[]string{"method", "service"},
		)

		// Circuit breaker metrics
		c.circuitBreakerState = promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_gateway_circuit_breaker_state",
				Help: "Current state of circuit breakers (0=open, 1=half-open, 2=closed)",
			},
			[]string{"service"},
		)

		// Rate limiting metrics
		c.rateLimitHits = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"service", "limit_type"},
		)

		// Cache metrics
		c.cacheHits = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"},
		)

		c.cacheMisses = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_gateway_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		)
	})
}

// RecordRequest records metrics for an HTTP request
func (c *Collector) RecordRequest(method, path, status, service string, duration float64) {
	c.requestsTotal.WithLabelValues(method, path, status, service).Inc()
	c.requestDuration.WithLabelValues(method, path, service).Observe(duration)
}

// IncRequestsInFlight increments the number of requests in flight
func (c *Collector) IncRequestsInFlight(method, service string) {
	c.requestsInFlight.WithLabelValues(method, service).Inc()
}

// DecRequestsInFlight decrements the number of requests in flight
func (c *Collector) DecRequestsInFlight(method, service string) {
	c.requestsInFlight.WithLabelValues(method, service).Dec()
}

// SetCircuitBreakerState sets the current state of a circuit breaker
func (c *Collector) SetCircuitBreakerState(service string, state float64) {
	c.circuitBreakerState.WithLabelValues(service).Set(state)
}

// RecordRateLimit records a rate limit hit
func (c *Collector) RecordRateLimit(service, limitType string) {
	c.rateLimitHits.WithLabelValues(service, limitType).Inc()
}

// RecordCacheHit records a cache hit
func (c *Collector) RecordCacheHit(cacheType string) {
	c.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (c *Collector) RecordCacheMiss(cacheType string) {
	c.cacheMisses.WithLabelValues(cacheType).Inc()
}
