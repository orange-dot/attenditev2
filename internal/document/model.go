package document

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// DocumentType defines the type of document
type DocumentType string

const (
	DocumentTypeReport         DocumentType = "REPORT"
	DocumentTypeStatement      DocumentType = "STATEMENT"
	DocumentTypeDecision       DocumentType = "DECISION"
	DocumentTypeCertificate    DocumentType = "CERTIFICATE"
	DocumentTypeEvidence       DocumentType = "EVIDENCE"
	DocumentTypeForm           DocumentType = "FORM"
	DocumentTypeCorrespondence DocumentType = "CORRESPONDENCE"
	DocumentTypeContract       DocumentType = "CONTRACT"
	DocumentTypeOther          DocumentType = "OTHER"
)

// DocumentStatus defines the status of a document
type DocumentStatus string

const (
	DocumentStatusDraft            DocumentStatus = "draft"
	DocumentStatusPendingSignature DocumentStatus = "pending_signature"
	DocumentStatusPartiallySigned  DocumentStatus = "partially_signed"
	DocumentStatusSigned           DocumentStatus = "signed"
	DocumentStatusRejected         DocumentStatus = "rejected"
	DocumentStatusArchived         DocumentStatus = "archived"
	DocumentStatusVoid             DocumentStatus = "void"
)

// Document represents a document in the system
type Document struct {
	ID             types.ID       `json:"id"`
	DocumentNumber string         `json:"document_number"`
	Type           DocumentType   `json:"type"`
	Status         DocumentStatus `json:"status"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`

	// Ownership
	OwnerAgencyID types.ID `json:"owner_agency_id"`
	CreatedBy     types.ID `json:"created_by"`

	// References
	CaseID *types.ID `json:"case_id,omitempty"`

	// Versioning
	CurrentVersion int               `json:"current_version"`
	Versions       []DocumentVersion `json:"versions,omitempty"`

	// Signatures
	Signatures  []Signature `json:"signatures,omitempty"`
	RequiresSig []types.ID  `json:"requires_sig,omitempty"`

	// Sharing
	SharedWith []types.ID `json:"shared_with,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewDocument creates a new document
func NewDocument(
	docType DocumentType,
	title, description string,
	ownerAgencyID, createdBy types.ID,
	caseID *types.ID,
) (*Document, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if ownerAgencyID.IsZero() {
		return nil, fmt.Errorf("owner agency is required")
	}

	now := time.Now()
	return &Document{
		ID:             types.NewID(),
		DocumentNumber: generateDocumentNumber(docType),
		Type:           docType,
		Status:         DocumentStatusDraft,
		Title:          title,
		Description:    description,
		OwnerAgencyID:  ownerAgencyID,
		CreatedBy:      createdBy,
		CaseID:         caseID,
		CurrentVersion: 0,
		SharedWith:     []types.ID{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// AddVersion adds a new version to the document
func (d *Document) AddVersion(filePath, mimeType string, fileSize int64, content io.Reader, createdBy types.ID, changeSummary string) (*DocumentVersion, error) {
	if d.Status == DocumentStatusVoid || d.Status == DocumentStatusArchived {
		return nil, fmt.Errorf("cannot add version to %s document", d.Status)
	}

	// Calculate hash
	hash := sha256.New()
	if _, err := io.Copy(hash, content); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	d.CurrentVersion++
	version := DocumentVersion{
		ID:            types.NewID(),
		DocumentID:    d.ID,
		Version:       d.CurrentVersion,
		FilePath:      filePath,
		FileHash:      fileHash,
		FileSize:      fileSize,
		MimeType:      mimeType,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
		ChangeSummary: changeSummary,
	}

	d.Versions = append(d.Versions, version)
	d.UpdatedAt = time.Now()

	// Reset signatures when new version is added
	d.Signatures = []Signature{}
	if d.Status == DocumentStatusSigned || d.Status == DocumentStatusPartiallySigned {
		d.Status = DocumentStatusDraft
	}

	return &version, nil
}

// RequestSignature requests a signature from a worker
func (d *Document) RequestSignature(signerID, signerAgencyID, requestedBy types.ID, sigType SignatureType, deadline *time.Time, reason, location string) (*Signature, error) {
	if d.CurrentVersion == 0 {
		return nil, fmt.Errorf("document must have at least one version")
	}

	// Check if already requested
	for _, s := range d.Signatures {
		if s.SignerID == signerID && s.Status == SignatureStatusPending {
			return nil, fmt.Errorf("signature already requested from this signer")
		}
	}

	sig := Signature{
		ID:            types.NewID(),
		DocumentID:    d.ID,
		Version:       d.CurrentVersion,
		SignerID:      signerID,
		SignerAgencyID: signerAgencyID,
		Type:          sigType,
		Status:        SignatureStatusPending,
		Reason:        reason,
		Location:      location,
		CreatedAt:     time.Now(),
	}

	d.Signatures = append(d.Signatures, sig)
	d.RequiresSig = append(d.RequiresSig, signerID)

	if d.Status == DocumentStatusDraft {
		d.Status = DocumentStatusPendingSignature
	}

	d.UpdatedAt = time.Now()

	return &sig, nil
}

// Sign signs the document
func (d *Document) Sign(signerID types.ID, signatureData, certificate, timestampToken []byte) error {
	// Find pending signature
	var sigIndex = -1
	for i, s := range d.Signatures {
		if s.SignerID == signerID && s.Status == SignatureStatusPending {
			sigIndex = i
			break
		}
	}

	if sigIndex == -1 {
		return fmt.Errorf("no pending signature found for this signer")
	}

	now := time.Now()
	d.Signatures[sigIndex].Status = SignatureStatusSigned
	d.Signatures[sigIndex].SignatureData = signatureData
	d.Signatures[sigIndex].Certificate = certificate
	d.Signatures[sigIndex].TimestampToken = timestampToken
	d.Signatures[sigIndex].SignedAt = &now

	// Check if all required signatures are done
	allSigned := true
	for _, s := range d.Signatures {
		if s.Status == SignatureStatusPending {
			allSigned = false
			break
		}
	}

	if allSigned {
		d.Status = DocumentStatusSigned
	} else {
		d.Status = DocumentStatusPartiallySigned
	}

	d.UpdatedAt = time.Now()

	return nil
}

// RejectSignature rejects a signature request
func (d *Document) RejectSignature(signerID types.ID, reason string) error {
	for i, s := range d.Signatures {
		if s.SignerID == signerID && s.Status == SignatureStatusPending {
			d.Signatures[i].Status = SignatureStatusRejected
			d.Signatures[i].Reason = reason
			d.Status = DocumentStatusRejected
			d.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("no pending signature found for this signer")
}

// Share shares the document with an agency
func (d *Document) Share(agencyID types.ID) error {
	if agencyID == d.OwnerAgencyID {
		return fmt.Errorf("cannot share with owner agency")
	}

	for _, id := range d.SharedWith {
		if id == agencyID {
			return nil // Already shared
		}
	}

	d.SharedWith = append(d.SharedWith, agencyID)
	d.UpdatedAt = time.Now()

	return nil
}

// Archive archives the document
func (d *Document) Archive() error {
	if d.Status == DocumentStatusVoid {
		return fmt.Errorf("cannot archive voided document")
	}

	d.Status = DocumentStatusArchived
	d.UpdatedAt = time.Now()

	return nil
}

// Void voids the document
func (d *Document) Void() error {
	if d.Status == DocumentStatusArchived {
		return fmt.Errorf("cannot void archived document")
	}

	d.Status = DocumentStatusVoid
	d.UpdatedAt = time.Now()

	return nil
}

// CanAccess checks if an agency can access the document
func (d *Document) CanAccess(agencyID types.ID) bool {
	if agencyID == d.OwnerAgencyID {
		return true
	}

	for _, id := range d.SharedWith {
		if id == agencyID {
			return true
		}
	}

	return false
}

// DocumentVersion represents a version of a document
type DocumentVersion struct {
	ID            types.ID  `json:"id"`
	DocumentID    types.ID  `json:"document_id"`
	Version       int       `json:"version"`
	FilePath      string    `json:"file_path"` // MinIO path
	FileHash      string    `json:"file_hash"` // SHA-256
	FileSize      int64     `json:"file_size"` // bytes
	MimeType      string    `json:"mime_type"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     types.ID  `json:"created_by"`
	ChangeSummary string    `json:"change_summary,omitempty"`
}

// SignatureType defines the type of signature
type SignatureType string

const (
	SignatureTypeSimple    SignatureType = "simple"    // Click to sign
	SignatureTypeAdvanced  SignatureType = "advanced"  // Certificate-based
	SignatureTypeQualified SignatureType = "qualified" // QES (eIDAS)
)

// SignatureStatus defines the status of a signature
type SignatureStatus string

const (
	SignatureStatusPending  SignatureStatus = "pending"
	SignatureStatusSigned   SignatureStatus = "signed"
	SignatureStatusRejected SignatureStatus = "rejected"
	SignatureStatusRevoked  SignatureStatus = "revoked"
)

// Signature represents a signature on a document
type Signature struct {
	ID              types.ID        `json:"id"`
	DocumentID      types.ID        `json:"document_id"`
	Version         int             `json:"version"`
	SignerID        types.ID        `json:"signer_id"`
	SignerAgencyID  types.ID        `json:"signer_agency_id"`
	Type            SignatureType   `json:"type"`
	Status          SignatureStatus `json:"status"`
	SignatureData   []byte          `json:"-"` // PAdES/XAdES
	Certificate     []byte          `json:"-"`
	TimestampToken  []byte          `json:"-"` // TSA token
	Reason          string          `json:"reason,omitempty"`
	Location        string          `json:"location,omitempty"`
	SignedAt        *time.Time      `json:"signed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// generateDocumentNumber generates a unique document number
func generateDocumentNumber(docType DocumentType) string {
	prefix := map[DocumentType]string{
		DocumentTypeReport:         "RPT",
		DocumentTypeStatement:      "STM",
		DocumentTypeDecision:       "DEC",
		DocumentTypeCertificate:    "CRT",
		DocumentTypeEvidence:       "EVD",
		DocumentTypeForm:           "FRM",
		DocumentTypeCorrespondence: "COR",
		DocumentTypeContract:       "CON",
		DocumentTypeOther:          "DOC",
	}

	year := time.Now().Year()
	seq := time.Now().UnixNano() % 1000000

	return fmt.Sprintf("%s-%d-%06d", prefix[docType], year, seq)
}

// --- Request/Response types ---

type CreateDocumentRequest struct {
	Type        DocumentType `json:"type"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	CaseID      *types.ID    `json:"case_id,omitempty"`
}

type UpdateDocumentRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

type RequestSignatureRequest struct {
	SignerID       types.ID      `json:"signer_id"`
	SignerAgencyID types.ID      `json:"signer_agency_id"`
	Type           SignatureType `json:"type"`
	Reason         string        `json:"reason,omitempty"`
	Location       string        `json:"location,omitempty"`
}

type ShareDocumentRequest struct {
	AgencyID types.ID `json:"agency_id"`
}

type ListDocumentsFilter struct {
	Type      *DocumentType   `json:"type,omitempty"`
	Status    *DocumentStatus `json:"status,omitempty"`
	CaseID    *types.ID       `json:"case_id,omitempty"`
	Search    string          `json:"search,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
}
