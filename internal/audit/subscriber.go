package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/serbia-gov/platform/internal/shared/events"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Subscriber listens to domain events and creates audit entries
type Subscriber struct {
	repo AuditRepository
	bus  events.EventBus
}

// NewSubscriber creates a new audit subscriber
func NewSubscriber(repo AuditRepository, bus events.EventBus) *Subscriber {
	return &Subscriber{repo: repo, bus: bus}
}

// Start subscribes to all relevant events
func (s *Subscriber) Start(ctx context.Context) error {
	// Subscribe to all domain events
	patterns := []struct {
		pattern      string
		consumerName string
	}{
		{"case.*", "audit-case-subscriber"},
		{"document.*", "audit-document-subscriber"},
		{"agency.*", "audit-agency-subscriber"},
		{"auth.*", "audit-auth-subscriber"},
		{"simulation.*", "audit-simulation-subscriber"},
	}

	for _, p := range patterns {
		if err := s.bus.Subscribe(ctx, p.pattern, p.consumerName, s.handleEvent); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", p.pattern, err)
		}
	}

	return nil
}

// handleEvent processes incoming events and creates audit entries
func (s *Subscriber) handleEvent(ctx context.Context, event events.Event) error {
	// Convert event to audit entry
	entry := s.eventToAuditEntry(event)
	if entry == nil {
		return nil // Skip events that don't need auditing
	}

	// Append to audit log
	if err := s.repo.Append(ctx, entry); err != nil {
		return fmt.Errorf("failed to append audit entry: %w", err)
	}

	return nil
}

// eventToAuditEntry converts a domain event to an audit entry
func (s *Subscriber) eventToAuditEntry(event events.Event) *AuditEntry {
	// Parse event type to determine action and resource
	parts := strings.SplitN(event.Type, ".", 2)
	if len(parts) < 2 {
		return nil
	}

	resourceType := parts[0]
	action := event.Type

	// Extract resource ID from event data
	var resourceID *types.ID
	if data, ok := event.Data.(map[string]any); ok {
		// Look for common ID field patterns
		idFields := []string{
			resourceType + "_id",
			"id",
		}
		for _, field := range idFields {
			if idVal, ok := data[field]; ok {
				if idStr, ok := idVal.(string); ok {
					id := types.ID(idStr)
					resourceID = &id
					break
				}
				if id, ok := idVal.(types.ID); ok {
					resourceID = &id
					break
				}
			}
		}
	}

	// Determine actor type
	actorType := ActorTypeWorker
	switch event.ActorType {
	case "citizen":
		actorType = ActorTypeCitizen
	case "system":
		actorType = ActorTypeSystem
	case "external":
		actorType = ActorTypeExternal
	}

	// Create audit entry
	// Truncate timestamp to microseconds for deterministic hash verification
	entry := &AuditEntry{
		ID:           types.NewID(),
		Timestamp:    event.Timestamp.UTC().Truncate(time.Microsecond),
		ActorType:    actorType,
		ActorID:      event.ActorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	// Set actor agency if available
	if !event.ActorAgency.IsZero() {
		entry.ActorAgencyID = &event.ActorAgency
	}

	// Set correlation ID if available
	if event.CorrelationID != "" {
		correlationID := types.ID(event.CorrelationID)
		entry.CorrelationID = &correlationID
	}

	// Extract changes from event data
	if data, ok := event.Data.(map[string]any); ok {
		entry.Changes = data
	}

	return entry
}
