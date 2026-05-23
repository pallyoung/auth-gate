package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "auth_gate",
			Subsystem: "",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests handled by auth-gate",
		},
		[]string{"route", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "auth_gate",
			Subsystem: "",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"route"},
	)

	BackendUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "auth_gate",
			Subsystem: "",
			Name:      "backend_up",
			Help:      "Whether the backend is reachable (1=up, 0=down)",
		},
		[]string{"route", "backend"},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "auth_gate",
			Subsystem: "",
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state per backend: 0=closed, 1=open, 2=half-open",
		},
		[]string{"backend"},
	)

	RateLimitExceededTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "auth_gate",
			Subsystem: "",
			Name:      "rate_limit_exceeded_total",
			Help:      "Total number of requests rejected due to rate limiting",
		},
		[]string{"route"},
	)
)

// RecordRequest increments the request counter and records request duration.
func RecordRequest(routeID, method, status string, durationMs float64) {
	RequestsTotal.WithLabelValues(routeID, method, status).Inc()
	RequestDuration.WithLabelValues(routeID).Observe(durationMs / 1000.0)
}

// RecordBackendHealth sets the backend up/down gauge.
func RecordBackendHealth(routeID, backend string, isUp bool) {
	val := 0.0
	if isUp {
		val = 1.0
	}
	BackendUp.WithLabelValues(routeID, backend).Set(val)
}

// RecordCircuitState updates the circuit breaker state gauge.
// state: 0=closed, 1=open, 2=half-open
func RecordCircuitState(backend string, state int) {
	CircuitBreakerState.WithLabelValues(backend).Set(float64(state))
}

// RecordRateLimitExceeded increments the rate limit exceeded counter.
func RecordRateLimitExceeded(routeID string) {
	RateLimitExceededTotal.WithLabelValues(routeID).Inc()
}