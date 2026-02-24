package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sakura_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sakura_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sakura_http_requests_in_flight",
			Help: "Number of HTTP requests currently being served",
		},
	)

	AgentsOnline = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sakura_agents_online",
			Help: "Number of connected agents",
		},
	)

	WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sakura_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	ServerCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sakura_servers_total",
			Help: "Total number of managed servers",
		},
	)

	IPMICommandsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sakura_ipmi_commands_total",
			Help: "Total IPMI commands dispatched",
		},
		[]string{"action", "status"},
	)

	KVMSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sakura_kvm_sessions_active",
			Help: "Number of active KVM sessions",
		},
	)
)

// PrometheusMetrics records HTTP request metrics.
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpRequestsInFlight.Inc()
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
		httpRequestsInFlight.Dec()
	}
}
