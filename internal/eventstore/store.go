// Package eventstore provides event sourcing infrastructure.
// It defines interfaces for event storage and streaming.
// Implementation is provided by the kurrentdb package using KurrentDB.
package eventstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Common errors
var (
	ErrConcurrencyConflict = errors.New("concurrency conflict: aggregate version mismatch")
	ErrEventNotFound       = errors.New("event not found")
	ErrAggregateNotFound   = errors.New("aggregate not found")
	ErrInvalidEvent        = errors.New("invalid event data")
)

// Event represents a domain event stored in the event store.
type Event struct {
	ID            types.ID       `json:"id"`
	AggregateID   types.ID       `json:"aggregate_id"`
	AggregateType string         `json:"aggregate_type"`
	EventType     string         `json:"event_type"`
	Version       int            `json:"version"`
	Timestamp     time.Time      `json:"timestamp"`
	Data          map[string]any `json:"data"`
	Metadata      EventMetadata  `json:"metadata"`
}

// EventMetadata contains contextual information about an event.
type EventMetadata struct {
	CorrelationID string   `json:"correlation_id"`
	CausationID   string   `json:"causation_id,omitempty"`
	ActorID       types.ID `json:"actor_id"`
	ActorAgency   types.ID `json:"actor_agency,omitempty"`
	ActorType     string   `json:"actor_type"` // citizen, worker, system
	Source        string   `json:"source"`
}

// NewEvent creates a new event with generated ID and timestamp.
func NewEvent(aggregateID types.ID, aggregateType, eventType string, data map[string]any) *Event {
	return &Event{
		ID:            types.NewID(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		Timestamp:     time.Now().UTC(),
		Data:          data,
		Metadata:      EventMetadata{},
	}
}

// WithMetadata adds metadata to the event.
func (e *Event) WithMetadata(meta EventMetadata) *Event {
	e.Metadata = meta
	return e
}

// WithCorrelation sets the correlation ID for event tracing.
func (e *Event) WithCorrelation(correlationID string) *Event {
	e.Metadata.CorrelationID = correlationID
	return e
}

// WithActor sets the actor information.
func (e *Event) WithActor(actorID types.ID, actorType string, agencyID types.ID) *Event {
	e.Metadata.ActorID = actorID
	e.Metadata.ActorType = actorType
	e.Metadata.ActorAgency = agencyID
	return e
}

// Hash returns SHA-256 hash of the event for integrity verification.
func (e *Event) Hash() string {
	data, _ := json.Marshal(e)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// EventHandler is a function that handles events.
type EventHandler func(ctx context.Context, event *Event) error

// EventStore defines the interface for event storage operations.
type EventStore interface {
	// Append stores new events for an aggregate with optimistic concurrency.
	// expectedVersion should be the current version of the aggregate (0 for new).
	Append(ctx context.Context, events []*Event, expectedVersion int) error

	// Load retrieves all events for an aggregate in version order.
	Load(ctx context.Context, aggregateID types.ID) ([]*Event, error)

	// LoadFrom retrieves events for an aggregate starting from a version.
	LoadFrom(ctx context.Context, aggregateID types.ID, fromVersion int) ([]*Event, error)

	// LoadByType retrieves events of a specific type within a time range.
	LoadByType(ctx context.Context, eventType string, from, to time.Time, limit int) ([]*Event, error)

	// GetAggregateVersion returns the current version of an aggregate.
	GetAggregateVersion(ctx context.Context, aggregateID types.ID) (int, error)
}

// EventPublisher publishes events to subscribers.
type EventPublisher interface {
	// Publish sends events to the message bus for real-time subscribers.
	Publish(ctx context.Context, events []*Event) error
}

// EventSubscriber subscribes to events.
type EventSubscriber interface {
	// Subscribe registers a handler for specific event types.
	// Empty eventTypes means subscribe to all events.
	Subscribe(ctx context.Context, eventTypes []string, handler EventHandler) error
}

// Snapshot represents a point-in-time state of an aggregate.
type Snapshot struct {
	AggregateID   types.ID       `json:"aggregate_id"`
	AggregateType string         `json:"aggregate_type"`
	Version       int            `json:"version"`
	State         map[string]any `json:"state"`
	CreatedAt     time.Time      `json:"created_at"`
}

// SnapshotStore manages aggregate snapshots for faster state rebuilding.
type SnapshotStore interface {
	// Save stores a snapshot.
	Save(ctx context.Context, snapshot *Snapshot) error

	// Load retrieves the latest snapshot for an aggregate.
	Load(ctx context.Context, aggregateID types.ID) (*Snapshot, error)

	// Delete removes a snapshot.
	Delete(ctx context.Context, aggregateID types.ID) error
}

// Projection tracks the processing state of a read model projection.
type Projection struct {
	Name                   string    `json:"name"`
	LastProcessedSequence  int64     `json:"last_processed_sequence"`
	LastProcessedEventID   types.ID  `json:"last_processed_event_id,omitempty"`
	LastProcessedAt        time.Time `json:"last_processed_at"`
	Status                 string    `json:"status"` // running, paused, failed
	ErrorMessage           string    `json:"error_message,omitempty"`
}

// ProjectionStore manages projection state.
type ProjectionStore interface {
	// Get retrieves projection state.
	Get(ctx context.Context, name string) (*Projection, error)

	// Update saves projection state.
	Update(ctx context.Context, projection *Projection) error

	// List returns all projections.
	List(ctx context.Context) ([]*Projection, error)
}

// AggregateRoot is the base interface for aggregates in event sourcing.
type AggregateRoot interface {
	// AggregateID returns the unique identifier.
	AggregateID() types.ID

	// AggregateType returns the type name.
	AggregateType() string

	// Version returns the current version.
	Version() int

	// ApplyEvent applies an event to update state.
	ApplyEvent(event *Event) error

	// GetUncommittedEvents returns events not yet persisted.
	GetUncommittedEvents() []*Event

	// ClearUncommittedEvents clears the uncommitted events.
	ClearUncommittedEvents()
}

// BaseAggregate provides common functionality for aggregates.
type BaseAggregate struct {
	id                types.ID
	aggregateType     string
	version           int
	uncommittedEvents []*Event
}

// NewBaseAggregate creates a new base aggregate.
func NewBaseAggregate(aggregateType string) *BaseAggregate {
	return &BaseAggregate{
		id:                types.ID(uuid.New().String()),
		aggregateType:     aggregateType,
		version:           0,
		uncommittedEvents: make([]*Event, 0),
	}
}

// NewBaseAggregateWithID creates a base aggregate with specific ID.
func NewBaseAggregateWithID(id types.ID, aggregateType string) *BaseAggregate {
	return &BaseAggregate{
		id:                id,
		aggregateType:     aggregateType,
		version:           0,
		uncommittedEvents: make([]*Event, 0),
	}
}

func (a *BaseAggregate) AggregateID() types.ID   { return a.id }
func (a *BaseAggregate) AggregateType() string   { return a.aggregateType }
func (a *BaseAggregate) Version() int            { return a.version }
func (a *BaseAggregate) SetVersion(v int)        { a.version = v }
func (a *BaseAggregate) SetID(id types.ID)       { a.id = id }

func (a *BaseAggregate) GetUncommittedEvents() []*Event {
	return a.uncommittedEvents
}

func (a *BaseAggregate) ClearUncommittedEvents() {
	a.uncommittedEvents = make([]*Event, 0)
}

// RaiseEvent creates and tracks a new domain event.
func (a *BaseAggregate) RaiseEvent(eventType string, data map[string]any) *Event {
	event := NewEvent(a.id, a.aggregateType, eventType, data)
	event.Version = a.version + len(a.uncommittedEvents) + 1
	a.uncommittedEvents = append(a.uncommittedEvents, event)
	return event
}
