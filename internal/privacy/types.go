// Package privacy provides pseudonymization and data protection functionality.
// It implements "privacy by design" where the central system never sees
// personally identifiable information (PII) like JMBG, names, addresses.
package privacy

import (
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// PseudonymID is a privacy-preserving identifier that replaces JMBG in the central system.
// It's generated using HMAC-SHA256 with a facility-specific secret key.
// Format: "PSE-" followed by 32 hex characters.
type PseudonymID string

// IsZero returns true if the pseudonym is empty.
func (p PseudonymID) IsZero() bool {
	return p == ""
}

// String returns the string representation of the pseudonym.
func (p PseudonymID) String() string {
	return string(p)
}

// DataAccessLevel defines what level of data access is permitted.
type DataAccessLevel int

const (
	// DataAccessLevelAggregated allows only aggregated/statistical data.
	// No individual records can be accessed.
	DataAccessLevelAggregated DataAccessLevel = 0

	// DataAccessLevelPseudonymized allows pseudonymized individual records.
	// Requires explicit approval.
	DataAccessLevelPseudonymized DataAccessLevel = 1

	// DataAccessLevelLinkable allows linking records to real identities.
	// Emergency only - requires court order or life threat.
	DataAccessLevelLinkable DataAccessLevel = 2
)

// String returns the string representation of the access level.
func (d DataAccessLevel) String() string {
	switch d {
	case DataAccessLevelAggregated:
		return "aggregated"
	case DataAccessLevelPseudonymized:
		return "pseudonymized"
	case DataAccessLevelLinkable:
		return "linkable"
	default:
		return "unknown"
	}
}

// LegalBasis defines the legal justification for depseudonymization.
type LegalBasis string

const (
	// LegalBasisCourtOrder - court order for identity revelation.
	LegalBasisCourtOrder LegalBasis = "court_order"

	// LegalBasisLifeThreat - immediate threat to life.
	LegalBasisLifeThreat LegalBasis = "life_threat"

	// LegalBasisChildProtection - child protection case.
	LegalBasisChildProtection LegalBasis = "child_protection"

	// LegalBasisLawEnforcement - law enforcement investigation.
	LegalBasisLawEnforcement LegalBasis = "law_enforcement"

	// LegalBasisSubjectConsent - subject has given explicit consent.
	LegalBasisSubjectConsent LegalBasis = "subject_consent"
)

// RequiresManualApproval returns true if the legal basis requires
// manual approval from a supervisor.
func (l LegalBasis) RequiresManualApproval() bool {
	switch l {
	case LegalBasisCourtOrder, LegalBasisLifeThreat:
		return false // Auto-approved for emergencies
	default:
		return true
	}
}

// RequestStatus defines the status of a depseudonymization request.
type RequestStatus string

const (
	RequestStatusPending  RequestStatus = "pending"
	RequestStatusApproved RequestStatus = "approved"
	RequestStatusRejected RequestStatus = "rejected"
	RequestStatusExpired  RequestStatus = "expired"
	RequestStatusRevoked  RequestStatus = "revoked"
)

// DepseudonymizationRequest represents a request to reveal real identity.
type DepseudonymizationRequest struct {
	ID              types.ID      `json:"id"`
	PseudonymID     PseudonymID   `json:"pseudonym_id"`
	RequestorID     types.ID      `json:"requestor_id"`
	RequestorAgency types.ID      `json:"requestor_agency"`
	Purpose         string        `json:"purpose"`
	LegalBasis      LegalBasis    `json:"legal_basis"`
	Justification   string        `json:"justification"`
	CaseID          types.ID      `json:"case_id,omitempty"`
	RequestedAt     time.Time     `json:"requested_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
	Status          RequestStatus `json:"status"`
	ApprovedBy      *types.ID     `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time    `json:"approved_at,omitempty"`
	RejectionReason string        `json:"rejection_reason,omitempty"`
}

// IsActive returns true if the request is approved and not expired.
func (r *DepseudonymizationRequest) IsActive() bool {
	return r.Status == RequestStatusApproved && time.Now().Before(r.ExpiresAt)
}

// DepseudonymizationToken is a time-limited token for accessing real identity.
type DepseudonymizationToken struct {
	Token       string      `json:"token"`
	RequestID   types.ID    `json:"request_id"`
	PseudonymID PseudonymID `json:"pseudonym_id"`
	IssuedAt    time.Time   `json:"issued_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	UsedCount   int         `json:"used_count"`
	MaxUses     int         `json:"max_uses"`
	LastUsedAt  *time.Time  `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time  `json:"revoked_at,omitempty"`
	RevokedBy   *types.ID   `json:"revoked_by,omitempty"`
}

// IsValid returns true if the token can still be used.
func (t *DepseudonymizationToken) IsValid() bool {
	if t.RevokedAt != nil {
		return false
	}
	if time.Now().After(t.ExpiresAt) {
		return false
	}
	if t.UsedCount >= t.MaxUses {
		return false
	}
	return true
}

// PIIField represents types of personally identifiable information.
type PIIField string

const (
	PIIFieldJMBG    PIIField = "jmbg"
	PIIFieldName    PIIField = "name"
	PIIFieldAddress PIIField = "address"
	PIIFieldPhone   PIIField = "phone"
	PIIFieldEmail   PIIField = "email"
	PIIFieldLBO     PIIField = "lbo" // Health insurance number
)

// PIIViolation represents a detected PII leak attempt.
type PIIViolation struct {
	ID            types.ID  `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Field         PIIField  `json:"field"`
	Location      string    `json:"location"` // API path, event type, etc.
	ActorID       types.ID  `json:"actor_id,omitempty"`
	ActorAgency   types.ID  `json:"actor_agency,omitempty"`
	Blocked       bool      `json:"blocked"`
	RawValue      string    `json:"-"` // Never exposed in JSON
	MaskedValue   string    `json:"masked_value"`
	RequestPath   string    `json:"request_path,omitempty"`
	RequestMethod string    `json:"request_method,omitempty"`
	RequestIP     string    `json:"request_ip,omitempty"`
}

// AIAccessRequest represents a request for AI system to access data.
type AIAccessRequest struct {
	ID             types.ID        `json:"id"`
	AISystemID     string          `json:"ai_system_id"`
	RequestedLevel DataAccessLevel `json:"requested_level"`
	Purpose        string          `json:"purpose"`
	DataScope      DataScope       `json:"data_scope"`
	RequestedBy    types.ID        `json:"requested_by"`
	RequestedAt    time.Time       `json:"requested_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	Status         RequestStatus   `json:"status"`
	ApprovedBy     *types.ID       `json:"approved_by,omitempty"`
	ApprovedAt     *time.Time      `json:"approved_at,omitempty"`
}

// IsActive returns true if the AI access is approved and not expired.
func (r *AIAccessRequest) IsActive() bool {
	return r.Status == RequestStatusApproved && time.Now().Before(r.ExpiresAt)
}

// DataScope defines what data an AI system can access.
type DataScope struct {
	CaseTypes     []string    `json:"case_types,omitempty"`
	AgencyIDs     []types.ID  `json:"agency_ids,omitempty"`
	TimeRangeFrom *time.Time  `json:"time_range_from,omitempty"`
	TimeRangeTo   *time.Time  `json:"time_range_to,omitempty"`
	MaxRecords    int         `json:"max_records,omitempty"`
	ExcludeFields []PIIField  `json:"exclude_fields,omitempty"`
}

// PseudonymMapping represents the local mapping between JMBG and PseudonymID.
// This is stored ONLY in the local facility, never in the central system.
type PseudonymMapping struct {
	ID            types.ID    `json:"id"`
	JMBGHash      string      `json:"-"` // SHA-256 of JMBG, never exposed
	JMBGEncrypted []byte      `json:"-"` // Encrypted JMBG, never exposed
	PseudonymID   PseudonymID `json:"pseudonym_id"`
	FacilityCode  string      `json:"facility_code"`
	CreatedAt     time.Time   `json:"created_at"`
}

// Audit action constants for privacy operations.
const (
	AuditActionPseudonymCreated     = "privacy.pseudonym_created"
	AuditActionDepseudoRequested    = "privacy.depseudo_requested"
	AuditActionDepseudoApproved     = "privacy.depseudo_approved"
	AuditActionDepseudoRejected     = "privacy.depseudo_rejected"
	AuditActionDepseudoUsed         = "privacy.depseudo_used"
	AuditActionDepseudoExpired      = "privacy.depseudo_expired"
	AuditActionDepseudoRevoked      = "privacy.depseudo_revoked"
	AuditActionPIIViolationDetected = "privacy.pii_violation"
	AuditActionPIIViolationBlocked  = "privacy.pii_blocked"
	AuditActionAIAccessRequested    = "privacy.ai_access_requested"
	AuditActionAIAccessGranted      = "privacy.ai_access_granted"
	AuditActionAIAccessDenied       = "privacy.ai_access_denied"
)
