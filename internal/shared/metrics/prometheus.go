package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// Business metrics
	casesCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cases_created_total",
			Help: "Total number of cases created",
		},
		[]string{"type", "agency"},
	)

	casesStatusChanged = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cases_status_changed_total",
			Help: "Total number of case status changes",
		},
		[]string{"from_status", "to_status"},
	)

	documentsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "documents_created_total",
			Help: "Total number of documents created",
		},
		[]string{"type", "agency"},
	)

	documentsSigned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "documents_signed_total",
			Help: "Total number of documents signed",
		},
	)

	federationRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "federation_requests_total",
			Help: "Total number of federation requests",
		},
		[]string{"direction", "target_agency", "status"},
	)

	federationRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "federation_request_duration_seconds",
			Help:    "Federation request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
		},
		[]string{"target_agency"},
	)

	auditEntriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "audit_entries_total",
			Help: "Total number of audit entries created",
		},
	)

	authorizationDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorization_decisions_total",
			Help: "Total number of authorization decisions",
		},
		[]string{"resource_type", "action", "decision"},
	)

	// Database metrics
	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"},
	)
)

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware creates HTTP metrics middleware
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)

		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// normalizePath normalizes URL paths for metrics to avoid cardinality explosion
func normalizePath(path string) string {
	// Replace UUIDs with placeholder
	// Simple heuristic: segments that look like UUIDs
	// In production, use proper path templates
	if len(path) > 100 {
		return "/api/..."
	}
	return path
}

// --- Business metric helpers ---

// RecordCaseCreated records a case creation
func RecordCaseCreated(caseType, agencyCode string) {
	casesCreated.WithLabelValues(caseType, agencyCode).Inc()
}

// RecordCaseStatusChange records a case status change
func RecordCaseStatusChange(fromStatus, toStatus string) {
	casesStatusChanged.WithLabelValues(fromStatus, toStatus).Inc()
}

// RecordDocumentCreated records a document creation
func RecordDocumentCreated(docType, agencyCode string) {
	documentsCreated.WithLabelValues(docType, agencyCode).Inc()
}

// RecordDocumentSigned records a document signature
func RecordDocumentSigned() {
	documentsSigned.Inc()
}

// RecordFederationRequest records a federation request
func RecordFederationRequest(direction, targetAgency, status string, duration time.Duration) {
	federationRequestsTotal.WithLabelValues(direction, targetAgency, status).Inc()
	federationRequestDuration.WithLabelValues(targetAgency).Observe(duration.Seconds())
}

// RecordAuditEntry records an audit entry creation
func RecordAuditEntry() {
	auditEntriesTotal.Inc()
}

// RecordAuthorizationDecision records an authorization decision
func RecordAuthorizationDecision(resourceType, action string, allowed bool) {
	decision := "deny"
	if allowed {
		decision = "allow"
	}
	authorizationDecisions.WithLabelValues(resourceType, action, decision).Inc()
}

// RecordDBConnections records active database connections
func RecordDBConnections(count int) {
	dbConnectionsActive.Set(float64(count))
}

// RecordDBQuery records a database query duration
func RecordDBQuery(operation string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
