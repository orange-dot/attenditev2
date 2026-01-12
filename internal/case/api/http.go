package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/case/domain"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for the case module
type Handler struct {
	repo domain.Repository
	bus  events.EventBus
}

// NewHandler creates a new case handler
func NewHandler(repo domain.Repository, bus events.EventBus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes registers the case routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListCases)
	r.Post("/", h.CreateCase)

	r.Route("/{caseID}", func(r chi.Router) {
		r.Get("/", h.GetCase)
		r.Put("/", h.UpdateCase)
		r.Delete("/", h.DeleteCase)

		// Status transitions
		r.Post("/open", h.OpenCase)
		r.Post("/start", h.StartCase)
		r.Post("/close", h.CloseCase)
		r.Post("/escalate", h.EscalateCase)

		// Sharing
		r.Post("/share", h.ShareCase)
		r.Post("/transfer", h.TransferCase)

		// Participants
		r.Route("/participants", func(r chi.Router) {
			r.Get("/", h.ListParticipants)
			r.Post("/", h.AddParticipant)
			r.Delete("/{participantID}", h.RemoveParticipant)
		})

		// Assignments
		r.Route("/assignments", func(r chi.Router) {
			r.Get("/", h.ListAssignments)
			r.Post("/", h.AddAssignment)
		})

		// Events/Timeline
		r.Get("/events", h.GetEvents)
	})

	return r
}

// --- Request/Response types ---

type CreateCaseRequest struct {
	Type        domain.CaseType `json:"type"`
	Priority    domain.Priority `json:"priority"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
}

type UpdateCaseRequest struct {
	Title       *string          `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
	Priority    *domain.Priority `json:"priority,omitempty"`
}

type CloseCaseRequest struct {
	Resolution string `json:"resolution"`
}

type EscalateCaseRequest struct {
	Level       int      `json:"level"`
	Reason      string   `json:"reason"`
	EscalateTo  types.ID `json:"escalate_to"`
}

type ShareCaseRequest struct {
	AgencyID    types.ID            `json:"agency_id"`
	AccessLevel domain.AccessLevel  `json:"access_level"`
}

type TransferCaseRequest struct {
	ToAgencyID      types.ID `json:"to_agency_id"`
	NewLeadWorkerID types.ID `json:"new_lead_worker_id"`
	Reason          string   `json:"reason"`
}

type AddParticipantRequest struct {
	CitizenID    *types.ID             `json:"citizen_id,omitempty"`
	Role         domain.ParticipantRole `json:"role"`
	Name         string                 `json:"name"`
	ContactEmail string                 `json:"contact_email,omitempty"`
	ContactPhone string                 `json:"contact_phone,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
}

type AddAssignmentRequest struct {
	WorkerID types.ID              `json:"worker_id"`
	AgencyID types.ID              `json:"agency_id"`
	Role     domain.AssignmentRole `json:"role"`
}

// --- Handlers ---

func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())

	filter := domain.ListFilter{
		Search: r.URL.Query().Get("search"),
	}

	if t := r.URL.Query().Get("type"); t != "" {
		caseType := domain.CaseType(t)
		filter.Type = &caseType
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := domain.CaseStatus(s)
		filter.Status = &status
	}

	if p := r.URL.Query().Get("priority"); p != "" {
		priority := domain.Priority(p)
		filter.Priority = &priority
	}

	var cases []domain.Case
	var total int
	var err error

	// Filter based on user's agency
	if user != nil && !user.AgencyID.IsZero() {
		// Get cases owned by or shared with user's agency
		ownedCases, ownedTotal, err1 := h.repo.FindByAgency(r.Context(), user.AgencyID, filter)
		if err1 != nil {
			writeError(w, err1)
			return
		}

		sharedCases, _, err2 := h.repo.FindSharedWith(r.Context(), user.AgencyID, filter)
		if err2 != nil {
			writeError(w, err2)
			return
		}

		// Merge and dedupe
		caseMap := make(map[types.ID]domain.Case)
		for _, c := range ownedCases {
			caseMap[c.ID] = c
		}
		for _, c := range sharedCases {
			if _, exists := caseMap[c.ID]; !exists {
				caseMap[c.ID] = c
			}
		}

		cases = make([]domain.Case, 0, len(caseMap))
		for _, c := range caseMap {
			cases = append(cases, c)
		}
		total = ownedTotal
	} else {
		cases, total, err = h.repo.List(r.Context(), filter)
		if err != nil {
			writeError(w, err)
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  cases,
		"total": total,
	})
}

func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return
	}

	c, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	// Check access
	user := auth.GetUser(r.Context())
	if user != nil && !user.AgencyID.IsZero() {
		if !c.CanAccess(user.AgencyID, domain.AccessLevelRead) {
			writeError(w, errors.Forbidden("no access to this case"))
			return
		}
	}

	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	var req CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	user := auth.GetUser(r.Context())
	var agencyID, workerID types.ID
	if user != nil {
		agencyID = user.AgencyID
		workerID = user.ID
	} else {
		// For development without auth
		agencyID = types.NewID()
		workerID = types.NewID()
	}

	c, err := domain.NewCase(
		req.Type,
		req.Priority,
		req.Title,
		req.Description,
		agencyID,
		workerID,
	)
	if err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Save(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	// Publish domain events
	h.publishEvents(r.Context(), c)

	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return
	}

	c, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if req.Title != nil {
		c.Title = *req.Title
	}
	if req.Description != nil {
		c.Description = *req.Description
	}
	if req.Priority != nil {
		c.Priority = *req.Priority
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) DeleteCase(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) OpenCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	if err := c.Open(user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) StartCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	if err := c.StartProgress(user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) CloseCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req CloseCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := c.Close(user.ID, user.AgencyID, req.Resolution); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) EscalateCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req EscalateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := c.Escalate(req.Level, req.Reason, req.EscalateTo, user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ShareCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req ShareCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := c.Share(req.AgencyID, req.AccessLevel, user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) TransferCase(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req TransferCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := c.Transfer(req.ToAgencyID, req.NewLeadWorkerID, user.ID, user.AgencyID, req.Reason); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), c); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	c, _ := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  c.Participants,
		"total": len(c.Participants),
	})
}

func (h *Handler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req AddParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	participant := domain.Participant{
		CitizenID:    req.CitizenID,
		Role:         req.Role,
		Name:         req.Name,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Notes:        req.Notes,
	}

	if err := c.AddParticipant(participant, user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	// Save the new participant
	newParticipant := &c.Participants[len(c.Participants)-1]
	if err := h.repo.AddParticipant(r.Context(), c.ID, newParticipant); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusCreated, newParticipant)
}

func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	caseID, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return
	}

	participantID, err := types.ParseID(chi.URLParam(r, "participantID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid participant ID"))
		return
	}

	if err := h.repo.RemoveParticipant(r.Context(), caseID, participantID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	c, _ := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  c.Assignments,
		"total": len(c.Assignments),
	})
}

func (h *Handler) AddAssignment(w http.ResponseWriter, r *http.Request) {
	c, user := h.getCaseAndUser(w, r)
	if c == nil {
		return
	}

	var req AddAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := c.Assign(req.WorkerID, req.AgencyID, req.Role, user.ID, user.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	// Save the new assignment
	newAssignment := &c.Assignments[len(c.Assignments)-1]
	if err := h.repo.AddAssignment(r.Context(), c.ID, newAssignment); err != nil {
		writeError(w, err)
		return
	}

	h.publishEvents(r.Context(), c)
	writeJSON(w, http.StatusCreated, newAssignment)
}

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return
	}

	events, err := h.repo.GetEvents(r.Context(), id, 50, 0)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  events,
		"total": len(events),
	})
}

// --- Helpers ---

func (h *Handler) getCaseAndUser(w http.ResponseWriter, r *http.Request) (*domain.Case, *auth.User) {
	id, err := types.ParseID(chi.URLParam(r, "caseID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid case ID"))
		return nil, nil
	}

	c, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return nil, nil
	}

	user := auth.GetUser(r.Context())
	if user == nil {
		// For development without auth
		user = &auth.User{
			ID:       types.NewID(),
			AgencyID: c.OwningAgencyID,
		}
	}

	return c, user
}

func (h *Handler) publishEvents(ctx context.Context, c *domain.Case) {
	if h.bus == nil {
		return
	}

	for _, e := range c.GetDomainEvents() {
		event := events.NewEvent("case."+e.Type, "case", map[string]any{
			"case_id":     c.ID,
			"case_number": c.CaseNumber,
			"event":       e.CaseEvent,
		}).WithActor(e.CaseEvent.ActorID, "worker", e.CaseEvent.ActorAgencyID)

		h.bus.Publish(ctx, event)
	}
}

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
