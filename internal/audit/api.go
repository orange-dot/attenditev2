package audit

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for the audit module
type Handler struct {
	repo               AuditRepository
	checkpointService  *CheckpointService
	devMode            bool
}

// NewHandler creates a new audit handler
func NewHandler(repo AuditRepository) *Handler {
	env := os.Getenv("ENV")
	devMode := env == "" || env == "development" || env == "dev"

	// Create checkpoint service with local witness for now
	// In production, this would be configured via TSA settings to use
	// RFC3161Witness, MultiAgencyWitness, or CompositeWitness
	witness := NewLocalWitness()
	checkpointService := NewCheckpointService(repo, witness)

	return &Handler{
		repo:              repo,
		checkpointService: checkpointService,
		devMode:           devMode,
	}
}

// Routes registers the audit routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Admin-only endpoints
	r.Get("/", h.ListEntries)
	r.Get("/verify", h.VerifyChain)
	r.Get("/resource/{resourceType}/{resourceID}", h.GetByResource)

	// Checkpoint endpoints (external witness for tamper evidence)
	r.Route("/checkpoints", func(r chi.Router) {
		r.Get("/", h.ListCheckpoints)
		r.Post("/", h.CreateCheckpoint)
		r.Get("/latest", h.GetLatestCheckpoint)
		r.Get("/{checkpointID}", h.GetCheckpoint)
		r.Get("/{checkpointID}/verify", h.VerifyCheckpoint)
	})

	// Entry by ID (must be after /verify and /checkpoints to avoid conflicts)
	r.Get("/{entryID}", h.GetEntry)

	return r
}

// ListEntries lists audit entries with filters
func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	// Only admins can view audit logs (skip in dev mode)
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	filter := ListEntriesFilter{}

	// Parse filters
	if actorID := r.URL.Query().Get("actor_id"); actorID != "" {
		id, err := types.ParseID(actorID)
		if err == nil {
			filter.ActorID = &id
		}
	}

	if actorType := r.URL.Query().Get("actor_type"); actorType != "" {
		at := ActorType(actorType)
		filter.ActorType = &at
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}

	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}

	if resourceID := r.URL.Query().Get("resource_id"); resourceID != "" {
		id, err := types.ParseID(resourceID)
		if err == nil {
			filter.ResourceID = &id
		}
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		t, err := time.Parse(time.RFC3339, startTime)
		if err == nil {
			filter.StartTime = &t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		t, err := time.Parse(time.RFC3339, endTime)
		if err == nil {
			filter.EndTime = &t
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	entries, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  entries,
		"total": total,
	})
}

// GetEntry gets an audit entry by ID
func (h *Handler) GetEntry(w http.ResponseWriter, r *http.Request) {
	// Only admins can view audit logs (skip in dev mode)
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	id, err := types.ParseID(chi.URLParam(r, "entryID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid entry ID"))
		return
	}

	entry, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// VerifyChain verifies the integrity of the audit chain
func (h *Handler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	// Only admins can verify audit chain (skip in dev mode)
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Include details if requested (for modal visualization)
	includeDetails := r.URL.Query().Get("details") == "true"

	result, err := h.repo.VerifyChain(r.Context(), limit, includeDetails)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetByResource gets audit entries for a specific resource
func (h *Handler) GetByResource(w http.ResponseWriter, r *http.Request) {
	// Only admins can view audit logs (skip in dev mode)
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	resourceType := chi.URLParam(r, "resourceType")
	resourceID, err := types.ParseID(chi.URLParam(r, "resourceID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid resource ID"))
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := h.repo.GetByResource(r.Context(), resourceType, resourceID, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  entries,
		"total": len(entries),
	})
}

// --- Checkpoint Handlers ---

// ListCheckpoints lists all checkpoints
func (h *Handler) ListCheckpoints(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	checkpoints, err := h.checkpointService.ListCheckpoints(r.Context(), limit)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  checkpoints,
		"total": len(checkpoints),
	})
}

// CreateCheckpoint creates a new checkpoint with external witness
func (h *Handler) CreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	checkpoint, err := h.checkpointService.CreateCheckpoint(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, checkpoint)
}

// GetLatestCheckpoint returns the most recent checkpoint
func (h *Handler) GetLatestCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	checkpoint, err := h.checkpointService.GetLatestCheckpoint(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	if checkpoint == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "No checkpoints created yet",
		})
		return
	}

	writeJSON(w, http.StatusOK, checkpoint)
}

// GetCheckpoint returns a specific checkpoint by ID
func (h *Handler) GetCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	id, err := types.ParseID(chi.URLParam(r, "checkpointID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid checkpoint ID"))
		return
	}

	// Use verify to get checkpoint details
	result, err := h.checkpointService.VerifyCheckpoint(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result.Checkpoint)
}

// VerifyCheckpoint verifies a checkpoint against the current chain state
func (h *Handler) VerifyCheckpoint(w http.ResponseWriter, r *http.Request) {
	if !h.devMode {
		user := auth.GetUser(r.Context())
		if user == nil || !user.IsAdmin() {
			writeError(w, errors.Forbidden("admin access required"))
			return
		}
	}

	id, err := types.ParseID(chi.URLParam(r, "checkpointID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid checkpoint ID"))
		return
	}

	result, err := h.checkpointService.VerifyCheckpoint(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
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
