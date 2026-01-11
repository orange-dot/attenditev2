package domain

import (
	"context"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// Repository defines the interface for case persistence
type Repository interface {
	// Case operations
	Save(ctx context.Context, c *Case) error
	FindByID(ctx context.Context, id types.ID) (*Case, error)
	FindByCaseNumber(ctx context.Context, caseNumber string) (*Case, error)
	Update(ctx context.Context, c *Case) error
	Delete(ctx context.Context, id types.ID) error

	// Query operations
	List(ctx context.Context, filter ListFilter) ([]Case, int, error)
	FindByAgency(ctx context.Context, agencyID types.ID, filter ListFilter) ([]Case, int, error)
	FindByWorker(ctx context.Context, workerID types.ID, filter ListFilter) ([]Case, int, error)
	FindSharedWith(ctx context.Context, agencyID types.ID, filter ListFilter) ([]Case, int, error)

	// Participant operations
	AddParticipant(ctx context.Context, caseID types.ID, p *Participant) error
	RemoveParticipant(ctx context.Context, caseID, participantID types.ID) error

	// Assignment operations
	AddAssignment(ctx context.Context, caseID types.ID, a *Assignment) error
	UpdateAssignment(ctx context.Context, a *Assignment) error

	// Event operations
	AddEvent(ctx context.Context, caseID types.ID, e *CaseEvent) error
	GetEvents(ctx context.Context, caseID types.ID, limit, offset int) ([]CaseEvent, error)
}

// ListFilter defines filters for listing cases
type ListFilter struct {
	Type       *CaseType   `json:"type,omitempty"`
	Status     *CaseStatus `json:"status,omitempty"`
	Priority   *Priority   `json:"priority,omitempty"`
	Search     string      `json:"search,omitempty"`
	SLAStatus  *SLAStatus  `json:"sla_status,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	OrderBy    string      `json:"order_by,omitempty"`
	OrderDesc  bool        `json:"order_desc,omitempty"`
}
