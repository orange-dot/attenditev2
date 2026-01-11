package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
	"github.com/serbia-gov/platform/internal/adapters/social"
)

// Service is the main coordination service
type Service struct {
	// Dependencies
	healthAdapter health.Adapter
	socialAdapter social.Adapter

	// Sub-services
	enrichment *EnrichmentService
	escalation *EscalationService
	protocol   *ProtocolEngine
	notifier   NotificationSender

	// Event processing
	eventCh chan *CoordinationEvent
	workers int

	// State
	mu       sync.RWMutex
	events   map[string]*CoordinationEvent
	stats    *CoordinationStats

	// Lifecycle
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	// Worker pool size
	Workers int

	// Event channel buffer size
	EventBufferSize int

	// Enrichment config
	Enrichment EnrichmentConfig

	// Escalation config
	Escalation EscalationConfig
}

// DefaultServiceConfig returns sensible defaults
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Workers:         4,
		EventBufferSize: 1000,
		Enrichment:      DefaultEnrichmentConfig(),
		Escalation:      DefaultEscalationConfig(),
	}
}

// NewService creates a new coordination service
func NewService(
	healthAdapter health.Adapter,
	socialAdapter social.Adapter,
	notifier NotificationSender,
	config ServiceConfig,
) *Service {
	enrichment := NewEnrichmentService(healthAdapter, socialAdapter, config.Enrichment)
	escalation := NewEscalationService(notifier, config.Escalation)

	return &Service{
		healthAdapter: healthAdapter,
		socialAdapter: socialAdapter,
		enrichment:    enrichment,
		escalation:    escalation,
		protocol:      NewProtocolEngine(enrichment, escalation, notifier),
		notifier:      notifier,
		eventCh:       make(chan *CoordinationEvent, config.EventBufferSize),
		workers:       config.Workers,
		events:        make(map[string]*CoordinationEvent),
		stats:         &CoordinationStats{},
		stopCh:        make(chan struct{}),
	}
}

// Start starts the coordination service
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("service already started")
	}
	s.started = true
	s.mu.Unlock()

	// Start worker pool
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// Start escalation service
	go s.escalation.Start(ctx)

	// Start health event subscriber if available
	if s.healthAdapter != nil {
		go s.subscribeHealthEvents(ctx)
	}

	return nil
}

// Stop stops the coordination service
func (s *Service) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return fmt.Errorf("service not started")
	}
	s.mu.Unlock()

	close(s.stopCh)
	s.escalation.Stop()
	s.wg.Wait()

	return nil
}

// SubmitEvent submits an event for processing
func (s *Service) SubmitEvent(event *CoordinationEvent) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event.UpdatedAt = time.Now()
	event.Status = EventStatusPending

	// Store event
	s.mu.Lock()
	s.events[event.ID] = event
	s.stats.TotalEvents++
	s.mu.Unlock()

	// Submit for processing
	select {
	case s.eventCh <- event:
		return nil
	default:
		return fmt.Errorf("event buffer full")
	}
}

// GetEvent returns an event by ID
func (s *Service) GetEvent(id string) (*CoordinationEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.events[id]
	return event, ok
}

// AcknowledgeEvent acknowledges an event
func (s *Service) AcknowledgeEvent(eventID string, ack Acknowledge) error {
	s.mu.Lock()
	event, ok := s.events[eventID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("event not found: %s", eventID)
	}

	ack.AcknowledgedAt = time.Now()
	event.Acknowledged = append(event.Acknowledged, ack)
	event.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Update escalation
	s.escalation.AcknowledgeEvent(eventID, ack)

	return nil
}

// ResolveEvent marks an event as resolved
func (s *Service) ResolveEvent(eventID string) error {
	s.mu.Lock()
	event, ok := s.events[eventID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("event not found: %s", eventID)
	}

	event.Status = EventStatusResolved
	event.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Update escalation
	s.escalation.ResolveEvent(eventID)

	return nil
}

// worker processes events from the channel
func (s *Service) worker(ctx context.Context, id int) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case event := <-s.eventCh:
			s.processEvent(ctx, event)
		}
	}
}

// processEvent processes a single event
func (s *Service) processEvent(ctx context.Context, event *CoordinationEvent) {
	start := time.Now()

	// Process through protocol engine
	if err := s.protocol.ProcessEvent(ctx, event); err != nil {
		fmt.Printf("Failed to process event %s: %v\n", event.ID, err)
	}

	// Update stats
	s.mu.Lock()
	if s.stats.EventsByType == nil {
		s.stats.EventsByType = make(map[string]int64)
	}
	if s.stats.EventsByPriority == nil {
		s.stats.EventsByPriority = make(map[string]int64)
	}
	if s.stats.EventsByStatus == nil {
		s.stats.EventsByStatus = make(map[string]int64)
	}

	s.stats.EventsByType[string(event.Type)]++
	s.stats.EventsByPriority[string(event.Priority)]++
	s.stats.EventsByStatus[string(event.Status)]++

	// Update average response time
	elapsed := time.Since(start)
	totalEvents := s.stats.TotalEvents
	if totalEvents > 0 {
		currentAvg := s.stats.AverageResponseTime
		s.stats.AverageResponseTime = (currentAvg*time.Duration(totalEvents-1) + elapsed) / time.Duration(totalEvents)
	}
	s.mu.Unlock()
}

// subscribeHealthEvents subscribes to health adapter events
func (s *Service) subscribeHealthEvents(ctx context.Context) {
	// Subscribe to admissions
	if err := s.healthAdapter.SubscribeAdmissions(ctx, s.handleAdmission); err != nil {
		fmt.Printf("Failed to subscribe to admissions: %v\n", err)
	}

	// Subscribe to discharges
	if err := s.healthAdapter.SubscribeDischarges(ctx, s.handleDischarge); err != nil {
		fmt.Printf("Failed to subscribe to discharges: %v\n", err)
	}
}

// handleAdmission handles hospital admission events
func (s *Service) handleAdmission(event health.AdmissionEvent) {
	coordEvent := &CoordinationEvent{
		ID:           generateEventID(),
		Type:         EventTypeAdmission,
		Priority:     s.determinePriority(event),
		Timestamp:    event.Timestamp,
		SubjectJMBG:  event.PatientJMBG,
		SubjectName:  event.PatientName,
		SourceSystem: event.SourceSystem,
		SourceAgency: event.SourceInst,
		Title:        fmt.Sprintf("Hospital Admission: %s", event.PatientName),
		Description: fmt.Sprintf("Patient admitted to %s, Department: %s, Type: %s",
			event.SourceInst, event.Department, event.AdmissionType),
		Details: map[string]any{
			"department":     event.Department,
			"admission_type": event.AdmissionType,
			"diagnosis_icd":  event.DiagnosisICD,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    EventStatusPending,
	}

	if err := s.SubmitEvent(coordEvent); err != nil {
		fmt.Printf("Failed to submit admission event: %v\n", err)
	}
}

// handleDischarge handles hospital discharge events
func (s *Service) handleDischarge(event health.DischargeEvent) {
	coordEvent := &CoordinationEvent{
		ID:           generateEventID(),
		Type:         EventTypeDischarge,
		Priority:     PriorityNormal,
		Timestamp:    event.Timestamp,
		SubjectJMBG:  event.PatientJMBG,
		SubjectName:  event.PatientName,
		SourceSystem: event.SourceSystem,
		SourceAgency: event.SourceInst,
		Title:        fmt.Sprintf("Hospital Discharge: %s", event.PatientName),
		Description: fmt.Sprintf("Patient discharged from %s, Department: %s, Type: %s",
			event.SourceInst, event.Department, event.DischargeType),
		Details: map[string]any{
			"department":       event.Department,
			"discharge_type":   event.DischargeType,
			"diagnosis_icd":    event.DiagnosisICD,
			"follow_up_needed": event.FollowUpNeeded,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    EventStatusPending,
	}

	// Higher priority if follow-up required
	if event.FollowUpNeeded {
		coordEvent.Priority = PriorityHigh
	}

	if err := s.SubmitEvent(coordEvent); err != nil {
		fmt.Printf("Failed to submit discharge event: %v\n", err)
	}
}

// determinePriority determines event priority based on admission type
func (s *Service) determinePriority(event health.AdmissionEvent) Priority {
	switch event.AdmissionType {
	case "emergency":
		return PriorityUrgent
	case "trauma":
		return PriorityCritical
	default:
		return PriorityNormal
	}
}

// RegisterProtocol registers a coordination protocol
func (s *Service) RegisterProtocol(protocol *Protocol) error {
	return s.protocol.RegisterProtocol(protocol)
}

// GetStats returns coordination statistics
func (s *Service) GetStats() CoordinationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.stats
}

// GetActiveEscalations returns active escalations
func (s *Service) GetActiveEscalations() []EscalationInfo {
	return s.escalation.GetActiveEscalations()
}

// CreateChildProtectionEvent creates a child protection event
func (s *Service) CreateChildProtectionEvent(
	ctx context.Context,
	childJMBG string,
	childName string,
	concern string,
	sourceAgency string,
	reportedBy string,
) error {
	event := &CoordinationEvent{
		ID:           generateEventID(),
		Type:         EventTypeChildProtection,
		Priority:     PriorityCritical,
		Timestamp:    time.Now(),
		SubjectJMBG:  childJMBG,
		SubjectName:  childName,
		SourceSystem: "coordination",
		SourceAgency: sourceAgency,
		Title:        fmt.Sprintf("Child Protection Concern: %s", childName),
		Description:  concern,
		Details: map[string]any{
			"concern_type": "child_protection",
			"reported_by":  reportedBy,
		},
		Metadata: map[string]string{
			"reported_by": reportedBy,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    EventStatusPending,
	}

	return s.SubmitEvent(event)
}

// CreateDomesticViolenceEvent creates a domestic violence event
func (s *Service) CreateDomesticViolenceEvent(
	ctx context.Context,
	victimJMBG string,
	victimName string,
	concern string,
	sourceAgency string,
	reportedBy string,
) error {
	event := &CoordinationEvent{
		ID:           generateEventID(),
		Type:         EventTypeDomesticViolence,
		Priority:     PriorityCritical,
		Timestamp:    time.Now(),
		SubjectJMBG:  victimJMBG,
		SubjectName:  victimName,
		SourceSystem: "coordination",
		SourceAgency: sourceAgency,
		Title:        fmt.Sprintf("Domestic Violence Alert: %s", victimName),
		Description:  concern,
		Details: map[string]any{
			"concern_type": "domestic_violence",
			"reported_by":  reportedBy,
		},
		Metadata: map[string]string{
			"reported_by": reportedBy,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    EventStatusPending,
	}

	return s.SubmitEvent(event)
}

// CreateVulnerablePersonEvent creates a vulnerable person event
func (s *Service) CreateVulnerablePersonEvent(
	ctx context.Context,
	personJMBG string,
	personName string,
	vulnerabilityType string,
	concern string,
	sourceAgency string,
) error {
	priority := PriorityHigh
	if vulnerabilityType == "elderly_alone" || vulnerabilityType == "disabled_without_support" {
		priority = PriorityUrgent
	}

	event := &CoordinationEvent{
		ID:           generateEventID(),
		Type:         EventTypeVulnerablePerson,
		Priority:     priority,
		Timestamp:    time.Now(),
		SubjectJMBG:  personJMBG,
		SubjectName:  personName,
		SourceSystem: "coordination",
		SourceAgency: sourceAgency,
		Title:        fmt.Sprintf("Vulnerable Person: %s", personName),
		Description:  concern,
		Details: map[string]any{
			"vulnerability_type": vulnerabilityType,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    EventStatusPending,
	}

	return s.SubmitEvent(event)
}

// Helper functions

func generateEventID() string {
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}
