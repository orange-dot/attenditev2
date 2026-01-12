package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for simulation
type Handler struct {
	bus events.EventBus
}

// NewHandler creates a new simulation handler
func NewHandler(bus events.EventBus) *Handler {
	return &Handler{bus: bus}
}

// Routes registers the simulation routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/start", h.StartSimulation)
	r.Post("/step", h.ExecuteStep)
	r.Post("/complete", h.CompleteSimulation)
	r.Get("/institutions", h.ListInstitutions)

	return r
}

// StartSimulation starts a new simulation session
func (h *Handler) StartSimulation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UseCaseID    string `json:"use_case_id"`
		UseCaseTitle string `json:"use_case_title"`
		TotalSteps   int    `json:"total_steps"`
		CitizenJMBG  string `json:"citizen_jmbg,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	sessionID := types.NewID()

	// Publish simulation started event
	systemActorID := types.NewDeterministicID("system", "simulation-api")
	event := events.NewEvent(EventSimulationStarted, "simulation-api", map[string]any{
		"session_id":     sessionID.String(),
		"use_case_id":    req.UseCaseID,
		"use_case_title": req.UseCaseTitle,
		"total_steps":    req.TotalSteps,
		"citizen_jmbg":   maskJMBG(req.CitizenJMBG),
	}).WithActor(systemActorID, "system", types.ID(""))

	if err := h.bus.Publish(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to publish event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"session_id": sessionID.String(),
		"message":    fmt.Sprintf("Simulacija '%s' pokrenuta", req.UseCaseTitle),
		"timestamp":  time.Now().UTC(),
	})
}

// ExecuteStep executes a simulation step
func (h *Handler) ExecuteStep(w http.ResponseWriter, r *http.Request) {
	var req SimulationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get institution names
	fromInst := getInstitutionName(req.Step.FromInstitution)
	toInst := getInstitutionName(req.Step.ToInstitution)

	// Determine event type based on whether it's a request or response
	eventType := EventDataRequest
	if req.Step.IsResponse {
		eventType = EventDataResponse
	}

	// Create event data
	eventData := map[string]any{
		"session_id":       req.SessionID,
		"use_case_id":      req.UseCaseID,
		"use_case_title":   req.UseCaseTitle,
		"step_id":          req.Step.StepID,
		"from_institution": req.Step.FromInstitution,
		"from_name":        fromInst,
		"to_institution":   req.Step.ToInstitution,
		"to_name":          toInst,
		"action":           req.Step.Action,
		"description":      req.Step.Description,
		"data_exchanged":   req.Step.DataExchanged,
		"is_response":      req.Step.IsResponse,
	}

	// Add citizen JMBG if present (masked)
	if req.CitizenJMBG != "" {
		eventData["citizen_jmbg"] = maskJMBG(req.CitizenJMBG)
	}

	// Determine actor - use a deterministic UUID based on institution ID
	actorID := types.NewDeterministicID("institution", req.Step.FromInstitution)
	actorType := "system"
	if req.Step.FromInstitution == "citizen" {
		actorType = "citizen"
	}

	// Publish event
	agencyID := types.NewDeterministicID("institution", req.Step.FromInstitution)
	event := events.NewEvent(eventType, "simulation-api", eventData).
		WithActor(actorID, actorType, agencyID).
		WithCorrelation(req.SessionID)

	if err := h.bus.Publish(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to publish event")
		return
	}

	// Generate response message
	var message string
	if req.Step.IsResponse {
		message = fmt.Sprintf("%s → %s: Odgovor poslat", fromInst, toInst)
	} else {
		message = fmt.Sprintf("%s → %s: Zahtev poslat", fromInst, toInst)
	}

	writeJSON(w, http.StatusOK, SimulationResponse{
		Success:      true,
		AuditEntryID: event.ID,
		Timestamp:    time.Now().UTC(),
		Message:      message,
	})
}

// CompleteSimulation marks a simulation as completed
func (h *Handler) CompleteSimulation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID    string `json:"session_id"`
		UseCaseID    string `json:"use_case_id"`
		UseCaseTitle string `json:"use_case_title"`
		TotalSteps   int    `json:"total_steps"`
		Success      bool   `json:"success"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Publish simulation completed event
	systemActorID := types.NewDeterministicID("system", "simulation-api")
	event := events.NewEvent(EventSimulationCompleted, "simulation-api", map[string]any{
		"session_id":     req.SessionID,
		"use_case_id":    req.UseCaseID,
		"use_case_title": req.UseCaseTitle,
		"total_steps":    req.TotalSteps,
		"success":        req.Success,
		"completed_at":   time.Now().UTC(),
	}).WithActor(systemActorID, "system", types.ID("")).
		WithCorrelation(req.SessionID)

	if err := h.bus.Publish(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to publish event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"message":   fmt.Sprintf("Simulacija '%s' završena uspešno", req.UseCaseTitle),
		"timestamp": time.Now().UTC(),
	})
}

// ListInstitutions returns the list of institutions
func (h *Handler) ListInstitutions(w http.ResponseWriter, r *http.Request) {
	institutions := make([]Institution, 0, len(Institutions))
	for _, inst := range Institutions {
		if inst.ID != "citizen" {
			institutions = append(institutions, inst)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  institutions,
		"total": len(institutions),
	})
}

// --- Helpers ---

func getInstitutionName(id string) string {
	if inst, ok := Institutions[id]; ok {
		return inst.Name
	}
	return id
}

func maskJMBG(jmbg string) string {
	if len(jmbg) < 13 {
		return "***"
	}
	return jmbg[:4] + "*****" + jmbg[9:]
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// PublishEvent allows external packages to publish simulation events
func (h *Handler) PublishEvent(ctx context.Context, eventType string, data map[string]any, actorID, actorType string) error {
	event := events.NewEvent(eventType, "simulation-api", data).
		WithActor(types.ID(actorID), actorType, types.ID(""))
	return h.bus.Publish(ctx, event)
}
