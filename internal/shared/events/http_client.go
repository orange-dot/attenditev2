package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/serbia-gov/platform/internal/shared/config"
)

// HTTPClient provides EventStoreDB operations via HTTP API (AtomPub)
// This is a fallback when gRPC doesn't work (e.g., Docker networking issues)
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// NewHTTPClient creates a new HTTP-based EventStoreDB client
func NewHTTPClient(cfg config.KurrentDBConfig) *HTTPClient {
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}

	return &HTTPClient{
		baseURL:  fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port),
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EventData represents an event to be written
type EventData struct {
	EventID   string `json:"eventId"`
	EventType string `json:"eventType"`
	Data      any    `json:"data"`
	Metadata  any    `json:"metadata,omitempty"`
}

// RecordedEvent represents an event read from a stream
type RecordedEvent struct {
	EventID             string          `json:"eventId"`
	EventType           string          `json:"eventType"`
	EventNumber         int64           `json:"eventNumber"`
	Data                json.RawMessage `json:"data"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	StreamID            string          `json:"streamId"`
	Created             time.Time       `json:"created"`
	IsLinkEvent         bool            `json:"isLinkEvent"`
	PositionEventNumber int64           `json:"positionEventNumber,omitempty"` // Position in projection stream
}

// StreamEntry represents an entry in the atom feed
type StreamEntry struct {
	Title   string `json:"title"`
	ID      string `json:"id"`
	Updated string `json:"updated"`
	Author  struct {
		Name string `json:"name"`
	} `json:"author"`
	Summary string `json:"summary"`
	Content struct {
		EventStreamID string          `json:"eventStreamId"`
		EventNumber   int64           `json:"eventNumber"`
		EventType     string          `json:"eventType"`
		EventID       string          `json:"eventId"`
		Data          json.RawMessage `json:"data"`
		Metadata      json.RawMessage `json:"metadata"`
	} `json:"content"`
	Links []struct {
		URI      string `json:"uri"`
		Relation string `json:"relation"`
	} `json:"links"`
	// Fields for embed=body format (used with category streams)
	EventID             string          `json:"eventId,omitempty"`
	EventType           string          `json:"eventType,omitempty"`
	EventNumber         int64           `json:"eventNumber,omitempty"`
	Data                json.RawMessage `json:"data,omitempty"`
	Metadata            json.RawMessage `json:"metaData,omitempty"`
	StreamID            string          `json:"streamId,omitempty"`
	PositionEventNumber int64           `json:"positionEventNumber,omitempty"` // Position in projection stream
}

// AtomFeed represents the atom feed response from EventStoreDB
type AtomFeed struct {
	Title   string        `json:"title"`
	ID      string        `json:"id"`
	Updated string        `json:"updated"`
	Author  struct{ Name string } `json:"author"`
	Links   []struct {
		URI      string `json:"uri"`
		Relation string `json:"relation"`
	} `json:"links"`
	Entries []StreamEntry `json:"entries"`
}

// AppendToStream appends events to a stream
func (c *HTTPClient) AppendToStream(ctx context.Context, stream string, events ...EventData) error {
	url := fmt.Sprintf("%s/streams/%s", c.baseURL, stream)

	// Prepare events array
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(eventsJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.eventstore.events+json")
	req.Header.Set("ES-ExpectedVersion", "-2") // Any version

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to append events: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ReadStreamOptions configures stream reading
type ReadStreamOptions struct {
	Direction string // "forward" or "backward"
	Start     int64  // Start position (-1 for head)
	Count     int    // Number of events to read
}

// ReadStream reads events from a stream
func (c *HTTPClient) ReadStream(ctx context.Context, stream string, opts ReadStreamOptions) ([]RecordedEvent, error) {
	if opts.Count == 0 {
		opts.Count = 20
	}

	// Build URL based on options
	var url string
	if opts.Start == -1 {
		// Read from head (latest events)
		url = fmt.Sprintf("%s/streams/%s/head/%d", c.baseURL, stream, opts.Count)
	} else {
		url = fmt.Sprintf("%s/streams/%s/%d/%s/%d", c.baseURL, stream, opts.Start, opts.Direction, opts.Count)
	}

	// Always add embed=body to get actual event data (not just metadata)
	url += "?embed=body"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.eventstore.atom+json")
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Stream doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read stream: status %d, body: %s", resp.StatusCode, string(body))
	}

	var feed AtomFeed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert entries to RecordedEvents
	events := make([]RecordedEvent, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		var event RecordedEvent

		// Check if embed=body format (direct fields) or regular format (Content nested)
		if entry.EventID != "" {
			// embed=body format - fields are directly on entry
			event = RecordedEvent{
				EventID:             entry.EventID,
				EventType:           entry.EventType,
				EventNumber:         entry.EventNumber,
				Data:                entry.Data,
				Metadata:            entry.Metadata,
				StreamID:            entry.StreamID,
				PositionEventNumber: entry.PositionEventNumber,
			}
		} else if entry.Content.EventID != "" {
			// Regular format - fields are nested in Content
			event = RecordedEvent{
				EventID:             entry.Content.EventID,
				EventType:           entry.Content.EventType,
				EventNumber:         entry.Content.EventNumber,
				Data:                entry.Content.Data,
				Metadata:            entry.Content.Metadata,
				StreamID:            entry.Content.EventStreamID,
				PositionEventNumber: entry.Content.EventNumber, // Same as EventNumber for regular streams
			}
		} else {
			// Skip entries without event data (e.g., unresolved links)
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

// ReadLastEvent reads the last event from a stream
func (c *HTTPClient) ReadLastEvent(ctx context.Context, stream string) (*RecordedEvent, error) {
	url := fmt.Sprintf("%s/streams/%s/head/backward/1?embed=body", c.baseURL, stream)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.eventstore.atom+json")
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Stream doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to read stream: status %d, body: %s", resp.StatusCode, string(body))
	}

	var feed AtomFeed
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(feed.Entries) == 0 {
		return nil, nil
	}

	entry := feed.Entries[0]

	// Check if embed=body format (direct fields) or regular format (Content nested)
	if entry.EventID != "" {
		// embed=body format - fields are directly on entry
		return &RecordedEvent{
			EventID:     entry.EventID,
			EventType:   entry.EventType,
			EventNumber: entry.EventNumber,
			Data:        entry.Data,
			Metadata:    entry.Metadata,
			StreamID:    entry.StreamID,
		}, nil
	}

	// Regular format - fields are nested in Content
	return &RecordedEvent{
		EventID:     entry.Content.EventID,
		EventType:   entry.Content.EventType,
		EventNumber: entry.Content.EventNumber,
		Data:        entry.Content.Data,
		Metadata:    entry.Content.Metadata,
		StreamID:    entry.Content.EventStreamID,
	}, nil
}

// Health checks the EventStoreDB connection via HTTP
func (c *HTTPClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/info", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// HTTPBus provides event publishing and subscription using HTTP API
type HTTPBus struct {
	client       *HTTPClient
	prefix       string
	subscriptions map[string]context.CancelFunc
	mu           sync.Mutex
}

// NewHTTPBus creates a new HTTP-based event bus
func NewHTTPBus(ctx context.Context, cfg config.KurrentDBConfig) (*HTTPBus, error) {
	client := NewHTTPClient(cfg)

	// Verify connection
	if err := client.Health(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to EventStoreDB via HTTP: %w", err)
	}

	return &HTTPBus{
		client:       client,
		prefix:       "gov",
		subscriptions: make(map[string]context.CancelFunc),
	}, nil
}

// Publish publishes an event via HTTP
func (b *HTTPBus) Publish(ctx context.Context, event Event) error {
	stream := fmt.Sprintf("%s-%s", b.prefix, normalizeEventType(event.Type))

	eventID := event.ID
	if eventID == "" {
		eventID = uuid.New().String()
	}

	eventData := EventData{
		EventID:   eventID,
		EventType: event.Type,
		Data:      event,
	}

	return b.client.AppendToStream(ctx, stream, eventData)
}

// Subscribe creates a polling subscription to events
func (b *HTTPBus) Subscribe(ctx context.Context, pattern string, consumerName string, handler Handler) error {
	b.mu.Lock()
	if cancel, exists := b.subscriptions[consumerName]; exists {
		cancel()
	}

	subCtx, cancel := context.WithCancel(ctx)
	b.subscriptions[consumerName] = cancel
	b.mu.Unlock()

	go b.pollForEvents(subCtx, pattern, handler)
	return nil
}

// pollForEvents polls for new events using HTTP long-polling
func (b *HTTPBus) pollForEvents(ctx context.Context, pattern string, handler Handler) {
	// Track last position per stream
	positions := make(map[string]int64)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Poll $all stream and filter by pattern
			streams := b.getStreamsForPattern(ctx, pattern)
			for _, stream := range streams {
				b.pollStream(ctx, stream, positions, pattern, handler)
			}
		}
	}
}

// getStreamsForPattern returns streams matching the pattern
func (b *HTTPBus) getStreamsForPattern(ctx context.Context, pattern string) []string {
	// Use category projection for the gov prefix
	// This contains all events from gov-* streams
	return []string{"$ce-" + b.prefix}
}

// pollStream polls a single stream for new events
func (b *HTTPBus) pollStream(ctx context.Context, stream string, positions map[string]int64, pattern string, handler Handler) {
	lastPos := positions[stream]

	events, err := b.client.ReadStream(ctx, stream, ReadStreamOptions{
		Direction: "forward",
		Start:     lastPos,
		Count:     100,
	})
	if err != nil {
		return // Silently continue on error
	}

	for _, recorded := range events {
		// Use PositionEventNumber for tracking (works for both regular and projection streams)
		nextPos := recorded.PositionEventNumber + 1
		if nextPos == 1 && recorded.EventNumber > 0 {
			// Fallback for regular streams where PositionEventNumber might be 0
			nextPos = recorded.EventNumber + 1
		}

		// Skip system events
		if len(recorded.EventType) > 0 && recorded.EventType[0] == '$' {
			positions[stream] = nextPos
			continue
		}

		// Filter by pattern (e.g., "simulation.*" matches "simulation.started")
		if !matchEventTypePattern(recorded.EventType, pattern) {
			positions[stream] = nextPos
			continue
		}

		var event Event
		// The Data field may be a JSON string (from embed=body) that needs double-unmarshal
		data := recorded.Data

		// Check for JSON string format (starts with " after trimming whitespace)
		trimmedData := bytes.TrimSpace(data)
		if len(trimmedData) > 0 && trimmedData[0] == '"' {
			// Data is a JSON string, unmarshal to get the inner string
			var dataStr string
			if err := json.Unmarshal(data, &dataStr); err != nil {
				positions[stream] = nextPos
				continue
			}
			data = json.RawMessage(dataStr)
		}

		if err := json.Unmarshal(data, &event); err != nil {
			positions[stream] = nextPos
			continue
		}

		if event.ID == "" {
			event.ID = recorded.EventID
		}
		if event.Type == "" {
			event.Type = recorded.EventType
		}

		// Call handler (ignore errors)
		handler(ctx, event)
		positions[stream] = nextPos
	}
}

// matchEventTypePattern checks if an event type matches a pattern
// Pattern format: "category.*" or "category.action" or "*"
func matchEventTypePattern(eventType, pattern string) bool {
	if pattern == "*" || pattern == ">" {
		return true
	}

	// Handle patterns like "simulation.*"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(eventType, prefix+".")
	}

	// Exact match
	return eventType == pattern
}

// Close closes the HTTP bus
func (b *HTTPBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, cancel := range b.subscriptions {
		cancel()
	}
	b.subscriptions = make(map[string]context.CancelFunc)
}

// Client returns nil (HTTP bus doesn't have a gRPC client)
func (b *HTTPBus) Client() interface{} {
	return b.client
}

// HTTPClient returns the underlying HTTP client
func (b *HTTPBus) HTTPClient() *HTTPClient {
	return b.client
}

// Health checks the connection
func (b *HTTPBus) Health() error {
	return b.client.Health(context.Background())
}
// Build timestamp: Mon, Jan 12, 2026  7:13:37 AM
