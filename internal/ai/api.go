package ai

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/errors"
)

// Handler provides HTTP handlers for the AI module
type Handler struct {
	client *Client
}

// NewHandler creates a new AI handler
func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

// Routes registers the AI routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/analyze", h.Analyze)
	r.Get("/examples", h.GetExamples)
	r.Get("/health", h.HealthCheck)

	return r
}

// Analyze handles document analysis requests
func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body: "+err.Error()))
		return
	}

	if req.DocumentText == "" {
		writeError(w, errors.BadRequest("document_text is required"))
		return
	}

	result, err := h.client.Analyze(r.Context(), req)
	if err != nil {
		writeError(w, errors.Wrap(err, "AI analysis failed"))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetExamples returns test examples
func (h *Handler) GetExamples(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.GetExamples(r.Context())
	if err != nil {
		writeError(w, errors.Wrap(err, "failed to get examples"))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// HealthCheck checks AI service health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	err := h.client.Health(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
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
