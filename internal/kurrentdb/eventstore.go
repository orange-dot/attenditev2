package kurrentdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/serbia-gov/platform/internal/eventstore"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// EventStore implements the eventstore.EventStore interface using KurrentDB.
type EventStore struct {
	client *Client
}

// NewEventStore creates a new KurrentDB-backed event store.
func NewEventStore(client *Client) *EventStore {
	return &EventStore{client: client}
}

// streamName returns the stream name for an aggregate.
func streamName(aggregateType string, aggregateID types.ID) string {
	return fmt.Sprintf("%s-%s", aggregateType, aggregateID)
}

// Append stores new events for an aggregate with optimistic concurrency.
func (s *EventStore) Append(ctx context.Context, events []*eventstore.Event, expectedVersion int) error {
	if len(events) == 0 {
		return nil
	}

	// All events should be for the same aggregate
	aggregateID := events[0].AggregateID
	aggregateType := events[0].AggregateType
	stream := streamName(aggregateType, aggregateID)

	// Convert events to EventStore format
	esdbEvents := make([]esdb.EventData, len(events))
	for i, event := range events {
		data, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}

		metadata, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal event metadata: %w", err)
		}

		esdbEvents[i] = esdb.EventData{
			EventType:   event.EventType,
			ContentType: esdb.ContentTypeJson,
			Data:        data,
			Metadata:    metadata,
			EventID:     toUUID(event.ID),
		}
	}

	// Set expected revision for optimistic concurrency
	var options esdb.AppendToStreamOptions
	if expectedVersion == 0 {
		options.ExpectedRevision = esdb.NoStream{}
	} else {
		options.ExpectedRevision = esdb.Revision(uint64(expectedVersion - 1))
	}

	_, err := s.client.DB().AppendToStream(ctx, stream, options, esdbEvents...)
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeWrongExpectedVersion {
				return eventstore.ErrConcurrencyConflict
			}
		}
		return fmt.Errorf("failed to append events: %w", err)
	}

	return nil
}

// Load retrieves all events for an aggregate in version order.
func (s *EventStore) Load(ctx context.Context, aggregateID types.ID) ([]*eventstore.Event, error) {
	return s.LoadFrom(ctx, aggregateID, 0)
}

// LoadFrom retrieves events for an aggregate starting from a version.
func (s *EventStore) LoadFrom(ctx context.Context, aggregateID types.ID, fromVersion int) ([]*eventstore.Event, error) {
	// We need to determine the aggregate type from the stream
	// For now, we'll use a category projection to find the stream
	// In practice, you'd typically know the aggregate type when loading

	// Try to find streams that match this aggregate ID
	streams, err := s.findStreamsForAggregate(ctx, aggregateID)
	if err != nil {
		return nil, err
	}

	if len(streams) == 0 {
		return nil, nil // No events for this aggregate
	}

	// Use the first matching stream
	stream := streams[0]

	var startFrom esdb.StreamPosition
	if fromVersion > 0 {
		startFrom = esdb.Revision(uint64(fromVersion - 1))
	} else {
		startFrom = esdb.Start{}
	}

	readStream, err := s.client.DB().ReadStream(ctx, stream, esdb.ReadStreamOptions{
		From:      startFrom,
		Direction: esdb.Forwards,
	}, 1000) // Read up to 1000 events

	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}
	defer readStream.Close()

	var events []*eventstore.Event
	for {
		resolvedEvent, err := readStream.Recv()
		if err != nil {
			break // End of stream
		}

		event, err := s.resolvedEventToEvent(resolvedEvent)
		if err != nil {
			return nil, fmt.Errorf("failed to convert event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// LoadByType retrieves events of a specific type within a time range.
func (s *EventStore) LoadByType(ctx context.Context, eventType string, from, to time.Time, limit int) ([]*eventstore.Event, error) {
	// Use the $et-{eventType} category projection
	categoryStream := fmt.Sprintf("$et-%s", eventType)

	readStream, err := s.client.DB().ReadStream(ctx, categoryStream, esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, uint64(limit))

	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to read category stream: %w", err)
	}
	defer readStream.Close()

	var events []*eventstore.Event
	for {
		resolvedEvent, err := readStream.Recv()
		if err != nil {
			break // End of stream
		}

		event, err := s.resolvedEventToEvent(resolvedEvent)
		if err != nil {
			continue
		}

		// Filter by time range
		if event.Timestamp.Before(from) || event.Timestamp.After(to) {
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// GetAggregateVersion returns the current version of an aggregate.
func (s *EventStore) GetAggregateVersion(ctx context.Context, aggregateID types.ID) (int, error) {
	streams, err := s.findStreamsForAggregate(ctx, aggregateID)
	if err != nil {
		return 0, err
	}

	if len(streams) == 0 {
		return 0, nil
	}

	stream := streams[0]
	pos, err := s.client.GetStreamLastPosition(ctx, stream)
	if err != nil {
		return 0, err
	}

	return int(pos) + 1, nil
}

// findStreamsForAggregate finds streams that contain events for the given aggregate ID.
func (s *EventStore) findStreamsForAggregate(ctx context.Context, aggregateID types.ID) ([]string, error) {
	// Read from the $streams system stream to find matching streams
	// This is a simplified approach - in production you might want to use projections

	readStream, err := s.client.DB().ReadStream(ctx, "$streams", esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, 1000)

	if err != nil {
		return nil, fmt.Errorf("failed to read $streams: %w", err)
	}
	defer readStream.Close()

	suffix := "-" + string(aggregateID)
	var matches []string

	for {
		event, err := readStream.Recv()
		if err != nil {
			break
		}

		streamName := string(event.Event.Data)
		if len(streamName) > len(suffix) && streamName[len(streamName)-len(suffix):] == suffix {
			matches = append(matches, streamName)
		}
	}

	return matches, nil
}

// resolvedEventToEvent converts a KurrentDB resolved event to our event type.
func (s *EventStore) resolvedEventToEvent(resolved *esdb.ResolvedEvent) (*eventstore.Event, error) {
	event := resolved.Event

	var data map[string]any
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	var metadata eventstore.EventMetadata
	if len(event.UserMetadata) > 0 {
		if err := json.Unmarshal(event.UserMetadata, &metadata); err != nil {
			// Metadata parsing is optional
			metadata = eventstore.EventMetadata{}
		}
	}

	// Parse stream name to get aggregate type and ID
	// Stream format: {aggregateType}-{aggregateID}
	streamName := event.StreamID
	aggregateType, aggregateID := parseStreamName(streamName)

	return &eventstore.Event{
		ID:            types.ID(event.EventID.String()),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     event.EventType,
		Version:       int(event.EventNumber) + 1,
		Timestamp:     event.CreatedDate,
		Data:          data,
		Metadata:      metadata,
	}, nil
}

// parseStreamName extracts aggregate type and ID from stream name.
func parseStreamName(stream string) (string, types.ID) {
	// Find the last hyphen followed by a UUID-like pattern
	for i := len(stream) - 1; i >= 0; i-- {
		if stream[i] == '-' && i > 0 {
			// Check if the rest looks like a UUID (36 chars with hyphens)
			remaining := stream[i+1:]
			if len(remaining) >= 36 {
				return stream[:i], types.ID(remaining)
			}
		}
	}
	return stream, ""
}
