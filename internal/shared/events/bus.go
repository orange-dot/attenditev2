package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/google/uuid"
	"github.com/serbia-gov/platform/internal/shared/config"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Event represents a domain event
type Event struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	Timestamp     time.Time `json:"timestamp"`
	CorrelationID string    `json:"correlation_id,omitempty"`

	// Actor information
	ActorID     types.ID `json:"actor_id"`
	ActorType   string   `json:"actor_type"` // citizen, worker, system
	ActorAgency types.ID `json:"actor_agency,omitempty"`

	// Event data
	Data any `json:"data"`
}

// NewEvent creates a new event with auto-generated ID and timestamp
func NewEvent(eventType, source string, data any) Event {
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

// WithActor sets the actor information on the event
func (e Event) WithActor(actorID types.ID, actorType string, actorAgency types.ID) Event {
	e.ActorID = actorID
	e.ActorType = actorType
	e.ActorAgency = actorAgency
	return e
}

// WithCorrelation sets the correlation ID for request tracing
func (e Event) WithCorrelation(correlationID string) Event {
	e.CorrelationID = correlationID
	return e
}

// Handler is a function that handles an event
type Handler func(ctx context.Context, event Event) error

// Bus provides event publishing and subscription using KurrentDB
type Bus struct {
	client *esdb.Client
	prefix string
}

// NewBus creates a new event bus connected to KurrentDB
func NewBus(ctx context.Context, cfg config.KurrentDBConfig) (*Bus, error) {
	connString := buildConnectionString(cfg)

	settings, err := esdb.ParseConnectionString(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	client, err := esdb.NewClient(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create KurrentDB client: %w", err)
	}

	bus := &Bus{
		client: client,
		prefix: "gov",
	}

	return bus, nil
}

// buildConnectionString creates the esdb:// connection string
func buildConnectionString(cfg config.KurrentDBConfig) string {
	var auth string
	if cfg.Username != "" && cfg.Password != "" {
		auth = fmt.Sprintf("%s:%s@", cfg.Username, cfg.Password)
	}

	// Build query parameters
	params := ""
	if cfg.Insecure {
		params = "?tls=false&tlsVerifyCert=false"
	}
	// Add keep-alive and timeout settings for better connection stability
	if params != "" {
		params += "&keepAliveInterval=10000&keepAliveTimeout=10000&discoveryInterval=100&maxDiscoverAttempts=3&gossipTimeout=5"
	}

	return fmt.Sprintf("esdb://%s%s:%d%s", auth, cfg.Host, cfg.Port, params)
}

// Publish publishes an event to the bus
func (b *Bus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create stream name from event type: gov.case.created -> gov-case-created
	stream := fmt.Sprintf("%s-%s", b.prefix, normalizeEventType(event.Type))

	eventID, err := uuid.Parse(event.ID)
	if err != nil {
		eventID = uuid.New()
	}

	esdbEvent := esdb.EventData{
		EventType:   event.Type,
		ContentType: esdb.ContentTypeJson,
		Data:        data,
		EventID:     eventID,
	}

	_, err = b.client.AppendToStream(ctx, stream, esdb.AppendToStreamOptions{
		ExpectedRevision: esdb.Any{},
	}, esdbEvent)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// normalizeEventType converts event type to stream-safe format
func normalizeEventType(eventType string) string {
	// Replace dots with hyphens for stream name
	result := make([]byte, len(eventType))
	for i := 0; i < len(eventType); i++ {
		if eventType[i] == '.' {
			result[i] = '-'
		} else {
			result[i] = eventType[i]
		}
	}
	return string(result)
}

// Subscribe creates a persistent subscription to events matching a pattern
func (b *Bus) Subscribe(ctx context.Context, pattern string, consumerName string, handler Handler) error {
	// Convert pattern to stream name
	// Pattern like "case.*" becomes subscription to gov-case-* streams
	stream := fmt.Sprintf("$ce-%s", normalizeEventType(pattern))
	if pattern == "*" || pattern == ">" {
		stream = "$all"
	}

	// Try to create persistent subscription (ignore if already exists)
	settings := esdb.SubscriptionSettingsDefault()
	settings.ResolveLinkTos = true

	if stream == "$all" {
		err := b.client.CreatePersistentSubscriptionToAll(ctx, consumerName, esdb.PersistentAllSubscriptionOptions{
			Settings:  &settings,
			StartFrom: esdb.End{},
		})
		if err != nil {
			if esdbErr, ok := esdb.FromError(err); ok {
				if esdbErr.Code() != esdb.ErrorCodeResourceAlreadyExists {
					return fmt.Errorf("failed to create persistent subscription: %w", err)
				}
			}
		}

		return b.subscribeToPersistentAll(ctx, consumerName, pattern, handler)
	}

	// For category streams, use a simpler approach - subscribe to streams matching the pattern
	return b.subscribeToPattern(ctx, pattern, consumerName, handler)
}

// subscribeToPattern subscribes to events matching a wildcard pattern using catch-up subscription
func (b *Bus) subscribeToPattern(ctx context.Context, pattern string, consumerName string, handler Handler) error {
	// Use $all stream with filtering
	sub, err := b.client.SubscribeToAll(ctx, esdb.SubscribeToAllOptions{
		From: esdb.End{},
		Filter: &esdb.SubscriptionFilter{
			Type:  esdb.EventFilterType,
			Regex: patternToRegex(pattern),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to pattern: %w", err)
	}

	go b.handleCatchUpSubscription(ctx, sub, pattern, handler)
	return nil
}

// patternToRegex converts a simple wildcard pattern to regex
func patternToRegex(pattern string) string {
	// Convert patterns like "case.*" to "case\\..*"
	result := make([]byte, 0, len(pattern)*2)
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '.':
			result = append(result, '\\', '.')
		case '*':
			result = append(result, '.', '*')
		default:
			result = append(result, pattern[i])
		}
	}
	return string(result)
}

// subscribeToPersistentAll subscribes to a persistent subscription on $all
func (b *Bus) subscribeToPersistentAll(ctx context.Context, consumerName, pattern string, handler Handler) error {
	sub, err := b.client.SubscribeToPersistentSubscriptionToAll(ctx, consumerName, esdb.SubscribeToPersistentSubscriptionOptions{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to persistent subscription: %w", err)
	}

	go b.handlePersistentSubscription(ctx, sub, pattern, handler)
	return nil
}

// handlePersistentSubscription processes events from a persistent subscription
func (b *Bus) handlePersistentSubscription(ctx context.Context, sub *esdb.PersistentSubscription, pattern string, handler Handler) {
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

			// Check if event matches pattern
			if !matchesPattern(recorded.EventType, pattern) {
				sub.Ack(resolved)
				continue
			}

			event, err := b.recordedEventToEvent(recorded)
			if err != nil {
				log.Printf("Failed to convert event: %v", err)
				sub.Nack("conversion error", esdb.NackActionRetry, resolved)
				continue
			}

			if err := handler(ctx, event); err != nil {
				log.Printf("Handler error for event %s: %v", event.ID, err)
				sub.Nack("handler error", esdb.NackActionRetry, resolved)
				continue
			}

			sub.Ack(resolved)
		}
	}
}

// handleCatchUpSubscription processes events from a catch-up subscription
func (b *Bus) handleCatchUpSubscription(ctx context.Context, sub *esdb.Subscription, pattern string, handler Handler) {
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

			recorded := subEvent.EventAppeared.Event
			if recorded == nil {
				continue
			}

			// Skip system events
			if len(recorded.EventType) > 0 && recorded.EventType[0] == '$' {
				continue
			}

			// Check if event matches pattern
			if !matchesPattern(recorded.EventType, pattern) {
				continue
			}

			event, err := b.recordedEventToEvent(recorded)
			if err != nil {
				log.Printf("Failed to convert event: %v", err)
				continue
			}

			if err := handler(ctx, event); err != nil {
				log.Printf("Handler error for event %s: %v", event.ID, err)
			}
		}
	}
}

// matchesPattern checks if an event type matches a wildcard pattern
func matchesPattern(eventType, pattern string) bool {
	if pattern == "*" || pattern == ">" {
		return true
	}

	// Simple pattern matching: "case.*" matches "case.created", "case.updated"
	patternParts := splitString(pattern, ".")
	typeParts := splitString(eventType, ".")

	for i, pp := range patternParts {
		if pp == "*" {
			// Wildcard matches the rest
			return true
		}
		if i >= len(typeParts) {
			return false
		}
		if pp != typeParts[i] {
			return false
		}
	}

	return len(patternParts) == len(typeParts)
}

// splitString splits a string by separator
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// recordedEventToEvent converts a KurrentDB event to our Event type
func (b *Bus) recordedEventToEvent(recorded *esdb.RecordedEvent) (Event, error) {
	var event Event
	if err := json.Unmarshal(recorded.Data, &event); err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Ensure ID is set
	if event.ID == "" {
		event.ID = recorded.EventID.String()
	}

	return event, nil
}

// Close closes the event bus connection
func (b *Bus) Close() {
	if b.client != nil {
		b.client.Close()
	}
}

// Client returns the underlying KurrentDB client
func (b *Bus) Client() *esdb.Client {
	return b.client
}

// Health checks the KurrentDB connection
func (b *Bus) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to read from $streams to verify connection
	stream, err := b.client.ReadStream(ctx, "$streams", esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, 1)

	if err != nil {
		return fmt.Errorf("KurrentDB health check failed: %w", err)
	}
	defer stream.Close()

	return nil
}
