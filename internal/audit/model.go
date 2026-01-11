package audit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// canonicalJSON produces deterministic JSON output with sorted map keys.
// This is critical for hash verification - Go maps have random iteration order,
// and PostgreSQL JSONB may reorder keys, so we must sort them for consistent hashing.
func canonicalJSON(v any) ([]byte, error) {
	// First marshal to get the raw JSON
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Parse and re-encode with sorted keys
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	return canonicalMarshal(parsed)
}

func canonicalMarshal(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		// Sort keys and recursively process values
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			// Write key
			keyBytes, _ := json.Marshal(k)
			buf.Write(keyBytes)
			buf.WriteByte(':')
			// Write value recursively
			valBytes, err := canonicalMarshal(val[k])
			if err != nil {
				return nil, err
			}
			buf.Write(valBytes)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil

	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			itemBytes, err := canonicalMarshal(item)
			if err != nil {
				return nil, err
			}
			buf.Write(itemBytes)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil

	default:
		// Primitive types - use standard marshal
		return json.Marshal(val)
	}
}

// ActorType defines the type of actor
type ActorType string

const (
	ActorTypeCitizen  ActorType = "citizen"
	ActorTypeWorker   ActorType = "worker"
	ActorTypeSystem   ActorType = "system"
	ActorTypeExternal ActorType = "external"
)

// AuditEntry represents an immutable audit log entry
type AuditEntry struct {
	ID            types.ID       `json:"id"`
	Sequence      int64          `json:"sequence"`
	Timestamp     time.Time      `json:"timestamp"`
	Hash          string         `json:"hash"`
	PrevHash      string         `json:"prev_hash,omitempty"`

	// Actor
	ActorType     ActorType `json:"actor_type"`
	ActorID       types.ID  `json:"actor_id"`
	ActorAgencyID *types.ID `json:"actor_agency_id,omitempty"`
	ActorIP       string    `json:"actor_ip,omitempty"`
	ActorDevice   string    `json:"actor_device,omitempty"`

	// Action
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   *types.ID `json:"resource_id,omitempty"`

	// Changes
	Changes map[string]any `json:"changes,omitempty"`

	// Context
	CorrelationID *types.ID `json:"correlation_id,omitempty"`
	SessionID     *types.ID `json:"session_id,omitempty"`
	Justification string    `json:"justification,omitempty"`
}

// NewAuditEntry creates a new audit entry
func NewAuditEntry(
	actorType ActorType,
	actorID types.ID,
	actorAgencyID *types.ID,
	action, resourceType string,
	resourceID *types.ID,
	changes map[string]any,
	prevHash string,
) *AuditEntry {
	entry := &AuditEntry{
		ID:            types.NewID(),
		Timestamp:     time.Now().UTC().Truncate(time.Microsecond), // Truncate to microseconds for PostgreSQL compatibility
		PrevHash:      prevHash,
		ActorType:     actorType,
		ActorID:       actorID,
		ActorAgencyID: actorAgencyID,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Changes:       changes,
	}

	// Calculate hash
	entry.Hash = entry.calculateHash()

	return entry
}

// calculateHash calculates the SHA-256 hash of the entry using canonical JSON
// for deterministic output regardless of map key ordering.
func (e *AuditEntry) calculateHash() string {
	// Build the hash input with explicit field ordering and canonical JSON for maps
	// IMPORTANT: Always use UTC for timestamp to ensure consistent hashing regardless
	// of timezone differences between creation and verification
	data := map[string]any{
		"id":            e.ID,
		"timestamp":     e.Timestamp.UTC().Format(time.RFC3339Nano),
		"prev_hash":     e.PrevHash,
		"actor_type":    e.ActorType,
		"actor_id":      e.ActorID,
		"action":        e.Action,
		"resource_type": e.ResourceType,
	}

	// Add optional fields only if present
	if e.ActorAgencyID != nil {
		data["actor_agency_id"] = e.ActorAgencyID
	}
	if e.ResourceID != nil {
		data["resource_id"] = e.ResourceID
	}
	if e.Changes != nil && len(e.Changes) > 0 {
		data["changes"] = e.Changes
	}

	// Use canonical JSON for deterministic key ordering
	jsonData, _ := canonicalJSON(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// VerifyHash verifies the entry's hash
func (e *AuditEntry) VerifyHash() bool {
	return e.Hash == e.calculateHash()
}

// ComputeHash computes and returns the correct hash for this entry
func (e *AuditEntry) ComputeHash() string {
	return e.calculateHash()
}

// WithContext adds context information to the entry
func (e *AuditEntry) WithContext(correlationID, sessionID *types.ID, justification string) *AuditEntry {
	e.CorrelationID = correlationID
	e.SessionID = sessionID
	e.Justification = justification
	return e
}

// WithRequest adds request information to the entry
func (e *AuditEntry) WithRequest(ip, device string) *AuditEntry {
	e.ActorIP = ip
	e.ActorDevice = device
	return e
}

// ListEntriesFilter defines filters for listing audit entries
type ListEntriesFilter struct {
	ActorID      *types.ID  `json:"actor_id,omitempty"`
	ActorType    *ActorType `json:"actor_type,omitempty"`
	Action       string     `json:"action,omitempty"`
	ResourceType string     `json:"resource_type,omitempty"`
	ResourceID   *types.ID  `json:"resource_id,omitempty"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
}

// Common audit actions
const (
	// Authentication
	ActionLogin       = "auth.login"
	ActionLogout      = "auth.logout"
	ActionLoginFailed = "auth.login_failed"

	// Cases
	ActionCaseCreated     = "case.created"
	ActionCaseUpdated     = "case.updated"
	ActionCaseViewed      = "case.viewed"
	ActionCaseShared      = "case.shared"
	ActionCaseTransferred = "case.transferred"
	ActionCaseClosed      = "case.closed"

	// Documents
	ActionDocumentCreated    = "document.created"
	ActionDocumentViewed     = "document.viewed"
	ActionDocumentUploaded   = "document.uploaded"
	ActionDocumentSigned     = "document.signed"
	ActionDocumentDownloaded = "document.downloaded"

	// Sensitive data access
	ActionJMBGAccessed = "sensitive.jmbg_accessed"
	ActionDataExported = "sensitive.data_exported"

	// Admin
	ActionWorkerCreated = "admin.worker_created"
	ActionWorkerUpdated = "admin.worker_updated"
	ActionRoleChanged   = "admin.role_changed"

	// Corrections - append-only corrections to previous entries
	// Instead of modifying data, create a correction event that references the original
	ActionCorrection         = "correction"           // General correction
	ActionCorrectionData     = "correction.data"      // Data entry error correction
	ActionCorrectionVoid     = "correction.void"      // Void a previous entry
	ActionCorrectionOverride = "correction.override"  // Administrative override
)

// CorrectionReason defines standard reasons for corrections
type CorrectionReason string

const (
	CorrectionReasonDataEntry     CorrectionReason = "data_entry_error"      // Typographical or data entry mistake
	CorrectionReasonLegalRequirement CorrectionReason = "legal_requirement"  // Required by law/regulation
	CorrectionReasonCourtOrder    CorrectionReason = "court_order"           // Ordered by court
	CorrectionReasonCitizenRequest CorrectionReason = "citizen_request"      // GDPR-style correction request
	CorrectionReasonSystemError   CorrectionReason = "system_error"          // Technical system error
	CorrectionReasonOther         CorrectionReason = "other"                 // Other (requires justification)
)

// CorrectionEntry represents data needed for a correction event
type CorrectionEntry struct {
	OriginalEntryID   types.ID         `json:"original_entry_id"`    // ID of entry being corrected
	OriginalAction    string           `json:"original_action"`      // Original action type
	OriginalTimestamp time.Time        `json:"original_timestamp"`   // When original occurred
	Reason            CorrectionReason `json:"reason"`               // Why correction is needed
	Justification     string           `json:"justification"`        // Detailed explanation (required)
	ApprovedBy        *types.ID        `json:"approved_by,omitempty"` // Supervisor who approved (if required)
	OldValue          map[string]any   `json:"old_value,omitempty"`  // What was recorded
	NewValue          map[string]any   `json:"new_value,omitempty"`  // What should have been recorded
}

// NewCorrectionAuditEntry creates an audit entry for a correction
// This maintains append-only integrity while allowing for corrections
func NewCorrectionAuditEntry(
	actorType ActorType,
	actorID types.ID,
	actorAgencyID *types.ID,
	correction CorrectionEntry,
	prevHash string,
) *AuditEntry {
	changes := map[string]any{
		"correction": map[string]any{
			"original_entry_id":   correction.OriginalEntryID,
			"original_action":     correction.OriginalAction,
			"original_timestamp":  correction.OriginalTimestamp,
			"reason":              correction.Reason,
			"justification":       correction.Justification,
			"old_value":           correction.OldValue,
			"new_value":           correction.NewValue,
		},
	}

	if correction.ApprovedBy != nil {
		changes["correction"].(map[string]any)["approved_by"] = correction.ApprovedBy
	}

	return NewAuditEntry(
		actorType,
		actorID,
		actorAgencyID,
		ActionCorrection,
		"correction",
		&correction.OriginalEntryID, // Reference to original entry
		changes,
		prevHash,
	)
}
