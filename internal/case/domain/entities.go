package domain

import (
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// ParticipantRole defines the role of a participant in a case
type ParticipantRole string

const (
	ParticipantRoleApplicant      ParticipantRole = "applicant"
	ParticipantRoleSubject        ParticipantRole = "subject"
	ParticipantRoleGuardian       ParticipantRole = "guardian"
	ParticipantRoleRepresentative ParticipantRole = "representative"
	ParticipantRoleWitness        ParticipantRole = "witness"
	ParticipantRoleExpert         ParticipantRole = "expert"
	ParticipantRoleOther          ParticipantRole = "other"
)

// Participant represents a person involved in a case
type Participant struct {
	ID        types.ID        `json:"id"`
	CaseID    types.ID        `json:"case_id"`
	CitizenID *types.ID       `json:"citizen_id,omitempty"`
	Role      ParticipantRole `json:"role"`
	Name      string          `json:"name"`

	// Contact info (denormalized for convenience)
	ContactEmail string `json:"contact_email,omitempty"`
	ContactPhone string `json:"contact_phone,omitempty"`

	Notes   string    `json:"notes,omitempty"`
	AddedAt time.Time `json:"added_at"`
	AddedBy types.ID  `json:"added_by"`
}

// AssignmentRole defines the role of an assigned worker
type AssignmentRole string

const (
	AssignmentRoleLead     AssignmentRole = "lead"
	AssignmentRoleSupport  AssignmentRole = "support"
	AssignmentRoleReviewer AssignmentRole = "reviewer"
	AssignmentRoleObserver AssignmentRole = "observer"
)

// AssignmentStatus defines the status of an assignment
type AssignmentStatus string

const (
	AssignmentStatusActive     AssignmentStatus = "active"
	AssignmentStatusCompleted  AssignmentStatus = "completed"
	AssignmentStatusReassigned AssignmentStatus = "reassigned"
	AssignmentStatusDeclined   AssignmentStatus = "declined"
)

// Assignment represents a worker assigned to a case
type Assignment struct {
	ID          types.ID         `json:"id"`
	CaseID      types.ID         `json:"case_id"`
	AgencyID    types.ID         `json:"agency_id"`
	WorkerID    types.ID         `json:"worker_id"`
	Role        AssignmentRole   `json:"role"`
	Status      AssignmentStatus `json:"status"`
	AssignedAt  time.Time        `json:"assigned_at"`
	AssignedBy  types.ID         `json:"assigned_by"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Notes       string           `json:"notes,omitempty"`
}

// Complete marks the assignment as completed
func (a *Assignment) Complete() {
	now := time.Now()
	a.Status = AssignmentStatusCompleted
	a.CompletedAt = &now
}

// CaseEventType defines types of case events
type CaseEventType string

const (
	CaseEventTypeCreated          CaseEventType = "created"
	CaseEventTypeUpdated          CaseEventType = "updated"
	CaseEventTypeStatusChanged    CaseEventType = "status_changed"
	CaseEventTypeAssigned         CaseEventType = "assigned"
	CaseEventTypeReassigned       CaseEventType = "reassigned"
	CaseEventTypeTransferred      CaseEventType = "transferred"
	CaseEventTypeEscalated        CaseEventType = "escalated"
	CaseEventTypeDocumentAdded    CaseEventType = "document_added"
	CaseEventTypeDocumentSigned   CaseEventType = "document_signed"
	CaseEventTypeNoteAdded        CaseEventType = "note_added"
	CaseEventTypeParticipantAdded CaseEventType = "participant_added"
	CaseEventTypeShared           CaseEventType = "shared"
	CaseEventTypeAccessChanged    CaseEventType = "access_changed"
	CaseEventTypeSLAWarning       CaseEventType = "sla_warning"
	CaseEventTypeSLABreached      CaseEventType = "sla_breached"
	CaseEventTypeClosed           CaseEventType = "closed"
	CaseEventTypeReopened         CaseEventType = "reopened"
)

// CaseEvent represents an event in the case timeline
type CaseEvent struct {
	ID            types.ID       `json:"id"`
	CaseID        types.ID       `json:"case_id"`
	Type          CaseEventType  `json:"type"`
	ActorID       types.ID       `json:"actor_id"`
	ActorAgencyID types.ID       `json:"actor_agency_id"`
	Description   string         `json:"description"`
	Data          map[string]any `json:"data,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

// Event is a domain event for publishing
type Event struct {
	Type      string    `json:"type"`
	CaseID    types.ID  `json:"case_id"`
	CaseEvent CaseEvent `json:"case_event"`
}
