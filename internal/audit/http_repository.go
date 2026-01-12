package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// HTTPRepository provides append-only audit log operations using EventStoreDB HTTP API
type HTTPRepository struct {
	client   *events.HTTPClient
	mu       sync.Mutex
	lastHash string
	sequence int64
}

// NewHTTPRepository creates a new HTTP-based audit repository
func NewHTTPRepository(client *events.HTTPClient) *HTTPRepository {
	return &HTTPRepository{client: client}
}

// Initialize loads the last hash and sequence from EventStoreDB via HTTP
func (r *HTTPRepository) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read the last event from the audit stream
	event, err := r.client.ReadLastEvent(ctx, AuditStreamName)
	if err != nil {
		return errors.Wrap(err, "failed to read audit stream")
	}

	if event == nil {
		// Stream doesn't exist yet
		r.lastHash = ""
		r.sequence = 0
		return nil
	}

	// Parse the last entry to get its hash
	if event.EventType == AuditEventType {
		// Handle JSON string format from embed=body
		data := event.Data
		if len(data) > 0 && data[0] == '"' {
			var dataStr string
			if err := json.Unmarshal(data, &dataStr); err == nil {
				data = json.RawMessage(dataStr)
			}
		}

		var entry AuditEntry
		if err := json.Unmarshal(data, &entry); err == nil {
			r.lastHash = entry.Hash
			r.sequence = entry.Sequence
		}
	}

	return nil
}

// Append appends a new audit entry (thread-safe)
func (r *HTTPRepository) Append(ctx context.Context, entry *AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set sequence and prev_hash
	r.sequence++
	entry.Sequence = r.sequence
	entry.PrevHash = r.lastHash

	// Calculate hash
	entry.Hash = entry.ComputeHash()

	// Create event data
	eventData := events.EventData{
		EventID:   entry.ID.String(),
		EventType: AuditEventType,
		Data:      entry,
	}

	// Append to stream
	if err := r.client.AppendToStream(ctx, AuditStreamName, eventData); err != nil {
		r.sequence-- // Rollback on failure
		return errors.Wrap(err, "failed to append audit entry")
	}

	r.lastHash = entry.Hash
	return nil
}

// FindByID finds an audit entry by ID
func (r *HTTPRepository) FindByID(ctx context.Context, id types.ID) (*AuditEntry, error) {
	entries, _, err := r.List(ctx, ListEntriesFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.ID == id {
			return entry, nil
		}
	}

	return nil, errors.NotFound("audit entry", string(id))
}

// List lists audit entries with filters
func (r *HTTPRepository) List(ctx context.Context, filter ListEntriesFilter) ([]*AuditEntry, int, error) {
	allEvents, err := r.readAllEvents(ctx)
	if err != nil {
		return nil, 0, err
	}

	var entries []*AuditEntry
	total := 0

	for i := len(allEvents) - 1; i >= 0; i-- { // Reverse order (newest first)
		recorded := allEvents[i]
		if recorded.EventType != AuditEventType {
			continue
		}

		// Handle JSON string format from embed=body (data may be wrapped in quotes)
		data := recorded.Data
		if len(data) > 0 && data[0] == '"' {
			var dataStr string
			if err := json.Unmarshal(data, &dataStr); err != nil {
				continue
			}
			data = json.RawMessage(dataStr)
		}

		var entry AuditEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		// Apply filters
		if filter.ActorID != nil && entry.ActorID != *filter.ActorID {
			continue
		}
		if filter.ActorType != nil && entry.ActorType != *filter.ActorType {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.ResourceType != "" && entry.ResourceType != filter.ResourceType {
			continue
		}
		if filter.ResourceID != nil && (entry.ResourceID == nil || *entry.ResourceID != *filter.ResourceID) {
			continue
		}
		if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
			continue
		}

		total++

		// Apply offset and limit
		if filter.Offset > 0 && total <= filter.Offset {
			continue
		}
		if filter.Limit > 0 && len(entries) >= filter.Limit {
			continue
		}

		entries = append(entries, &entry)
	}

	return entries, total, nil
}

// readAllEvents reads all events from the audit stream
func (r *HTTPRepository) readAllEvents(ctx context.Context) ([]events.RecordedEvent, error) {
	var allEvents []events.RecordedEvent
	var start int64 = 0

	for {
		batch, err := r.client.ReadStream(ctx, AuditStreamName, events.ReadStreamOptions{
			Direction: "forward",
			Start:     start,
			Count:     100,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to read audit stream")
		}

		if len(batch) == 0 {
			break
		}

		allEvents = append(allEvents, batch...)
		start = batch[len(batch)-1].EventNumber + 1

		if len(allEvents) > 10000 {
			break
		}
	}

	return allEvents, nil
}

// GetByResource gets audit entries for a specific resource
func (r *HTTPRepository) GetByResource(ctx context.Context, resourceType string, resourceID types.ID, limit int) ([]*AuditEntry, error) {
	filter := ListEntriesFilter{
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
	}
	entries, _, err := r.List(ctx, filter)
	return entries, err
}

// VerifyChain verifies the integrity of the audit chain
func (r *HTTPRepository) VerifyChain(ctx context.Context, limit int, includeDetails bool) (*VerifyResult, error) {
	allEvents, err := r.readAllEvents(ctx)
	if err != nil {
		return nil, err
	}

	result := &VerifyResult{
		Valid:          true,
		Checked:        0,
		ContentValid:   0,
		ContentInvalid: 0,
		LinkageValid:   0,
		LinkageInvalid: 0,
	}

	var prevHash string
	count := 0

	for _, recorded := range allEvents {
		if recorded.EventType != AuditEventType {
			continue
		}

		// Handle JSON string format from embed=body
		data := recorded.Data
		if len(data) > 0 && data[0] == '"' {
			var dataStr string
			if err := json.Unmarshal(data, &dataStr); err != nil {
				continue
			}
			data = json.RawMessage(dataStr)
		}

		var entry AuditEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		result.Checked++
		count++

		// Verify content hash
		computed := entry.ComputeHash()
		if computed == entry.Hash {
			result.ContentValid++
		} else {
			result.ContentInvalid++
			result.Valid = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Entry %d: content hash mismatch", entry.Sequence))
		}

		// Verify chain linkage
		if entry.PrevHash == prevHash {
			result.LinkageValid++
		} else {
			result.LinkageInvalid++
			result.Valid = false
			result.Violations = append(result.Violations,
				fmt.Sprintf("Entry %d: chain linkage broken", entry.Sequence))
		}

		prevHash = entry.Hash

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, nil
}

// GetLastHash returns the last hash in the chain
func (r *HTTPRepository) GetLastHash() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastHash
}

// GetSequence returns the current sequence number
func (r *HTTPRepository) GetSequence() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sequence
}

// Count returns the total number of audit entries
func (r *HTTPRepository) Count(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int(r.sequence), nil
}

// SaveCheckpoint saves a new checkpoint
func (r *HTTPRepository) SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	eventData := events.EventData{
		EventID:   checkpoint.ID.String(),
		EventType: CheckpointEventType,
		Data:      checkpoint,
	}

	return r.client.AppendToStream(ctx, "$audit-checkpoints", eventData)
}

// GetLatestCheckpoint returns the most recent checkpoint
func (r *HTTPRepository) GetLatestCheckpoint(ctx context.Context) (*Checkpoint, error) {
	event, err := r.client.ReadLastEvent(ctx, "$audit-checkpoints")
	if err != nil {
		return nil, errors.Wrap(err, "failed to read checkpoint stream")
	}

	if event == nil || event.EventType != CheckpointEventType {
		return nil, nil
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(event.Data, &checkpoint); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal checkpoint")
	}

	return &checkpoint, nil
}

// ListCheckpoints lists checkpoints
func (r *HTTPRepository) ListCheckpoints(ctx context.Context, limit int) ([]Checkpoint, error) {
	evts, err := r.client.ReadStream(ctx, "$audit-checkpoints", events.ReadStreamOptions{
		Direction: "backward",
		Start:     -1,
		Count:     limit,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to read checkpoints")
	}

	var checkpoints []Checkpoint
	for _, e := range evts {
		if e.EventType == CheckpointEventType {
			var cp Checkpoint
			if err := json.Unmarshal(e.Data, &cp); err == nil {
				checkpoints = append(checkpoints, cp)
			}
		}
	}

	return checkpoints, nil
}

// GetCheckpoint gets a specific checkpoint by ID
func (r *HTTPRepository) GetCheckpoint(ctx context.Context, id types.ID) (*Checkpoint, error) {
	checkpoints, err := r.ListCheckpoints(ctx, 1000)
	if err != nil {
		return nil, err
	}

	for _, cp := range checkpoints {
		if cp.ID == id {
			return &cp, nil
		}
	}

	return nil, errors.NotFound("checkpoint", string(id))
}

// Ensure HTTPRepository implements AuditRepository
var _ AuditRepository = (*HTTPRepository)(nil)
