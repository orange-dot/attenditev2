package domain

import (
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// CaseType defines the type of case
type CaseType string

const (
	CaseTypeChildWelfare     CaseType = "CHILD_WELFARE"
	CaseTypeCriminal         CaseType = "CRIMINAL"
	CaseTypeAdministrative   CaseType = "ADMINISTRATIVE"
	CaseTypeHealthcare       CaseType = "HEALTHCARE"
	CaseTypeSocialAssistance CaseType = "SOCIAL_ASSISTANCE"
	CaseTypeTax              CaseType = "TAX"
	CaseTypeCivil            CaseType = "CIVIL"
)

// CaseStatus defines the status of a case
type CaseStatus string

const (
	CaseStatusDraft            CaseStatus = "draft"
	CaseStatusOpen             CaseStatus = "open"
	CaseStatusInProgress       CaseStatus = "in_progress"
	CaseStatusPendingTransfer  CaseStatus = "pending_transfer"
	CaseStatusPendingDocuments CaseStatus = "pending_documents"
	CaseStatusUnderReview      CaseStatus = "under_review"
	CaseStatusEscalated        CaseStatus = "escalated"
	CaseStatusClosed           CaseStatus = "closed"
	CaseStatusArchived         CaseStatus = "archived"
)

// Priority defines case priority
type Priority string

const (
	PriorityLow       Priority = "low"
	PriorityMedium    Priority = "medium"
	PriorityHigh      Priority = "high"
	PriorityUrgent    Priority = "urgent"
	PriorityEmergency Priority = "emergency"
)

// SLAStatus defines SLA tracking status
type SLAStatus string

const (
	SLAStatusOnTrack  SLAStatus = "on_track"
	SLAStatusAtRisk   SLAStatus = "at_risk"
	SLAStatusBreached SLAStatus = "breached"
	SLAStatusPaused   SLAStatus = "paused"
)

// AccessLevel defines the level of access for shared agencies
type AccessLevel int

const (
	AccessLevelNone       AccessLevel = 0
	AccessLevelRead       AccessLevel = 1
	AccessLevelComment    AccessLevel = 2
	AccessLevelContribute AccessLevel = 3
	AccessLevelFull       AccessLevel = 4
)

// Case is the aggregate root for case management
type Case struct {
	ID          types.ID   `json:"id"`
	CaseNumber  string     `json:"case_number"`
	Type        CaseType   `json:"type"`
	Status      CaseStatus `json:"status"`
	Priority    Priority   `json:"priority"`
	Title       string     `json:"title"`
	Description string     `json:"description"`

	// Ownership
	OwningAgencyID types.ID `json:"owning_agency_id"`
	LeadWorkerID   types.ID `json:"lead_worker_id"`

	// Embedded entities
	Participants []Participant `json:"participants"`
	Assignments  []Assignment  `json:"assignments"`
	Events       []CaseEvent   `json:"events,omitempty"`

	// SLA
	SLADeadline *time.Time `json:"sla_deadline,omitempty"`
	SLAStatus   SLAStatus  `json:"sla_status"`

	// Cross-agency sharing
	SharedWith   []types.ID             `json:"shared_with"`
	AccessLevels map[string]AccessLevel `json:"access_levels"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`

	// Domain events (not persisted, used for event sourcing)
	domainEvents []Event
}

// NewCase creates a new case with validation
func NewCase(
	caseType CaseType,
	priority Priority,
	title, description string,
	owningAgencyID, leadWorkerID types.ID,
) (*Case, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if owningAgencyID.IsZero() {
		return nil, fmt.Errorf("owning agency is required")
	}
	if leadWorkerID.IsZero() {
		return nil, fmt.Errorf("lead worker is required")
	}

	now := time.Now()
	c := &Case{
		ID:             types.NewID(),
		CaseNumber:     generateCaseNumber(caseType),
		Type:           caseType,
		Status:         CaseStatusDraft,
		Priority:       priority,
		Title:          title,
		Description:    description,
		OwningAgencyID: owningAgencyID,
		LeadWorkerID:   leadWorkerID,
		SLAStatus:      SLAStatusOnTrack,
		SharedWith:     []types.ID{},
		AccessLevels:   make(map[string]AccessLevel),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Calculate SLA deadline based on type and priority
	c.SLADeadline = c.calculateSLADeadline()

	// Add creation event
	c.addEvent(CaseEventTypeCreated, leadWorkerID, owningAgencyID, "Case created", nil)

	return c, nil
}

// Open transitions the case from draft to open
func (c *Case) Open(actorID, actorAgencyID types.ID) error {
	if c.Status != CaseStatusDraft {
		return fmt.Errorf("can only open a draft case")
	}

	c.Status = CaseStatusOpen
	c.UpdatedAt = time.Now()
	c.addEvent(CaseEventTypeStatusChanged, actorID, actorAgencyID, "Case opened", map[string]any{
		"old_status": CaseStatusDraft,
		"new_status": CaseStatusOpen,
	})

	return nil
}

// StartProgress transitions the case to in_progress
func (c *Case) StartProgress(actorID, actorAgencyID types.ID) error {
	if c.Status != CaseStatusOpen {
		return fmt.Errorf("can only start progress on an open case")
	}

	c.Status = CaseStatusInProgress
	c.UpdatedAt = time.Now()
	c.addEvent(CaseEventTypeStatusChanged, actorID, actorAgencyID, "Case work started", map[string]any{
		"old_status": CaseStatusOpen,
		"new_status": CaseStatusInProgress,
	})

	return nil
}

// Close closes the case
func (c *Case) Close(actorID, actorAgencyID types.ID, resolution string) error {
	if c.Status == CaseStatusClosed || c.Status == CaseStatusArchived {
		return fmt.Errorf("case is already closed")
	}

	// Check for pending assignments
	for _, a := range c.Assignments {
		if a.Status == AssignmentStatusActive {
			return fmt.Errorf("cannot close case with active assignments")
		}
	}

	now := time.Now()
	oldStatus := c.Status
	c.Status = CaseStatusClosed
	c.ClosedAt = &now
	c.UpdatedAt = now

	c.addEvent(CaseEventTypeClosed, actorID, actorAgencyID, resolution, map[string]any{
		"old_status": oldStatus,
		"resolution": resolution,
	})

	return nil
}

// AddParticipant adds a participant to the case
func (c *Case) AddParticipant(participant Participant, actorID, actorAgencyID types.ID) error {
	if participant.Name == "" {
		return fmt.Errorf("participant name is required")
	}

	participant.ID = types.NewID()
	participant.CaseID = c.ID
	participant.AddedAt = time.Now()
	participant.AddedBy = actorID

	c.Participants = append(c.Participants, participant)
	c.UpdatedAt = time.Now()

	c.addEvent(CaseEventTypeParticipantAdded, actorID, actorAgencyID,
		fmt.Sprintf("Added participant: %s (%s)", participant.Name, participant.Role), nil)

	return nil
}

// Assign assigns a worker to the case
func (c *Case) Assign(workerID, agencyID types.ID, role AssignmentRole, actorID, actorAgencyID types.ID) error {
	// Check if already assigned
	for _, a := range c.Assignments {
		if a.WorkerID == workerID && a.Status == AssignmentStatusActive {
			return fmt.Errorf("worker is already assigned to this case")
		}
	}

	assignment := Assignment{
		ID:         types.NewID(),
		CaseID:     c.ID,
		AgencyID:   agencyID,
		WorkerID:   workerID,
		Role:       role,
		Status:     AssignmentStatusActive,
		AssignedAt: time.Now(),
		AssignedBy: actorID,
	}

	c.Assignments = append(c.Assignments, assignment)
	c.UpdatedAt = time.Now()

	c.addEvent(CaseEventTypeAssigned, actorID, actorAgencyID,
		fmt.Sprintf("Assigned worker as %s", role), map[string]any{
			"worker_id": workerID,
			"agency_id": agencyID,
			"role":      role,
		})

	return nil
}

// Share shares the case with another agency
func (c *Case) Share(agencyID types.ID, level AccessLevel, actorID, actorAgencyID types.ID) error {
	if agencyID == c.OwningAgencyID {
		return fmt.Errorf("cannot share with owning agency")
	}

	// Update or add sharing
	found := false
	for i, id := range c.SharedWith {
		if id == agencyID {
			found = true
			if level == AccessLevelNone {
				// Remove sharing
				c.SharedWith = append(c.SharedWith[:i], c.SharedWith[i+1:]...)
				delete(c.AccessLevels, agencyID.String())
			} else {
				c.AccessLevels[agencyID.String()] = level
			}
			break
		}
	}

	if !found && level != AccessLevelNone {
		c.SharedWith = append(c.SharedWith, agencyID)
		c.AccessLevels[agencyID.String()] = level
	}

	c.UpdatedAt = time.Now()

	c.addEvent(CaseEventTypeShared, actorID, actorAgencyID,
		fmt.Sprintf("Shared with agency (level: %d)", level), map[string]any{
			"agency_id":    agencyID,
			"access_level": level,
		})

	return nil
}

// Transfer transfers case ownership to another agency
func (c *Case) Transfer(toAgencyID, newLeadWorkerID, actorID, actorAgencyID types.ID, reason string) error {
	if toAgencyID == c.OwningAgencyID {
		return fmt.Errorf("cannot transfer to same agency")
	}

	fromAgencyID := c.OwningAgencyID

	// Transfer ownership first
	c.OwningAgencyID = toAgencyID
	c.LeadWorkerID = newLeadWorkerID
	c.Status = CaseStatusOpen // Reset to open for new agency
	c.UpdatedAt = time.Now()

	// Previous owner gets read access (must be after ownership change)
	c.Share(fromAgencyID, AccessLevelRead, actorID, actorAgencyID)

	// Remove new owner from shared list
	for i, id := range c.SharedWith {
		if id == toAgencyID {
			c.SharedWith = append(c.SharedWith[:i], c.SharedWith[i+1:]...)
			delete(c.AccessLevels, toAgencyID.String())
			break
		}
	}

	c.addEvent(CaseEventTypeTransferred, actorID, actorAgencyID, reason, map[string]any{
		"from_agency":     fromAgencyID,
		"to_agency":       toAgencyID,
		"new_lead_worker": newLeadWorkerID,
	})

	return nil
}

// Escalate escalates the case
func (c *Case) Escalate(level int, reason string, escalatedTo types.ID, actorID, actorAgencyID types.ID) error {
	c.Status = CaseStatusEscalated
	c.UpdatedAt = time.Now()

	c.addEvent(CaseEventTypeEscalated, actorID, actorAgencyID, reason, map[string]any{
		"level":        level,
		"escalated_to": escalatedTo,
	})

	return nil
}

// CanAccess checks if an agency can access this case with the required level
func (c *Case) CanAccess(agencyID types.ID, requiredLevel AccessLevel) bool {
	// Owner always has full access
	if agencyID == c.OwningAgencyID {
		return true
	}

	// Check shared access
	level, ok := c.AccessLevels[agencyID.String()]
	if !ok {
		return false
	}

	return level >= requiredLevel
}

// GetDomainEvents returns and clears domain events
func (c *Case) GetDomainEvents() []Event {
	events := c.domainEvents
	c.domainEvents = nil
	return events
}

// addEvent adds a domain event
func (c *Case) addEvent(eventType CaseEventType, actorID, actorAgencyID types.ID, description string, data map[string]any) {
	event := CaseEvent{
		ID:            types.NewID(),
		CaseID:        c.ID,
		Type:          eventType,
		ActorID:       actorID,
		ActorAgencyID: actorAgencyID,
		Description:   description,
		Data:          data,
		Timestamp:     time.Now(),
	}

	c.Events = append(c.Events, event)

	// Also add to domain events for publishing
	c.domainEvents = append(c.domainEvents, Event{
		Type:      string(eventType),
		CaseID:    c.ID,
		CaseEvent: event,
	})
}

// calculateSLADeadline calculates the SLA deadline based on type and priority
func (c *Case) calculateSLADeadline() *time.Time {
	// Base SLA in hours by type
	baseSLA := map[CaseType]int{
		CaseTypeChildWelfare:     24,
		CaseTypeCriminal:         48,
		CaseTypeAdministrative:   120,
		CaseTypeHealthcare:       24,
		CaseTypeSocialAssistance: 72,
		CaseTypeTax:              240,
		CaseTypeCivil:            168,
	}

	// Priority multiplier (lower = faster)
	priorityMultiplier := map[Priority]float64{
		PriorityEmergency: 0.25,
		PriorityUrgent:    0.5,
		PriorityHigh:      0.75,
		PriorityMedium:    1.0,
		PriorityLow:       1.5,
	}

	hours := float64(baseSLA[c.Type]) * priorityMultiplier[c.Priority]
	deadline := c.CreatedAt.Add(time.Duration(hours) * time.Hour)
	return &deadline
}

// generateCaseNumber generates a unique case number
func generateCaseNumber(caseType CaseType) string {
	// Format: TYPE-YEAR-SEQUENCE (e.g., CW-2026-000001)
	prefix := map[CaseType]string{
		CaseTypeChildWelfare:     "CW",
		CaseTypeCriminal:         "CR",
		CaseTypeAdministrative:   "AD",
		CaseTypeHealthcare:       "HC",
		CaseTypeSocialAssistance: "SA",
		CaseTypeTax:              "TX",
		CaseTypeCivil:            "CV",
	}

	year := time.Now().Year()
	// In production, this would use a database sequence
	seq := time.Now().UnixNano() % 1000000

	return fmt.Sprintf("%s-%d-%06d", prefix[caseType], year, seq)
}
