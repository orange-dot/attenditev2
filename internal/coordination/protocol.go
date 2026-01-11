package coordination

import (
	"context"
	"fmt"
	"sync"
)

// ProtocolEngine evaluates and executes coordination protocols
type ProtocolEngine struct {
	protocols map[string]*Protocol
	mu        sync.RWMutex

	// Dependencies
	enrichment *EnrichmentService
	escalation *EscalationService
	notifier   NotificationSender

	// Protocol registry by event type
	byEventType map[EventType][]*Protocol
}

// NewProtocolEngine creates a new protocol engine
func NewProtocolEngine(
	enrichment *EnrichmentService,
	escalation *EscalationService,
	notifier NotificationSender,
) *ProtocolEngine {
	return &ProtocolEngine{
		protocols:   make(map[string]*Protocol),
		byEventType: make(map[EventType][]*Protocol),
		enrichment:  enrichment,
		escalation:  escalation,
		notifier:    notifier,
	}
}

// RegisterProtocol registers a coordination protocol
func (e *ProtocolEngine) RegisterProtocol(protocol *Protocol) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.protocols[protocol.ID]; exists {
		return fmt.Errorf("protocol %s already registered", protocol.ID)
	}

	e.protocols[protocol.ID] = protocol
	e.byEventType[protocol.TriggerType] = append(e.byEventType[protocol.TriggerType], protocol)

	return nil
}

// UnregisterProtocol removes a protocol
func (e *ProtocolEngine) UnregisterProtocol(protocolID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	protocol, exists := e.protocols[protocolID]
	if !exists {
		return
	}

	delete(e.protocols, protocolID)

	// Remove from byEventType
	protocols := e.byEventType[protocol.TriggerType]
	for i, p := range protocols {
		if p.ID == protocolID {
			e.byEventType[protocol.TriggerType] = append(protocols[:i], protocols[i+1:]...)
			break
		}
	}
}

// ProcessEvent processes a coordination event through applicable protocols
func (e *ProtocolEngine) ProcessEvent(ctx context.Context, event *CoordinationEvent) error {
	// Get applicable protocols
	protocols := e.getApplicableProtocols(event)
	if len(protocols) == 0 {
		// No protocols, but still enrich and route
		if err := e.enrichment.Enrich(ctx, event); err != nil {
			// Log but don't fail
			fmt.Printf("Failed to enrich event %s: %v\n", event.ID, err)
		}
		return nil
	}

	// Enrich the event
	if err := e.enrichment.Enrich(ctx, event); err != nil {
		fmt.Printf("Failed to enrich event %s: %v\n", event.ID, err)
	}

	// Execute all matching protocols
	for _, protocol := range protocols {
		if e.evaluateConditions(event, protocol.Conditions) {
			if err := e.executeProtocol(ctx, event, protocol); err != nil {
				fmt.Printf("Failed to execute protocol %s for event %s: %v\n", protocol.ID, event.ID, err)
			}
		}
	}

	return nil
}

// getApplicableProtocols returns protocols that may apply to an event
func (e *ProtocolEngine) getApplicableProtocols(event *CoordinationEvent) []*Protocol {
	e.mu.RLock()
	defer e.mu.RUnlock()

	protocols := make([]*Protocol, 0)
	for _, p := range e.byEventType[event.Type] {
		if p.IsActive {
			protocols = append(protocols, p)
		}
	}
	return protocols
}

// evaluateConditions checks if all conditions are met
func (e *ProtocolEngine) evaluateConditions(event *CoordinationEvent, conditions []Condition) bool {
	for _, cond := range conditions {
		if !e.evaluateCondition(event, cond) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition
func (e *ProtocolEngine) evaluateCondition(event *CoordinationEvent, cond Condition) bool {
	var fieldValue any

	// Get field value based on field path
	switch cond.Field {
	case "priority":
		fieldValue = string(event.Priority)
	case "type":
		fieldValue = string(event.Type)
	case "source_agency":
		fieldValue = event.SourceAgency
	case "source_system":
		fieldValue = event.SourceSystem
	case "risk_level":
		if event.Enrichment != nil {
			fieldValue = event.Enrichment.RiskLevel
		}
	case "risk_score":
		if event.Enrichment != nil {
			fieldValue = event.Enrichment.RiskScore
		}
	case "has_open_cases":
		if event.Enrichment != nil && event.Enrichment.SocialContext != nil {
			fieldValue = event.Enrichment.SocialContext.HasOpenCases
		}
	case "is_beneficiary":
		if event.Enrichment != nil && event.Enrichment.SocialContext != nil {
			fieldValue = event.Enrichment.SocialContext.IsBeneficiary
		}
	case "has_minor_family_members":
		if event.Enrichment != nil {
			for _, m := range event.Enrichment.FamilyMembers {
				if m.IsMinor {
					fieldValue = true
					break
				}
			}
		}
	case "requires_immediate_action":
		if event.Enrichment != nil && event.Enrichment.SocialContext != nil {
			fieldValue = event.Enrichment.SocialContext.RequiresImmediateAction
		}
	default:
		// Check in details map
		if event.Details != nil {
			fieldValue = event.Details[cond.Field]
		}
	}

	// Evaluate based on operator
	switch cond.Operator {
	case "eq":
		return fieldValue == cond.Value
	case "ne":
		return fieldValue != cond.Value
	case "gt":
		return compareNumeric(fieldValue, cond.Value) > 0
	case "gte":
		return compareNumeric(fieldValue, cond.Value) >= 0
	case "lt":
		return compareNumeric(fieldValue, cond.Value) < 0
	case "lte":
		return compareNumeric(fieldValue, cond.Value) <= 0
	case "contains":
		strVal, ok := fieldValue.(string)
		condStr, ok2 := cond.Value.(string)
		if ok && ok2 {
			return strVal != "" && condStr != "" && contains(strVal, condStr)
		}
	case "in":
		if arr, ok := cond.Value.([]any); ok {
			for _, v := range arr {
				if fieldValue == v {
					return true
				}
			}
		}
	}

	return false
}

// executeProtocol executes a protocol's actions
func (e *ProtocolEngine) executeProtocol(ctx context.Context, event *CoordinationEvent, protocol *Protocol) error {
	for _, action := range protocol.Actions {
		// Handle delays
		// Note: In production, use proper scheduling
		if action.Delay > 0 {
			// For now, skip delayed actions (would need async handling)
			continue
		}

		if err := e.executeAction(ctx, event, action); err != nil {
			return fmt.Errorf("action %s failed: %w", action.Type, err)
		}
	}

	// Register for escalation if protocol has escalation rules
	if protocol.Escalation != nil && len(protocol.Escalation.Levels) > 0 {
		e.escalation.RegisterEvent(event, protocol)
	}

	return nil
}

// executeAction executes a single protocol action
func (e *ProtocolEngine) executeAction(ctx context.Context, event *CoordinationEvent, action Action) error {
	switch action.Type {
	case "notify":
		return e.executeNotifyAction(ctx, event, action)
	case "route":
		return e.executeRouteAction(ctx, event, action)
	case "escalate":
		return e.executeEscalateAction(ctx, event, action)
	case "set_priority":
		return e.executeSetPriorityAction(event, action)
	case "add_target":
		return e.executeAddTargetAction(event, action)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// executeNotifyAction sends a notification
func (e *ProtocolEngine) executeNotifyAction(ctx context.Context, event *CoordinationEvent, action Action) error {
	target, _ := action.Parameters["target"].(string)
	notificationType, _ := action.Parameters["notification_type"].(string)
	if notificationType == "" {
		notificationType = "push"
	}

	template, _ := action.Parameters["template"].(string)
	subject, body := e.formatNotification(event, template)

	notification := Notification{
		ID:       fmt.Sprintf("ntf-%s-%d", event.ID, len(event.Acknowledged)),
		EventID:  event.ID,
		Type:     notificationType,
		Priority: event.Priority,
		Subject:  subject,
		Body:     body,
		Recipient: Recipient{
			Type: "role",
			ID:   target,
		},
		Status: "pending",
		Data: map[string]any{
			"event_id":     event.ID,
			"event_type":   string(event.Type),
			"subject_jmbg": event.SubjectJMBG,
		},
	}

	return e.notifier.Send(ctx, notification)
}

// executeRouteAction routes an event to agencies
func (e *ProtocolEngine) executeRouteAction(ctx context.Context, event *CoordinationEvent, action Action) error {
	targets, _ := action.Parameters["targets"].([]any)
	for _, t := range targets {
		if target, ok := t.(string); ok {
			event.TargetAgencies = append(event.TargetAgencies, target)
		}
	}
	event.Status = EventStatusRouted
	return nil
}

// executeEscalateAction triggers immediate escalation
func (e *ProtocolEngine) executeEscalateAction(ctx context.Context, event *CoordinationEvent, action Action) error {
	level, _ := action.Parameters["level"].(int)
	if level == 0 {
		level = 1
	}

	// Create escalation notification
	notification := Notification{
		ID:       fmt.Sprintf("esc-%s-%d", event.ID, level),
		EventID:  event.ID,
		Type:     "push",
		Priority: PriorityCritical,
		Subject:  fmt.Sprintf("[IMMEDIATE ESCALATION] %s", event.Title),
		Body:     fmt.Sprintf("Event requires immediate escalation to level %d.\n\n%s", level, event.Description),
		Recipient: Recipient{
			Type: "role",
			ID:   fmt.Sprintf("level_%d_responder", level),
		},
		Status: "pending",
	}

	event.Status = EventStatusEscalated
	return e.notifier.Send(ctx, notification)
}

// executeSetPriorityAction changes event priority
func (e *ProtocolEngine) executeSetPriorityAction(event *CoordinationEvent, action Action) error {
	priority, _ := action.Parameters["priority"].(string)
	if priority != "" {
		event.Priority = Priority(priority)
	}
	return nil
}

// executeAddTargetAction adds a target agency
func (e *ProtocolEngine) executeAddTargetAction(event *CoordinationEvent, action Action) error {
	target, _ := action.Parameters["agency"].(string)
	if target != "" {
		event.TargetAgencies = append(event.TargetAgencies, target)
	}
	return nil
}

// formatNotification formats notification subject and body
func (e *ProtocolEngine) formatNotification(event *CoordinationEvent, template string) (subject, body string) {
	switch template {
	case "hospital_admission":
		subject = fmt.Sprintf("Hospital Admission: %s", event.SubjectName)
		body = fmt.Sprintf(`A person has been admitted to hospital.

Subject: %s (JMBG: %s)
Type: %s
Source: %s

%s

Please review and acknowledge this notification.`,
			event.SubjectName, event.SubjectJMBG, event.Type, event.SourceAgency, event.Description)

	case "hospital_discharge":
		subject = fmt.Sprintf("Hospital Discharge: %s", event.SubjectName)
		body = fmt.Sprintf(`A person has been discharged from hospital.

Subject: %s (JMBG: %s)
Type: %s
Source: %s

%s

Please ensure follow-up care is arranged if needed.`,
			event.SubjectName, event.SubjectJMBG, event.Type, event.SourceAgency, event.Description)

	case "child_protection":
		subject = fmt.Sprintf("[URGENT] Child Protection Alert: %s", event.SubjectName)
		body = fmt.Sprintf(`URGENT: Child protection concern identified.

Subject: %s (JMBG: %s)
Priority: %s
Source: %s

%s

This requires immediate attention. Please acknowledge and take action.`,
			event.SubjectName, event.SubjectJMBG, event.Priority, event.SourceAgency, event.Description)

	case "domestic_violence":
		subject = fmt.Sprintf("[URGENT] Domestic Violence Alert: %s", event.SubjectName)
		body = fmt.Sprintf(`URGENT: Domestic violence concern identified.

Subject: %s (JMBG: %s)
Priority: %s
Source: %s

%s

Ensure victim safety. Coordinate with police if needed.`,
			event.SubjectName, event.SubjectJMBG, event.Priority, event.SourceAgency, event.Description)

	default:
		subject = fmt.Sprintf("[%s] %s", event.Priority, event.Title)
		body = fmt.Sprintf(`Coordination Event

Type: %s
Subject: %s (JMBG: %s)
Priority: %s
Source: %s

%s`,
			event.Type, event.SubjectName, event.SubjectJMBG, event.Priority, event.SourceAgency, event.Description)
	}

	// Append enrichment info if available
	if event.Enrichment != nil {
		if event.Enrichment.RiskLevel != "" && event.Enrichment.RiskLevel != "low" {
			body += fmt.Sprintf("\n\n⚠️ Risk Level: %s (Score: %d)", event.Enrichment.RiskLevel, event.Enrichment.RiskScore)
		}

		if len(event.Enrichment.RecommendedActions) > 0 {
			body += "\n\nRecommended Actions:"
			for _, action := range event.Enrichment.RecommendedActions {
				body += fmt.Sprintf("\n• %s", action)
			}
		}
	}

	return subject, body
}

// GetProtocol returns a protocol by ID
func (e *ProtocolEngine) GetProtocol(id string) (*Protocol, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.protocols[id]
	return p, ok
}

// ListProtocols returns all registered protocols
func (e *ProtocolEngine) ListProtocols() []*Protocol {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*Protocol, 0, len(e.protocols))
	for _, p := range e.protocols {
		result = append(result, p)
	}
	return result
}

// Helper functions

func compareNumeric(a, b any) int {
	aFloat := toFloat64(a)
	bFloat := toFloat64(b)

	if aFloat < bFloat {
		return -1
	} else if aFloat > bFloat {
		return 1
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
