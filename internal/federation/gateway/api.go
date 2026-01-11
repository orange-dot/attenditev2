package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/events"
)

// Handler provides HTTP handlers for the gateway
type Handler struct {
	gateway   *Gateway
	bus       *events.Bus
	localMux  http.Handler // Router for local services
}

// NewHandler creates a new gateway handler
func NewHandler(gateway *Gateway, bus *events.Bus, localMux http.Handler) *Handler {
	return &Handler{
		gateway:  gateway,
		bus:      bus,
		localMux: localMux,
	}
}

// Routes registers the gateway routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Receive incoming cross-agency requests
	r.Post("/receive", h.ReceiveRequest)

	// Send outgoing requests (internal API)
	r.Post("/send", h.SendRequest)

	// Health check for federation
	r.Get("/health", h.HealthCheck)

	return r
}

// --- Request types ---

type SendRequestPayload struct {
	TargetAgency string            `json:"target_agency"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         json.RawMessage   `json:"body,omitempty"`
}

// --- Handlers ---

// ReceiveRequest handles incoming cross-agency requests
func (h *Handler) ReceiveRequest(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, errors.BadRequest("failed to read request body"))
		return
	}

	// Parse signed request
	var signedReq SignedRequest
	if err := json.Unmarshal(body, &signedReq); err != nil {
		writeError(w, errors.BadRequest("invalid request format"))
		return
	}

	// Verify signature
	if err := h.gateway.VerifyRequest(r.Context(), &signedReq); err != nil {
		writeError(w, errors.Unauthorized("signature verification failed: "+err.Error()))
		return
	}

	// Publish federation event
	if h.bus != nil {
		event := events.NewEvent("federation.request.received", "gateway", map[string]any{
			"request_id":    signedReq.ID,
			"source_agency": signedReq.SourceAgency,
			"method":        signedReq.Method,
			"path":          signedReq.Path,
		})
		h.bus.Publish(r.Context(), event)
	}

	// Forward to local service
	var respBody []byte
	var statusCode int

	if h.localMux != nil {
		// Create internal request
		internalReq, err := http.NewRequestWithContext(
			r.Context(),
			signedReq.Method,
			signedReq.Path,
			nil,
		)
		if err != nil {
			statusCode = http.StatusInternalServerError
			respBody = []byte(`{"error":"failed to create internal request"}`)
		} else {
			// Add federation headers
			internalReq.Header.Set("X-Federation-Source", signedReq.SourceAgency)
			internalReq.Header.Set("X-Federation-Request-ID", signedReq.ID)

			// Capture response
			rw := &responseWriter{
				header: make(http.Header),
				body:   &jsonBuffer{},
			}
			h.localMux.ServeHTTP(rw, internalReq)

			statusCode = rw.statusCode
			respBody = rw.body.Bytes()
		}
	} else {
		statusCode = http.StatusNotImplemented
		respBody = []byte(`{"error":"no local handler configured"}`)
	}

	// Create signed response
	signedResp, err := h.gateway.CreateResponse(signedReq.ID, statusCode, respBody)
	if err != nil {
		writeError(w, errors.Internal(fmt.Errorf("failed to sign response: %w", err)))
		return
	}

	writeJSON(w, http.StatusOK, signedResp)
}

// SendRequest handles outgoing cross-agency requests (internal API)
func (h *Handler) SendRequest(w http.ResponseWriter, r *http.Request) {
	var req SendRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if req.TargetAgency == "" || req.Method == "" || req.Path == "" {
		writeError(w, errors.BadRequest("target_agency, method, and path are required"))
		return
	}

	// Publish federation event
	if h.bus != nil {
		event := events.NewEvent("federation.request.sent", "gateway", map[string]any{
			"target_agency": req.TargetAgency,
			"method":        req.Method,
			"path":          req.Path,
		})
		h.bus.Publish(r.Context(), event)
	}

	// Send request through gateway
	resp, err := h.gateway.SendRequest(r.Context(), req.TargetAgency, req.Method, req.Path, req.Body)
	if err != nil {
		writeError(w, errors.Internal(fmt.Errorf("federation request failed: %w", err)))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status_code": resp.StatusCode,
		"body":        json.RawMessage(resp.Body),
		"request_id":  resp.RequestID,
		"timestamp":   resp.Timestamp,
	})
}

// HealthCheck returns federation gateway health status
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"gateway": "active",
	})
}

// --- Helper types ---

type responseWriter struct {
	header     http.Header
	body       *jsonBuffer
	statusCode int
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
}

type jsonBuffer struct {
	data []byte
}

func (b *jsonBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *jsonBuffer) Bytes() []byte {
	return b.data
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	if appErr, ok := err.(*errors.AppError); ok {
		w.WriteHeader(appErr.HTTPStatus)
		json.NewEncoder(w).Encode(map[string]any{
			"error":   appErr.Message,
			"code":    appErr.Code,
			"details": appErr.Details,
		})
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
}
