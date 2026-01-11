package trust

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for the Trust Authority
type Handler struct {
	authority *Authority
}

// NewHandler creates a new trust handler
func NewHandler(authority *Authority) *Handler {
	return &Handler{authority: authority}
}

// Routes registers the trust authority routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Agency registry
	r.Get("/agencies", h.ListAgencies)
	r.Post("/agencies", h.RegisterAgency)
	r.Get("/agencies/{agencyID}", h.GetAgency)
	r.Post("/agencies/{agencyID}/suspend", h.SuspendAgency)
	r.Post("/agencies/{agencyID}/revoke", h.RevokeAgency)

	// Service catalog
	r.Get("/agencies/{agencyID}/services", h.GetServices)
	r.Post("/agencies/{agencyID}/services", h.RegisterService)
	r.Get("/services/{serviceType}", h.FindService)

	// Certificates
	r.Get("/ca/certificate", h.GetRootCertificate)
	r.Post("/verify", h.VerifyCertificate)

	return r
}

// --- Request types ---

type RegisterAgencyRequest struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	GatewayURL string `json:"gateway_url"`
}

type RegisterServiceRequest struct {
	ServiceType string `json:"service_type"`
	Path        string `json:"path"`
	Version     string `json:"version"`
}

type SuspendRequest struct {
	Reason string `json:"reason"`
}

type VerifyCertificateRequest struct {
	Certificate string `json:"certificate"` // PEM encoded
}

// --- Handlers ---

func (h *Handler) ListAgencies(w http.ResponseWriter, r *http.Request) {
	agencies, err := h.authority.ListAgencies(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  agencies,
		"total": len(agencies),
	})
}

func (h *Handler) RegisterAgency(w http.ResponseWriter, r *http.Request) {
	// Only admins can register agencies
	user := auth.GetUser(r.Context())
	if user != nil && !user.IsAdmin() {
		writeError(w, errors.Forbidden("admin access required"))
		return
	}

	var req RegisterAgencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if req.Name == "" || req.Code == "" {
		writeError(w, errors.BadRequest("name and code are required"))
		return
	}

	agency, err := h.authority.RegisterAgency(r.Context(), req.Name, req.Code, req.GatewayURL)
	if err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, agency)
}

func (h *Handler) GetAgency(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		// Try by code
		agency, err := h.authority.GetAgencyByCode(r.Context(), chi.URLParam(r, "agencyID"))
		if err != nil {
			writeError(w, errors.NotFound("agency", chi.URLParam(r, "agencyID")))
			return
		}
		writeJSON(w, http.StatusOK, agency)
		return
	}

	agency, err := h.authority.GetAgency(r.Context(), id)
	if err != nil {
		writeError(w, errors.NotFound("agency", id.String()))
		return
	}

	writeJSON(w, http.StatusOK, agency)
}

func (h *Handler) SuspendAgency(w http.ResponseWriter, r *http.Request) {
	// Only admins can suspend agencies
	user := auth.GetUser(r.Context())
	if user != nil && !user.IsAdmin() {
		writeError(w, errors.Forbidden("admin access required"))
		return
	}

	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	var req SuspendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := h.authority.SuspendAgency(r.Context(), id, req.Reason); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) RevokeAgency(w http.ResponseWriter, r *http.Request) {
	// Only admins can revoke agencies
	user := auth.GetUser(r.Context())
	if user != nil && !user.IsAdmin() {
		writeError(w, errors.Forbidden("admin access required"))
		return
	}

	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	var req SuspendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := h.authority.RevokeAgency(r.Context(), id, req.Reason); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) GetServices(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	services, err := h.authority.GetServices(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  services,
		"total": len(services),
	})
}

func (h *Handler) RegisterService(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "agencyID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid agency ID"))
		return
	}

	var req RegisterServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	service, err := h.authority.RegisterService(r.Context(), id, req.ServiceType, req.Path, req.Version)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, service)
}

func (h *Handler) FindService(w http.ResponseWriter, r *http.Request) {
	serviceType := chi.URLParam(r, "serviceType")

	services, err := h.authority.FindService(r.Context(), serviceType)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  services,
		"total": len(services),
	})
}

func (h *Handler) GetRootCertificate(w http.ResponseWriter, r *http.Request) {
	cert := h.authority.GetRootCertificatePEM()
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=root-ca.pem")
	w.Write(cert)
}

func (h *Handler) VerifyCertificate(w http.ResponseWriter, r *http.Request) {
	var req VerifyCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	err := h.authority.VerifyCertificate([]byte(req.Certificate))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"valid": true,
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
