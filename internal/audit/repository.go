package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Repository provides append-only audit log operations
type Repository struct {
	pool     *pgxpool.Pool
	mu       sync.Mutex
	lastHash string
}

// NewRepository creates a new audit repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Initialize loads the last hash from the database
func (r *Repository) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var hash string
	err := r.pool.QueryRow(ctx, `
		SELECT hash FROM audit.entries
		ORDER BY sequence DESC
		LIMIT 1
	`).Scan(&hash)

	if err != nil && !strings.Contains(err.Error(), "no rows") {
		return errors.Wrap(err, "failed to get last audit hash")
	}

	r.lastHash = hash
	return nil
}

// Append appends a new audit entry (thread-safe)
func (r *Repository) Append(ctx context.Context, entry *AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set previous hash
	entry.PrevHash = r.lastHash

	// Recalculate hash with prev_hash
	entry.Hash = entry.calculateHash()

	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		return errors.Wrap(err, "failed to marshal changes")
	}

	query := `
		INSERT INTO audit.entries (
			id, timestamp, hash, prev_hash,
			actor_type, actor_id, actor_agency_id, actor_ip, actor_device,
			action, resource_type, resource_id,
			changes, correlation_id, session_id, justification
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) RETURNING sequence`

	err = r.pool.QueryRow(ctx, query,
		entry.ID, entry.Timestamp, entry.Hash, entry.PrevHash,
		entry.ActorType, entry.ActorID, entry.ActorAgencyID, entry.ActorIP, entry.ActorDevice,
		entry.Action, entry.ResourceType, entry.ResourceID,
		changesJSON, entry.CorrelationID, entry.SessionID, entry.Justification,
	).Scan(&entry.Sequence)

	if err != nil {
		return errors.Wrap(err, "failed to append audit entry")
	}

	// Update last hash
	r.lastHash = entry.Hash

	return nil
}

// List lists audit entries with filters (read-only)
func (r *Repository) List(ctx context.Context, filter ListEntriesFilter) ([]AuditEntry, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.ActorID != nil {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argNum))
		args = append(args, *filter.ActorID)
		argNum++
	}

	if filter.ActorType != nil {
		conditions = append(conditions, fmt.Sprintf("actor_type = $%d", argNum))
		args = append(args, *filter.ActorType)
		argNum++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action LIKE $%d", argNum))
		args = append(args, filter.Action+"%")
		argNum++
	}

	if filter.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argNum))
		args = append(args, filter.ResourceType)
		argNum++
	}

	if filter.ResourceID != nil {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argNum))
		args = append(args, *filter.ResourceID)
		argNum++
	}

	if filter.StartTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, *filter.StartTime)
		argNum++
	}

	if filter.EndTime != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, *filter.EndTime)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit.entries %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count audit entries")
	}

	// Limit
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(`
		SELECT id, sequence, timestamp, hash, prev_hash,
			actor_type, actor_id, actor_agency_id, actor_ip, actor_device,
			action, resource_type, resource_id,
			changes, correlation_id, session_id, justification
		FROM audit.entries
		%s
		ORDER BY sequence DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list audit entries")
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var changesJSON []byte

		err := rows.Scan(
			&e.ID, &e.Sequence, &e.Timestamp, &e.Hash, &e.PrevHash,
			&e.ActorType, &e.ActorID, &e.ActorAgencyID, &e.ActorIP, &e.ActorDevice,
			&e.Action, &e.ResourceType, &e.ResourceID,
			&changesJSON, &e.CorrelationID, &e.SessionID, &e.Justification,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan audit entry")
		}

		if err := json.Unmarshal(changesJSON, &e.Changes); err != nil {
			e.Changes = nil
		}

		entries = append(entries, e)
	}

	return entries, total, nil
}

// FindByID finds an audit entry by ID (read-only)
func (r *Repository) FindByID(ctx context.Context, id types.ID) (*AuditEntry, error) {
	query := `
		SELECT id, sequence, timestamp, hash, prev_hash,
			actor_type, actor_id, actor_agency_id, actor_ip, actor_device,
			action, resource_type, resource_id,
			changes, correlation_id, session_id, justification
		FROM audit.entries
		WHERE id = $1`

	var e AuditEntry
	var changesJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Sequence, &e.Timestamp, &e.Hash, &e.PrevHash,
		&e.ActorType, &e.ActorID, &e.ActorAgencyID, &e.ActorIP, &e.ActorDevice,
		&e.Action, &e.ResourceType, &e.ResourceID,
		&changesJSON, &e.CorrelationID, &e.SessionID, &e.Justification,
	)

	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, errors.NotFound("audit entry", id.String())
		}
		return nil, errors.Wrap(err, "failed to find audit entry")
	}

	if err := json.Unmarshal(changesJSON, &e.Changes); err != nil {
		e.Changes = nil
	}

	return &e, nil
}

// VerifyResult contains detailed verification results
type VerifyResult struct {
	Valid           bool                `json:"valid"`
	Checked         int                 `json:"checked"`
	ContentValid    int                 `json:"content_valid"`    // Entries with valid content hash
	ContentInvalid  int                 `json:"content_invalid"`  // Entries with tampered content
	LinkageValid    int                 `json:"linkage_valid"`    // Entries with valid chain linkage
	LinkageInvalid  int                 `json:"linkage_invalid"`  // Entries with broken chain
	Violations      []string            `json:"violations,omitempty"`
	Entries         []VerifyEntryResult `json:"entries,omitempty"`
	LastCheckpoint  string              `json:"last_checkpoint,omitempty"`  // Last witnessed hash
	CheckpointValid bool                `json:"checkpoint_valid,omitempty"` // Checkpoint matches
}

// VerifyEntryResult contains verification result for a single entry
type VerifyEntryResult struct {
	ID            types.ID `json:"id"`
	Sequence      int64    `json:"sequence"`
	Hash          string   `json:"hash"`
	ComputedHash  string   `json:"computed_hash,omitempty"` // Recalculated hash
	PrevHash      string   `json:"prev_hash"`
	Valid         bool     `json:"valid"`
	ContentValid  bool     `json:"content_valid"`  // Hash matches content
	LinkageValid  bool     `json:"linkage_valid"`  // Chain link is valid
	Action        string   `json:"action"`
	ViolationType string   `json:"violation_type,omitempty"` // "content", "linkage", "both"
}

// VerifyChain verifies the integrity of the audit chain
// Performs two checks:
// 1. Content verification: Recalculates hash from entry data and compares to stored hash
// 2. Linkage verification: Verifies each entry's prev_hash matches the previous entry's hash
func (r *Repository) VerifyChain(ctx context.Context, limit int, includeDetails bool) (*VerifyResult, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Load full entries to verify content hash
	rows, err := r.pool.Query(ctx, `
		SELECT id, sequence, timestamp, hash, prev_hash,
			   actor_type, actor_id, actor_agency_id, actor_ip, actor_device,
			   action, resource_type, resource_id,
			   changes, correlation_id, session_id, justification
		FROM audit.entries
		ORDER BY sequence DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query audit entries")
	}
	defer rows.Close()

	result := &VerifyResult{
		Valid:   true,
		Entries: make([]VerifyEntryResult, 0),
	}

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var changesJSON []byte

		err := rows.Scan(
			&e.ID, &e.Sequence, &e.Timestamp, &e.Hash, &e.PrevHash,
			&e.ActorType, &e.ActorID, &e.ActorAgencyID, &e.ActorIP, &e.ActorDevice,
			&e.Action, &e.ResourceType, &e.ResourceID,
			&changesJSON, &e.CorrelationID, &e.SessionID, &e.Justification,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan audit entry")
		}

		if len(changesJSON) > 0 {
			json.Unmarshal(changesJSON, &e.Changes)
		}

		entries = append(entries, e)
	}

	// Verify each entry
	var prevStoredHash string // Hash that the NEXT entry (in time) expects

	for i, e := range entries {
		verifyEntry := VerifyEntryResult{
			ID:           e.ID,
			Sequence:     e.Sequence,
			Hash:         e.Hash,
			PrevHash:     e.PrevHash,
			Action:       e.Action,
			ContentValid: true,
			LinkageValid: true,
			Valid:        true,
		}

		// 1. Content verification: Recalculate hash and compare
		computedHash := e.ComputeHash()
		verifyEntry.ComputedHash = computedHash

		if computedHash != e.Hash {
			verifyEntry.ContentValid = false
			verifyEntry.Valid = false
			result.ContentInvalid++
			result.Valid = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("CONTENT TAMPERED: Entry %s (seq %d) - stored hash doesn't match content", e.ID, e.Sequence))
			verifyEntry.ViolationType = "content"
		} else {
			result.ContentValid++
		}

		// 2. Linkage verification: Check if this entry's hash matches what the next entry expects
		// (entries are in DESC order, so prevStoredHash is from the entry that comes AFTER this one in time)
		if i > 0 && prevStoredHash != "" && e.Hash != prevStoredHash {
			verifyEntry.LinkageValid = false
			verifyEntry.Valid = false
			result.LinkageInvalid++
			result.Valid = false
			violation := fmt.Sprintf("CHAIN BROKEN: Entry %s (seq %d) - hash doesn't match next entry's prev_hash", e.ID, e.Sequence)
			result.Violations = append(result.Violations, violation)
			if verifyEntry.ViolationType == "content" {
				verifyEntry.ViolationType = "both"
			} else {
				verifyEntry.ViolationType = "linkage"
			}
		} else if i > 0 {
			result.LinkageValid++
		}

		if includeDetails {
			result.Entries = append(result.Entries, verifyEntry)
		}

		prevStoredHash = e.PrevHash
		result.Checked++
	}

	return result, nil
}

// GetByResource gets all audit entries for a specific resource
func (r *Repository) GetByResource(ctx context.Context, resourceType string, resourceID types.ID, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	filter := ListEntriesFilter{
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
	}

	entries, _, err := r.List(ctx, filter)
	return entries, err
}
