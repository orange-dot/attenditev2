package kurrentdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/serbia-gov/platform/internal/eventstore"
)

// Subscriber implements the eventstore.EventSubscriber interface using KurrentDB.
type Subscriber struct {
	client *Client
}

// NewSubscriber creates a new KurrentDB-backed event subscriber.
func NewSubscriber(client *Client) *Subscriber {
	return &Subscriber{client: client}
}

// Subscribe registers a handler for specific event types.
// Empty eventTypes means subscribe to all events.
func (s *Subscriber) Subscribe(ctx context.Context, eventTypes []string, handler eventstore.EventHandler) error {
	// Determine which stream to subscribe to
	var stream string
	if len(eventTypes) == 0 {
		// Subscribe to all events using $all
		stream = "$all"
	} else if len(eventTypes) == 1 {
		// Subscribe to a specific event type category
		stream = fmt.Sprintf("$et-%s", eventTypes[0])
	} else {
		// For multiple event types, create separate subscriptions
		for _, et := range eventTypes {
			if err := s.Subscribe(ctx, []string{et}, handler); err != nil {
				return err
			}
		}
		return nil
	}

	// Create a persistent subscription
	groupName := "subscriber-all"
	if len(eventTypes) > 0 {
		groupName = fmt.Sprintf("subscriber-%s", eventTypes[0])
	}

	// Try to create the subscription (ignore if already exists)
	settings := esdb.SubscriptionSettingsDefault()
	settings.ResolveLinkTos = true

	if stream == "$all" {
		err := s.client.DB().CreatePersistentSubscriptionToAll(ctx, groupName, esdb.PersistentAllSubscriptionOptions{
			Settings:  &settings,
			StartFrom: esdb.End{},
		})
		if err != nil {
			// Ignore "already exists" errors
			if esdbErr, ok := esdb.FromError(err); ok {
				if esdbErr.Code() != esdb.ErrorCodeResourceAlreadyExists {
					return fmt.Errorf("failed to create persistent subscription: %w", err)
				}
			}
		}

		return s.subscribeToPersistentAll(ctx, groupName, handler)
	}

	err := s.client.DB().CreatePersistentSubscription(ctx, stream, groupName, esdb.PersistentStreamSubscriptionOptions{
		Settings:  &settings,
		StartFrom: esdb.End{},
	})
	if err != nil {
		// Ignore "already exists" errors
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() != esdb.ErrorCodeResourceAlreadyExists {
				return fmt.Errorf("failed to create persistent subscription: %w", err)
			}
		}
	}

	return s.subscribeToPersistentStream(ctx, stream, groupName, handler)
}

// subscribeToPersistentStream subscribes to a persistent subscription on a stream.
func (s *Subscriber) subscribeToPersistentStream(ctx context.Context, stream, groupName string, handler eventstore.EventHandler) error {
	sub, err := s.client.DB().SubscribeToPersistentSubscription(ctx, stream, groupName, esdb.SubscribeToPersistentSubscriptionOptions{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to persistent subscription: %w", err)
	}

	go s.handleSubscription(ctx, sub, handler)
	return nil
}

// subscribeToPersistentAll subscribes to a persistent subscription on $all.
func (s *Subscriber) subscribeToPersistentAll(ctx context.Context, groupName string, handler eventstore.EventHandler) error {
	sub, err := s.client.DB().SubscribeToPersistentSubscriptionToAll(ctx, groupName, esdb.SubscribeToPersistentSubscriptionOptions{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to persistent subscription: %w", err)
	}

	go s.handleSubscription(ctx, sub, handler)
	return nil
}

// handleSubscription processes events from a subscription.
func (s *Subscriber) handleSubscription(ctx context.Context, sub *esdb.PersistentSubscription, handler eventstore.EventHandler) {
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			subEvent := sub.Recv()
			if subEvent.EventAppeared == nil {
				if subEvent.SubscriptionDropped != nil {
					log.Printf("Subscription dropped: %v", subEvent.SubscriptionDropped.Error)
					return
				}
				continue
			}

			resolved := subEvent.EventAppeared.Event
			if resolved == nil {
				continue
			}
			recorded := resolved.Event
			if recorded == nil {
				continue
			}

			// Skip system events
			if len(recorded.EventType) > 0 && recorded.EventType[0] == '$' {
				sub.Ack(resolved)
				continue
			}

			esEvent, err := s.eventAppearedToEvent(resolved)
			if err != nil {
				log.Printf("Failed to convert event: %v", err)
				sub.Nack("conversion error", esdb.NackActionRetry, resolved)
				continue
			}

			if err := handler(ctx, esEvent); err != nil {
				log.Printf("Handler error for event %s: %v", esEvent.ID, err)
				sub.Nack("handler error", esdb.NackActionRetry, resolved)
				continue
			}

			sub.Ack(resolved)
		}
	}
}

// eventAppearedToEvent converts a KurrentDB EventAppeared to our event type.
func (s *Subscriber) eventAppearedToEvent(appeared *esdb.ResolvedEvent) (*eventstore.Event, error) {
	event := appeared.Event
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

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
	aggregateType, aggregateID := parseStreamName(event.StreamID)

	return &eventstore.Event{
		ID:            parseEventID(event.EventID),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     event.EventType,
		Version:       int(event.EventNumber) + 1,
		Timestamp:     event.CreatedDate,
		Data:          data,
		Metadata:      metadata,
	}, nil
}

// Close is a no-op for KurrentDB as the client manages the connection.
func (s *Subscriber) Close() {
	// Connection managed by Client
}

// CatchUpSubscriber replays events from a position and then switches to live streaming.
type CatchUpSubscriber struct {
	client *Client
}

// NewCatchUpSubscriber creates a subscriber that catches up from a position first.
func NewCatchUpSubscriber(client *Client) *CatchUpSubscriber {
	return &CatchUpSubscriber{client: client}
}

// SubscribeToStream subscribes to a stream from a specific position.
func (s *CatchUpSubscriber) SubscribeToStream(
	ctx context.Context,
	stream string,
	fromPosition uint64,
	handler eventstore.EventHandler,
) error {
	var from esdb.StreamPosition
	if fromPosition == 0 {
		from = esdb.Start{}
	} else {
		from = esdb.Revision(fromPosition)
	}

	sub, err := s.client.DB().SubscribeToStream(ctx, stream, esdb.SubscribeToStreamOptions{
		From: from,
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to stream: %w", err)
	}

	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				subEvent := sub.Recv()
				if subEvent.EventAppeared == nil {
					if subEvent.SubscriptionDropped != nil {
						log.Printf("Subscription dropped: %v", subEvent.SubscriptionDropped.Error)
						return
					}
					time.Sleep(10 * time.Millisecond)
					continue
				}

				resolved := subEvent.EventAppeared
				esEvent, err := eventAppearedToEventDirect(resolved)
				if err != nil {
					log.Printf("Failed to convert event: %v", err)
					continue
				}

				if err := handler(ctx, esEvent); err != nil {
					log.Printf("Handler error for event %s: %v", esEvent.ID, err)
				}
			}
		}
	}()

	return nil
}

// eventAppearedToEventDirect converts without a subscriber receiver.
func eventAppearedToEventDirect(appeared *esdb.ResolvedEvent) (*eventstore.Event, error) {
	event := appeared.Event
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	var data map[string]any
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	var metadata eventstore.EventMetadata
	if len(event.UserMetadata) > 0 {
		if err := json.Unmarshal(event.UserMetadata, &metadata); err != nil {
			metadata = eventstore.EventMetadata{}
		}
	}

	aggregateType, aggregateID := parseStreamName(event.StreamID)

	return &eventstore.Event{
		ID:            parseEventID(event.EventID),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     event.EventType,
		Version:       int(event.EventNumber) + 1,
		Timestamp:     event.CreatedDate,
		Data:          data,
		Metadata:      metadata,
	}, nil
}
