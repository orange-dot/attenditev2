package agency

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for the agency module
type Handler struct {
	repo *Repository
	bus  *events.Bus
}

// NewHandler creates a new agency handler
func NewHandler(repo *Repository, bus *events.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes registers the agency routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Agency routes
	r.Route("/agencies", func(r chi.Router) {
		r.Get("/", h.ListAgencies)
		r.Post("/", h.CreateAgency)

		r.Route("/{agencyID}", func(r chi.Router) {
			r.Get("/", h.GetAgency)
			r.Put("/", h.UpdateAgency)
			r.Delete("/", h.DeleteAgency)

			// Workers under agency
			r.Route("/workers", func(r chi.Router) {
				r.Get("/", h.ListWorkers)
				r.Post("/", h.CreateWorker)
			})
		})
	})

	// Worker routes (direct access)
	r.Route("/workers", func(r chi.Router) {
		r.Route("/{workerID}", func(r chi.Router) {
			r.Get("/", h.GetWorker)
			r.Put("/", h.UpdateWorker)
			r.Delete("/", h.DeleteWorker)
		})
	})

	return r
}

// --- Agency Handlers ---

// ListAgencies lists all agencies
func (h *Handler) ListAgencies(w http.ResponseWriter, r *http.Request) {
	filter := ListAgenciesFilter{
		Search: r.URL.Query().Get("search"),
	}

	if t := r.URL.Query().Get("type"); t != "" {
		agencyType := AgencyType(t)
		filter.Type = &agencyType
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := AgencyStatus(s)
		filter.Status = &status
	}

	agencies, total, err := h.repo.ListAgencies(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  agencies,
		"total": total,
	})
}

// GetAgency gets an agency by ID
func (h *Handler) GetAgency(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	agency, err := h.repo.GetAgency(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, agency)
}

// CreateAgency creates a new agency
func (h *Handler) CreateAgency(w http.ResponseWriter, r *http.Request) {
	var req CreateAgencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	// Validate request
	if req.Code == "" || req.Name == "" {
		writeError(w, errors.Validation("validation failed", map[string]string{
			"code": "code is required",
			"name": "name is required",
		}))
		return
	}

	agency := &Agency{
		ID:       types.NewID(),
		Code:     req.Code,
		Name:     req.Name,
		Type:     req.Type,
		ParentID: req.ParentID,
		Status:   AgencyStatusActive,
		Address:  req.Address,
		Contact:  req.Contact,
	}

	if err := h.repo.CreateAgency(r.Context(), agency); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		user := auth.GetUser(r.Context())
		actorID := types.ID("")
		if user != nil {
			actorID = user.ID
		}

		event := events.NewEvent("agency.created", "agency", map[string]any{
			"agency_id":   agency.ID,
			"agency_code": agency.Code,
			"agency_name": agency.Name,
		}).WithActor(actorID, "worker", types.ID(""))

		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusCreated, agency)
}

// UpdateAgency updates an agency
func (h *Handler) UpdateAgency(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	agency, err := h.repo.GetAgency(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req UpdateAgencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	// Apply updates
	if req.Name != nil {
		agency.Name = *req.Name
	}
	if req.ParentID != nil {
		agency.ParentID = req.ParentID
	}
	if req.Status != nil {
		agency.Status = *req.Status
	}
	if req.Address != nil {
		agency.Address = *req.Address
	}
	if req.Contact != nil {
		agency.Contact = *req.Contact
	}

	if err := h.repo.UpdateAgency(r.Context(), agency); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, agency)
}

// DeleteAgency deletes an agency
func (h *Handler) DeleteAgency(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	if err := h.repo.DeleteAgency(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Worker Handlers ---

// ListWorkers lists workers for an agency
func (h *Handler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	filter := ListWorkersFilter{
		Search: r.URL.Query().Get("search"),
	}

	// If under /agencies/{agencyID}/workers, filter by agency
	if agencyID := chi.URLParam(r, "agencyID"); agencyID != "" {
		id, err := types.ParseID(agencyID)
		if err != nil {
			writeError(w, errors.BadRequest("invalid agency ID"))
			return
		}
		filter.AgencyID = &id
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := WorkerStatus(s)
		filter.Status = &status
	}

	workers, total, err := h.repo.ListWorkers(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  workers,
		"total": total,
	})
}

// GetWorker gets a worker by ID
func (h *Handler) GetWorker(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "workerID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid worker ID"))
		return
	}

	worker, err := h.repo.GetWorker(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, worker)
}

// CreateWorker creates a new worker
func (h *Handler) CreateWorker(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	// Get agency ID from URL or request
	agencyID := req.AgencyID
	if urlAgencyID := chi.URLParam(r, "agencyID"); urlAgencyID != "" {
		id, err := types.ParseID(urlAgencyID)
		if err != nil {
			writeError(w, errors.BadRequest("invalid agency ID"))
			return
		}
		agencyID = id
	}

	// Validate request
	if agencyID.IsZero() || req.EmployeeID == "" || req.Email == "" {
		writeError(w, errors.Validation("validation failed", map[string]string{
			"agency_id":   "agency_id is required",
			"employee_id": "employee_id is required",
			"email":       "email is required",
		}))
		return
	}

	user := auth.GetUser(r.Context())
	grantedBy := types.ID("")
	if user != nil {
		grantedBy = user.ID
	}

	worker := &Worker{
		ID:         types.NewID(),
		AgencyID:   agencyID,
		EmployeeID: req.EmployeeID,
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Email:      req.Email,
		Position:   req.Position,
		Department: req.Department,
		Status:     WorkerStatusActive,
	}

	// Add roles
	for _, role := range req.Roles {
		worker.Roles = append(worker.Roles, WorkerRole{
			ID:        types.NewID(),
			WorkerID:  worker.ID,
			Role:      role,
			Scope:     "all",
			GrantedAt: time.Now(),
			GrantedBy: grantedBy,
		})
	}

	if err := h.repo.CreateWorker(r.Context(), worker); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		event := events.NewEvent("agency.worker.added", "agency", map[string]any{
			"worker_id":   worker.ID,
			"agency_id":   worker.AgencyID,
			"employee_id": worker.EmployeeID,
			"roles":       req.Roles,
		}).WithActor(grantedBy, "worker", agencyID)

		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusCreated, worker)
}

// UpdateWorker updates a worker
func (h *Handler) UpdateWorker(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "workerID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid worker ID"))
		return
	}

	worker, err := h.repo.GetWorker(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req UpdateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	// Apply updates
	if req.FirstName != nil {
		worker.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		worker.LastName = *req.LastName
	}
	if req.Email != nil {
		worker.Email = *req.Email
	}
	if req.Position != nil {
		worker.Position = *req.Position
	}
	if req.Department != nil {
		worker.Department = *req.Department
	}
	if req.Status != nil {
		worker.Status = *req.Status
	}

	if err := h.repo.UpdateWorker(r.Context(), worker); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, worker)
}

// DeleteWorker deletes a worker
func (h *Handler) DeleteWorker(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "workerID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid worker ID"))
		return
	}

	if err := h.repo.DeleteWorker(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
