package kurrentdb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
)

// Client wraps the EventStore client with additional functionality.
type Client struct {
	db     *esdb.Client
	config *Config
	mu     sync.RWMutex
}

// NewClient creates a new KurrentDB client.
func NewClient(cfg *Config) (*Client, error) {
	settings, err := esdb.ParseConnectionString(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	db, err := esdb.NewClient(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Client{
		db:     db,
		config: cfg,
	}, nil
}

// Connect establishes connection to KurrentDB and verifies it's ready.
func (c *Client) Connect(ctx context.Context) error {
	// Verify connection by reading server info
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to read from a system stream to verify connection
	_, err := c.db.ReadStream(ctx, "$streams", esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, 1)

	// ReadStream returns an iterator, not an error for missing streams
	// So we check if we can read at all
	if err != nil {
		return fmt.Errorf("failed to verify connection: %w", err)
	}

	return nil
}

// DB returns the underlying EventStore client.
func (c *Client) DB() *esdb.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.db
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// HealthCheck verifies the connection is alive.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to read stream to verify connection is alive
	stream, err := c.db.ReadStream(ctx, "$streams", esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, 1)

	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer stream.Close()

	return nil
}

// StreamExists checks if a stream exists.
func (c *Client) StreamExists(ctx context.Context, streamName string) (bool, error) {
	stream, err := c.db.ReadStream(ctx, streamName, esdb.ReadStreamOptions{
		From:      esdb.Start{},
		Direction: esdb.Forwards,
	}, 1)

	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return false, nil
			}
		}
		return false, err
	}
	defer stream.Close()

	// Try to read one event
	_, err = stream.Recv()
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return false, nil
			}
		}
		// Stream exists but might be empty - that's ok
		return true, nil
	}

	return true, nil
}

// GetStreamLastPosition returns the last event position in a stream.
func (c *Client) GetStreamLastPosition(ctx context.Context, streamName string) (uint64, error) {
	stream, err := c.db.ReadStream(ctx, streamName, esdb.ReadStreamOptions{
		From:      esdb.End{},
		Direction: esdb.Backwards,
	}, 1)

	if err != nil {
		return 0, err
	}
	defer stream.Close()

	event, err := stream.Recv()
	if err != nil {
		if esdbErr, ok := esdb.FromError(err); ok {
			if esdbErr.Code() == esdb.ErrorCodeResourceNotFound {
				return 0, nil
			}
		}
		return 0, err
	}

	return event.Event.EventNumber, nil
}
