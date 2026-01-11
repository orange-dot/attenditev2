package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/google/uuid"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

const (
	// AuditStreamName is the stream where all audit entries are stored
	AuditStreamName = "$audit"
	// AuditEventType is the event type for audit entries
	AuditEventType = "AuditEntry"
	// CheckpointEventType is the event type for checkpoints
	CheckpointEventType = "AuditCheckpoint"
)

// KurrentDBRepository provides append-only audit log operations using KurrentDB.
// KurrentDB is inherently append-only - events cannot be modified or deleted.
type KurrentDBRepository struct {
	client   *esdb.Client
	mu       sync.Mutex
	lastHash string
	sequence int64
}

// NewKurrentDBRepository creates a new KurrentDB-based audit repository
func NewKurrentDBRepository(client *esdb.Client) *KurrentDBRepository {
	return &KurrentDBRepository{client: client}
}

// Initialize loads the last hash and sequence from KurrentDB
func (r *KurrentDBRepository) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read the last event from the audit stream
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, 1)
	if err != nil {
		// Stream doesn't exist yet - that's OK
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				r.lastHash = ""
				r.sequence = 0
				return nil
			}
		}
		return errors.Wrap(err, "failed to read audit stream")
	}
	defer stream.Close()

	event, err := stream.Recv()
	if err != nil {
		// No events yet
		r.lastHash = ""
		r.sequence = 0
		return nil
	}

	// Parse the last entry to get its hash
	if event.Event != nil && event.Event.EventType == AuditEventType {
		var entry AuditEntry
		if err := json.Unmarshal(event.Event.Data, &entry); err == nil {
			r.lastHash = entry.Hash
			r.sequence = entry.Sequence
		}
	}

	return nil
}

// Append appends a new audit entry (thread-safe)
func (r *KurrentDBRepository) Append(ctx context.Context, entry *AuditEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set sequence and prev_hash
	r.sequence++
	entry.Sequence = r.sequence
	entry.PrevHash = r.lastHash

	// Calculate hash
	entry.Hash = entry.ComputeHash()

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		return errors.Wrap(err, "failed to marshal audit entry")
	}

	// Create event
	eventID := uuid.New()
	eventData := esdb.EventData{
		EventID:     eventID,
		EventType:   AuditEventType,
		ContentType: esdb.ContentTypeJson,
		Data:        data,
		Metadata: []byte(fmt.Sprintf(`{"sequence":%d,"hash":"%s"}`,
			entry.Sequence, entry.Hash)),
	}

	// Append to stream
	_, err = r.client.AppendToStream(ctx, AuditStreamName, esdb.AppendToStreamOptions{}, eventData)
	if err != nil {
		return errors.Wrap(err, "failed to append audit entry")
	}

	// Update last hash
	r.lastHash = entry.Hash

	return nil
}

// FindByID finds an audit entry by ID
func (r *KurrentDBRepository) FindByID(ctx context.Context, id types.ID) (*AuditEntry, error) {
	// Read all events and find by ID (not efficient for large streams)
	// In production, you'd use a projection or secondary index
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Forwards,
		From:      esdb.Start{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, 10000)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read audit stream")
	}
	defer stream.Close()

	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Event != nil && event.Event.EventType == AuditEventType {
			var entry AuditEntry
			if err := json.Unmarshal(event.Event.Data, &entry); err == nil {
				if entry.ID == id {
					return &entry, nil
				}
			}
		}
	}

	return nil, errors.NotFound("audit entry", string(id))
}

// List lists audit entries with filters
func (r *KurrentDBRepository) List(ctx context.Context, filter ListEntriesFilter) ([]*AuditEntry, int, error) {
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}

	maxEvents := uint64(1000)
	if filter.Limit > 0 {
		maxEvents = uint64(filter.Limit + filter.Offset + 100) // Read extra to account for filtering
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, maxEvents)
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return []*AuditEntry{}, 0, nil
			}
		}
		return nil, 0, errors.Wrap(err, "failed to read audit stream")
	}
	defer stream.Close()

	var entries []*AuditEntry
	total := 0

	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Event != nil && event.Event.EventType == AuditEventType {
			var entry AuditEntry
			if err := json.Unmarshal(event.Event.Data, &entry); err == nil {
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
		}
	}

	return entries, total, nil
}

// GetByResource gets audit entries for a specific resource
func (r *KurrentDBRepository) GetByResource(ctx context.Context, resourceType string, resourceID types.ID, limit int) ([]*AuditEntry, error) {
	filter := ListEntriesFilter{
		ResourceType: resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
	}
	entries, _, err := r.List(ctx, filter)
	return entries, err
}

// VerifyChain verifies the integrity of the audit chain
func (r *KurrentDBRepository) VerifyChain(ctx context.Context, limit int, includeDetails bool) (*VerifyResult, error) {
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, uint64(limit))
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return &VerifyResult{Valid: true, Checked: 0}, nil
			}
		}
		return nil, errors.Wrap(err, "failed to read audit stream")
	}
	defer stream.Close()

	result := &VerifyResult{
		Valid:          true,
		Checked:        0,
		ContentValid:   0,
		ContentInvalid: 0,
		LinkageValid:   0,
		LinkageInvalid: 0,
	}

	var entries []*AuditEntry
	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Event != nil && event.Event.EventType == AuditEventType {
			var entry AuditEntry
			if err := json.Unmarshal(event.Event.Data, &entry); err == nil {
				entries = append(entries, &entry)
			}
		}
	}

	result.Checked = len(entries)

	// Verify each entry (entries are in reverse order)
	for i, entry := range entries {
		// 1. Content verification: Recalculate hash
		computedHash := entry.ComputeHash()
		contentValid := computedHash == entry.Hash

		if contentValid {
			result.ContentValid++
		} else {
			result.Valid = false
			result.ContentInvalid++
			result.Violations = append(result.Violations,
				fmt.Sprintf("CONTENT TAMPERED: Entry %d hash mismatch (stored: %s, computed: %s)",
					entry.Sequence, entry.Hash[:16], computedHash[:16]))
		}

		// 2. Linkage verification: Check prev_hash matches previous entry's hash
		linkageValid := true
		if i < len(entries)-1 {
			prevEntry := entries[i+1]
			if entry.PrevHash != prevEntry.Hash {
				linkageValid = false
				result.Valid = false
				result.LinkageInvalid++
				result.Violations = append(result.Violations,
					fmt.Sprintf("CHAIN BROKEN: Entry %d prev_hash doesn't match entry %d hash",
						entry.Sequence, prevEntry.Sequence))
			} else {
				result.LinkageValid++
			}
		} else {
			result.LinkageValid++ // First entry has no prev to check
		}

		if includeDetails {
			result.Entries = append(result.Entries, VerifyEntryResult{
				ID:           entry.ID,
				Sequence:     entry.Sequence,
				Hash:         entry.Hash,
				ComputedHash: computedHash,
				PrevHash:     entry.PrevHash,
				Valid:        contentValid && linkageValid,
				ContentValid: contentValid,
				LinkageValid: linkageValid,
				Action:       entry.Action,
			})
		}
	}

	return result, nil
}

// GetLastHash returns the last hash in the chain
func (r *KurrentDBRepository) GetLastHash() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastHash
}

// GetSequence returns the current sequence number
func (r *KurrentDBRepository) GetSequence() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sequence
}

// Count returns the total number of audit entries
func (r *KurrentDBRepository) Count(ctx context.Context) (int, error) {
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Forwards,
		From:      esdb.Start{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName, opts, 100000)
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return 0, nil
			}
		}
		return 0, errors.Wrap(err, "failed to read audit stream")
	}
	defer stream.Close()

	count := 0
	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}
		if event.Event != nil && event.Event.EventType == AuditEventType {
			count++
		}
	}

	return count, nil
}

// SaveCheckpoint saves a checkpoint to KurrentDB
func (r *KurrentDBRepository) SaveCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return errors.Wrap(err, "failed to marshal checkpoint")
	}

	eventID := uuid.New()
	eventData := esdb.EventData{
		EventID:     eventID,
		EventType:   CheckpointEventType,
		ContentType: esdb.ContentTypeJson,
		Data:        data,
	}

	_, err = r.client.AppendToStream(ctx, AuditStreamName+"-checkpoints", esdb.AppendToStreamOptions{}, eventData)
	if err != nil {
		return errors.Wrap(err, "failed to save checkpoint")
	}

	return nil
}

// GetLatestCheckpoint returns the most recent checkpoint
func (r *KurrentDBRepository) GetLatestCheckpoint(ctx context.Context) (*Checkpoint, error) {
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName+"-checkpoints", opts, 1)
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return nil, nil
			}
		}
		return nil, errors.Wrap(err, "failed to read checkpoints stream")
	}
	defer stream.Close()

	event, err := stream.Recv()
	if err != nil {
		return nil, nil
	}

	if event.Event != nil && event.Event.EventType == CheckpointEventType {
		var checkpoint Checkpoint
		if err := json.Unmarshal(event.Event.Data, &checkpoint); err == nil {
			return &checkpoint, nil
		}
	}

	return nil, nil
}

// ListCheckpoints returns all checkpoints
func (r *KurrentDBRepository) ListCheckpoints(ctx context.Context, limit int) ([]Checkpoint, error) {
	opts := esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}

	stream, err := r.client.ReadStream(ctx, AuditStreamName+"-checkpoints", opts, uint64(limit))
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return []Checkpoint{}, nil
			}
		}
		return nil, errors.Wrap(err, "failed to read checkpoints stream")
	}
	defer stream.Close()

	var checkpoints []Checkpoint
	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Event != nil && event.Event.EventType == CheckpointEventType {
			var checkpoint Checkpoint
			if err := json.Unmarshal(event.Event.Data, &checkpoint); err == nil {
				checkpoints = append(checkpoints, checkpoint)
			}
		}
	}

	return checkpoints, nil
}

// GetCheckpoint returns a checkpoint by ID
func (r *KurrentDBRepository) GetCheckpoint(ctx context.Context, id types.ID) (*Checkpoint, error) {
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

// computeCheckpointHash computes a hash for checkpoint verification
func computeCheckpointHash(lastHash string, sequence int64, count int, timestamp time.Time) string {
	data := fmt.Sprintf("%s:%d:%d:%d", lastHash, sequence, count, timestamp.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
