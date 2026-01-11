package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EscalationService manages event escalation
type EscalationService struct {
	notifier NotificationSender

	// Active escalations
	active   map[string]*activeEscalation
	activeMu sync.RWMutex

	// Configuration
	config EscalationConfig

	// Channels
	stopCh chan struct{}
}

// NotificationSender interface for sending notifications
type NotificationSender interface {
	Send(ctx context.Context, notification Notification) error
}

type activeEscalation struct {
	event        *CoordinationEvent
	protocol     *Protocol
	currentLevel int
	startedAt    time.Time
	nextCheck    time.Time
	timer        *time.Timer
}

// EscalationConfig holds escalation configuration
type EscalationConfig struct {
	// Default escalation timeouts per priority
	TimeoutByPriority map[Priority]time.Duration

	// Default escalation targets per level
	DefaultTargets map[int][]string

	// Check interval for escalation
	CheckInterval time.Duration

	// Maximum escalation level
	MaxLevel int

	// Enable automatic escalation
	AutoEscalate bool
}

// DefaultEscalationConfig returns sensible defaults
func DefaultEscalationConfig() EscalationConfig {
	return EscalationConfig{
		TimeoutByPriority: map[Priority]time.Duration{
			PriorityCritical: 15 * time.Minute,
			PriorityUrgent:   30 * time.Minute,
			PriorityHigh:     2 * time.Hour,
			PriorityNormal:   8 * time.Hour,
			PriorityLow:      24 * time.Hour,
		},
		DefaultTargets: map[int][]string{
			1: {"assigned_worker"},
			2: {"supervisor"},
			3: {"department_head"},
			4: {"agency_director"},
		},
		CheckInterval: 1 * time.Minute,
		MaxLevel:      4,
		AutoEscalate:  true,
	}
}

// NewEscalationService creates a new escalation service
func NewEscalationService(notifier NotificationSender, config EscalationConfig) *EscalationService {
	return &EscalationService{
		notifier: notifier,
		active:   make(map[string]*activeEscalation),
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the escalation monitoring loop
func (s *EscalationService) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			s.checkEscalations(ctx)
		}
	}
}

// Stop stops the escalation service
func (s *EscalationService) Stop() {
	close(s.stopCh)
}

// RegisterEvent registers an event for escalation tracking
func (s *EscalationService) RegisterEvent(event *CoordinationEvent, protocol *Protocol) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	timeout := s.getTimeout(event.Priority, protocol)

	s.active[event.ID] = &activeEscalation{
		event:        event,
		protocol:     protocol,
		currentLevel: 0,
		startedAt:    time.Now(),
		nextCheck:    time.Now().Add(timeout),
	}
}

// AcknowledgeEvent marks an event as acknowledged
func (s *EscalationService) AcknowledgeEvent(eventID string, ack Acknowledge) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	if active, ok := s.active[eventID]; ok {
		active.event.Status = EventStatusAcknowledged
		active.event.Acknowledged = append(active.event.Acknowledged, ack)

		// If all target agencies acknowledged, remove from active
		if s.allAcknowledged(active) {
			delete(s.active, eventID)
		}
	}
}

// ResolveEvent marks an event as resolved
func (s *EscalationService) ResolveEvent(eventID string) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()

	if active, ok := s.active[eventID]; ok {
		active.event.Status = EventStatusResolved
		delete(s.active, eventID)
	}
}

// checkEscalations checks all active events for escalation
func (s *EscalationService) checkEscalations(ctx context.Context) {
	s.activeMu.Lock()
	toEscalate := make([]*activeEscalation, 0)

	now := time.Now()
	for _, active := range s.active {
		if now.After(active.nextCheck) && s.config.AutoEscalate {
			toEscalate = append(toEscalate, active)
		}
	}
	s.activeMu.Unlock()

	// Escalate outside the lock
	for _, active := range toEscalate {
		s.escalate(ctx, active)
	}
}

// escalate escalates an event to the next level
func (s *EscalationService) escalate(ctx context.Context, active *activeEscalation) {
	s.activeMu.Lock()

	// Check if already at max level
	if active.currentLevel >= s.config.MaxLevel {
		active.event.Status = EventStatusExpired
		delete(s.active, active.event.ID)
		s.activeMu.Unlock()
		return
	}

	// Increase level
	active.currentLevel++
	active.event.Status = EventStatusEscalated

	// Get next timeout
	timeout := s.getTimeoutForLevel(active.event.Priority, active.currentLevel, active.protocol)
	active.nextCheck = time.Now().Add(timeout)

	event := active.event
	level := active.currentLevel
	protocol := active.protocol

	s.activeMu.Unlock()

	// Send escalation notifications
	targets := s.getTargetsForLevel(level, protocol)
	for _, target := range targets {
		notification := s.createEscalationNotification(event, level, target)
		if err := s.notifier.Send(ctx, notification); err != nil {
			// Log error but continue
			fmt.Printf("Failed to send escalation notification: %v\n", err)
		}
	}
}

// getTimeout returns the timeout for an event
func (s *EscalationService) getTimeout(priority Priority, protocol *Protocol) time.Duration {
	if protocol != nil && protocol.Timeout > 0 {
		return protocol.Timeout
	}
	if timeout, ok := s.config.TimeoutByPriority[priority]; ok {
		return timeout
	}
	return s.config.TimeoutByPriority[PriorityNormal]
}

// getTimeoutForLevel returns the timeout for a specific escalation level
func (s *EscalationService) getTimeoutForLevel(priority Priority, level int, protocol *Protocol) time.Duration {
	// Check protocol-specific timeout
	if protocol != nil && protocol.Escalation != nil {
		for _, l := range protocol.Escalation.Levels {
			if l.Level == level && l.Timeout > 0 {
				return l.Timeout
			}
		}
	}

	// Decrease timeout at each level (more urgent)
	baseTimeout := s.getTimeout(priority, protocol)
	divisor := float64(level + 1)
	return time.Duration(float64(baseTimeout) / divisor)
}

// getTargetsForLevel returns notification targets for an escalation level
func (s *EscalationService) getTargetsForLevel(level int, protocol *Protocol) []string {
	// Check protocol-specific targets
	if protocol != nil && protocol.Escalation != nil {
		for _, l := range protocol.Escalation.Levels {
			if l.Level == level && len(l.Targets) > 0 {
				return l.Targets
			}
		}
	}

	// Use default targets
	if targets, ok := s.config.DefaultTargets[level]; ok {
		return targets
	}

	return []string{"supervisor"}
}

// allAcknowledged checks if all target agencies have acknowledged
func (s *EscalationService) allAcknowledged(active *activeEscalation) bool {
	if len(active.event.TargetAgencies) == 0 {
		return len(active.event.Acknowledged) > 0
	}

	acked := make(map[string]bool)
	for _, ack := range active.event.Acknowledged {
		acked[ack.AgencyCode] = true
	}

	for _, target := range active.event.TargetAgencies {
		if !acked[target] {
			return false
		}
	}
	return true
}

// createEscalationNotification creates a notification for escalation
func (s *EscalationService) createEscalationNotification(event *CoordinationEvent, level int, target string) Notification {
	levelNames := map[int]string{
		1: "Level 1 - Worker",
		2: "Level 2 - Supervisor",
		3: "Level 3 - Department Head",
		4: "Level 4 - Agency Director",
	}

	levelName := levelNames[level]
	if levelName == "" {
		levelName = fmt.Sprintf("Level %d", level)
	}

	subject := fmt.Sprintf("[ESCALATION %s] %s", levelName, event.Title)
	body := fmt.Sprintf(`Event escalated to %s

Event: %s
Type: %s
Priority: %s
Subject: %s

This event has not been acknowledged within the required timeframe.
Immediate attention is required.

Escalation History:
- Level %d escalation at %s
- Original event created: %s

Please acknowledge this event immediately.`,
		levelName,
		event.Title,
		event.Type,
		event.Priority,
		event.SubjectJMBG,
		level,
		time.Now().Format(time.RFC3339),
		event.CreatedAt.Format(time.RFC3339),
	)

	return Notification{
		ID:        fmt.Sprintf("esc-%s-%d-%d", event.ID, level, time.Now().Unix()),
		EventID:   event.ID,
		Type:      "push",
		Priority:  event.Priority,
		Subject:   subject,
		Body:      body,
		Recipient: Recipient{
			Type: "role",
			ID:   target,
		},
		ScheduledAt: time.Now(),
		Status:      "pending",
		Data: map[string]any{
			"event_id":         event.ID,
			"escalation_level": level,
			"event_type":       string(event.Type),
			"subject_jmbg":     event.SubjectJMBG,
		},
	}
}

// GetActiveEscalations returns all active escalations
func (s *EscalationService) GetActiveEscalations() []EscalationInfo {
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()

	result := make([]EscalationInfo, 0, len(s.active))
	for _, active := range s.active {
		result = append(result, EscalationInfo{
			EventID:      active.event.ID,
			EventType:    active.event.Type,
			Priority:     active.event.Priority,
			CurrentLevel: active.currentLevel,
			StartedAt:    active.startedAt,
			NextCheck:    active.nextCheck,
			SubjectJMBG:  active.event.SubjectJMBG,
		})
	}
	return result
}

// EscalationInfo contains info about an active escalation
type EscalationInfo struct {
	EventID      string    `json:"event_id"`
	EventType    EventType `json:"event_type"`
	Priority     Priority  `json:"priority"`
	CurrentLevel int       `json:"current_level"`
	StartedAt    time.Time `json:"started_at"`
	NextCheck    time.Time `json:"next_check"`
	SubjectJMBG  string    `json:"subject_jmbg"`
}
