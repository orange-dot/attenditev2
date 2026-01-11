package audit

import (
	"context"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// AuditRepository defines the interface for audit storage operations.
// This allows swapping between PostgreSQL and KurrentDB implementations.
type AuditRepository interface {
	// Initialize loads initial state (last hash, sequence)
	Initialize(ctx context.Context) error

	// Append appends a new audit entry
	Append(ctx context.Context, entry *AuditEntry) error

	// FindByID finds an audit entry by ID
	FindByID(ctx context.Context, id types.ID) (*AuditEntry, error)

	// List lists audit entries with filters
	List(ctx context.Context, filter ListEntriesFilter) ([]*AuditEntry, int, error)

	// GetByResource gets audit entries for a specific resource
	GetByResource(ctx context.Context, resourceType string, resourceID types.ID, limit int) ([]*AuditEntry, error)

	// VerifyChain verifies the integrity of the audit chain
	VerifyChain(ctx context.Context, limit int, includeDetails bool) (*VerifyResult, error)

	// GetLastHash returns the last hash in the chain
	GetLastHash() string

	// GetSequence returns the current sequence number
	GetSequence() int64

	// Count returns the total number of audit entries
	Count(ctx context.Context) (int, error)

	// Checkpoint operations
	SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
	GetLatestCheckpoint(ctx context.Context) (*Checkpoint, error)
	ListCheckpoints(ctx context.Context, limit int) ([]Checkpoint, error)
	GetCheckpoint(ctx context.Context, id types.ID) (*Checkpoint, error)
}

// Ensure implementations satisfy the interface
var _ AuditRepository = (*KurrentDBRepository)(nil)
