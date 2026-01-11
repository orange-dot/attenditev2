package kurrentdb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/serbia-gov/platform/internal/eventstore"
)

// Publisher implements the eventstore.EventPublisher interface using KurrentDB.
// In KurrentDB, publishing is the same as appending to a stream, so this
// publishes to category streams for real-time subscribers.
type Publisher struct {
	client *Client
}

// NewPublisher creates a new KurrentDB-backed event publisher.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish sends events to KurrentDB streams.
// Events are published to both aggregate-specific streams and category streams.
func (p *Publisher) Publish(ctx context.Context, events []*eventstore.Event) error {
	for _, event := range events {
		// Publish to the aggregate stream
		stream := streamName(event.AggregateType, event.AggregateID)

		data, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}

		metadata, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal event metadata: %w", err)
		}

		esdbEvent := esdb.EventData{
			EventType:   event.EventType,
			ContentType: esdb.ContentTypeJson,
			Data:        data,
			Metadata:    metadata,
			EventID:     toUUID(event.ID),
		}

		_, err = p.client.DB().AppendToStream(ctx, stream, esdb.AppendToStreamOptions{
			ExpectedRevision: esdb.Any{},
		}, esdbEvent)

		if err != nil {
			return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
		}
	}

	return nil
}

// Close is a no-op for KurrentDB as the client manages the connection.
func (p *Publisher) Close() {
	// Connection managed by Client
}

// Health checks the KurrentDB connection.
func (p *Publisher) Health() error {
	return p.client.HealthCheck(context.Background())
}
