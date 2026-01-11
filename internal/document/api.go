package document

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Handler provides HTTP handlers for the document module
type Handler struct {
	repo *Repository
	bus  *events.Bus
}

// NewHandler creates a new document handler
func NewHandler(repo *Repository, bus *events.Bus) *Handler {
	return &Handler{repo: repo, bus: bus}
}

// Routes registers the document routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListDocuments)
	r.Post("/", h.CreateDocument)

	r.Route("/{documentID}", func(r chi.Router) {
		r.Get("/", h.GetDocument)
		r.Put("/", h.UpdateDocument)
		r.Delete("/", h.DeleteDocument)

		// Actions
		r.Post("/share", h.ShareDocument)
		r.Post("/archive", h.ArchiveDocument)
		r.Post("/void", h.VoidDocument)

		// Versions
		r.Get("/versions", h.ListVersions)
		// POST /versions would handle file upload - simplified here

		// Signatures
		r.Get("/signatures", h.ListSignatures)
		r.Post("/signatures", h.RequestSignature)
		r.Post("/signatures/{signatureID}/sign", h.SignDocument)
		r.Post("/signatures/{signatureID}/reject", h.RejectSignature)
	})

	return r
}

// ListDocuments lists documents
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	filter := ListDocumentsFilter{
		Search: r.URL.Query().Get("search"),
	}

	if t := r.URL.Query().Get("type"); t != "" {
		docType := DocumentType(t)
		filter.Type = &docType
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := DocumentStatus(s)
		filter.Status = &status
	}

	if c := r.URL.Query().Get("case_id"); c != "" {
		caseID, err := types.ParseID(c)
		if err == nil {
			filter.CaseID = &caseID
		}
	}

	docs, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  docs,
		"total": total,
	})
}

// GetDocument gets a document by ID
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	// Check access
	user := auth.GetUser(r.Context())
	if user != nil && !user.AgencyID.IsZero() {
		if !doc.CanAccess(user.AgencyID) {
			writeError(w, errors.Forbidden("no access to this document"))
			return
		}
	}

	writeJSON(w, http.StatusOK, doc)
}

// CreateDocument creates a new document
func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	user := auth.GetUser(r.Context())
	var agencyID, userID types.ID
	if user != nil {
		agencyID = user.AgencyID
		userID = user.ID
	} else {
		agencyID = types.NewID()
		userID = types.NewID()
	}

	doc, err := NewDocument(
		req.Type,
		req.Title,
		req.Description,
		agencyID,
		userID,
		req.CaseID,
	)
	if err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Save(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		event := events.NewEvent("document.created", "document", map[string]any{
			"document_id":     doc.ID,
			"document_number": doc.DocumentNumber,
			"title":           doc.Title,
		}).WithActor(userID, "worker", agencyID)

		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusCreated, doc)
}

// UpdateDocument updates a document
func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if req.Title != nil {
		doc.Title = *req.Title
	}
	if req.Description != nil {
		doc.Description = *req.Description
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		user := auth.GetUser(r.Context())
		actorID := types.NewID()
		if user != nil {
			actorID = user.ID
		}
		event := events.NewEvent("document.updated", "document", map[string]any{
			"document_id": doc.ID,
			"title":       doc.Title,
		}).WithActor(actorID, "worker", doc.OwnerAgencyID)
		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
}

// DeleteDocument deletes a document
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ShareDocument shares a document with an agency
func (h *Handler) ShareDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req ShareDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := doc.Share(req.AgencyID); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		user := auth.GetUser(r.Context())
		actorID := types.NewID()
		if user != nil {
			actorID = user.ID
		}
		event := events.NewEvent("document.shared", "document", map[string]any{
			"document_id":     doc.ID,
			"shared_with":     req.AgencyID,
		}).WithActor(actorID, "worker", doc.OwnerAgencyID)
		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
}

// ArchiveDocument archives a document
func (h *Handler) ArchiveDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := doc.Archive(); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		user := auth.GetUser(r.Context())
		actorID := types.NewID()
		if user != nil {
			actorID = user.ID
		}
		event := events.NewEvent("document.archived", "document", map[string]any{
			"document_id": doc.ID,
		}).WithActor(actorID, "worker", doc.OwnerAgencyID)
		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
}

// VoidDocument voids a document
func (h *Handler) VoidDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := doc.Void(); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		user := auth.GetUser(r.Context())
		actorID := types.NewID()
		if user != nil {
			actorID = user.ID
		}
		event := events.NewEvent("document.voided", "document", map[string]any{
			"document_id": doc.ID,
		}).WithActor(actorID, "worker", doc.OwnerAgencyID)
		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
}

// ListVersions lists document versions
func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  doc.Versions,
		"total": len(doc.Versions),
	})
}

// ListSignatures lists document signatures
func (h *Handler) ListSignatures(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  doc.Signatures,
		"total": len(doc.Signatures),
	})
}

// RequestSignature requests a signature on a document
func (h *Handler) RequestSignature(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	var req RequestSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	user := auth.GetUser(r.Context())
	requestedBy := types.NewID()
	if user != nil {
		requestedBy = user.ID
	}

	sig, err := doc.RequestSignature(
		req.SignerID,
		req.SignerAgencyID,
		requestedBy,
		req.Type,
		nil,
		req.Reason,
		req.Location,
	)
	if err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	// Save signature
	if err := h.repo.AddSignature(r.Context(), sig); err != nil {
		writeError(w, err)
		return
	}

	// Update document status
	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		event := events.NewEvent("document.signature.requested", "document", map[string]any{
			"document_id": doc.ID,
			"signer_id":   req.SignerID,
		}).WithActor(requestedBy, "worker", doc.OwnerAgencyID)

		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusCreated, sig)
}

// SignDocument signs a document
func (h *Handler) SignDocument(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	user := auth.GetUser(r.Context())
	signerID := types.NewID()
	if user != nil {
		signerID = user.ID
	}

	// In production, signature data would come from the request
	if err := doc.Sign(signerID, nil, nil, nil); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	// Update signature and document
	for _, s := range doc.Signatures {
		if s.SignerID == signerID && s.Status == SignatureStatusSigned {
			if err := h.repo.UpdateSignature(r.Context(), &s); err != nil {
				writeError(w, err)
				return
			}
			break
		}
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		event := events.NewEvent("document.signed", "document", map[string]any{
			"document_id": doc.ID,
			"signer_id":   signerID,
			"all_signed":  doc.Status == DocumentStatusSigned,
		}).WithActor(signerID, "worker", types.ID(""))

		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
}

// RejectSignature rejects a signature
func (h *Handler) RejectSignature(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseID(chi.URLParam(r, "documentID"))
	if err != nil {
		writeError(w, errors.BadRequest("invalid document ID"))
		return
	}

	doc, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	user := auth.GetUser(r.Context())
	signerID := types.NewID()
	if user != nil {
		signerID = user.ID
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := doc.RejectSignature(signerID, req.Reason); err != nil {
		writeError(w, errors.BadRequest(err.Error()))
		return
	}

	if err := h.repo.Update(r.Context(), doc); err != nil {
		writeError(w, err)
		return
	}

	// Publish event
	if h.bus != nil {
		event := events.NewEvent("document.signature.rejected", "document", map[string]any{
			"document_id": doc.ID,
			"signer_id":   signerID,
			"reason":      req.Reason,
		}).WithActor(signerID, "worker", doc.OwnerAgencyID)
		h.bus.Publish(r.Context(), event)
	}

	writeJSON(w, http.StatusOK, doc)
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
