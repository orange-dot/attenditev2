package events

import (
	"context"
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/shared/config"
)

// EventBus defines the interface for event publishing and subscription
type EventBus interface {
	// Publish publishes an event to the bus
	Publish(ctx context.Context, event Event) error

	// Subscribe creates a subscription to events matching a pattern
	Subscribe(ctx context.Context, pattern string, consumerName string, handler Handler) error

	// Close closes the event bus connection
	Close()

	// Health checks the event bus connection
	Health() error
}

// NewEventBus creates an event bus, trying gRPC first and falling back to HTTP
func NewEventBus(ctx context.Context, cfg config.KurrentDBConfig) (EventBus, string, error) {
	// First, try HTTP since gRPC has issues with Docker networking
	// HTTP is more reliable across different network configurations
	httpBus, err := tryHTTPBus(ctx, cfg)
	if err == nil {
		return httpBus, "http", nil
	}
	httpErr := err

	// Then try gRPC
	grpcBus, err := tryGRPCBus(ctx, cfg)
	if err == nil {
		return grpcBus, "grpc", nil
	}
	grpcErr := err

	return nil, "", fmt.Errorf("failed to connect to EventStoreDB: HTTP error: %v, gRPC error: %v", httpErr, grpcErr)
}

// tryHTTPBus attempts to create an HTTP-based event bus
func tryHTTPBus(ctx context.Context, cfg config.KurrentDBConfig) (EventBus, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bus, err := NewHTTPBus(timeoutCtx, cfg)
	if err != nil {
		return nil, err
	}

	return bus, nil
}

// tryGRPCBus attempts to create a gRPC-based event bus
func tryGRPCBus(ctx context.Context, cfg config.KurrentDBConfig) (EventBus, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	bus, err := NewBus(timeoutCtx, cfg)
	if err != nil {
		return nil, err
	}

	// Test the connection with a health check
	if err := bus.Health(); err != nil {
		bus.Close()
		return nil, fmt.Errorf("gRPC health check failed: %w", err)
	}

	return bus, nil
}

// Ensure Bus implements EventBus
var _ EventBus = (*Bus)(nil)

// Ensure HTTPBus implements EventBus
var _ EventBus = (*HTTPBus)(nil)
