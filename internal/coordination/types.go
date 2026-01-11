package coordination

import (
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
	"github.com/serbia-gov/platform/internal/adapters/social"
)

// EventType represents the type of coordination event
type EventType string

const (
	EventTypeAdmission        EventType = "admission"
	EventTypeDischarge        EventType = "discharge"
	EventTypeEmergency        EventType = "emergency"
	EventTypeSocialAlert      EventType = "social_alert"
	EventTypeChildProtection  EventType = "child_protection"
	EventTypeDomesticViolence EventType = "domestic_violence"
	EventTypeVulnerablePerson EventType = "vulnerable_person"
)

// Priority levels for coordination events
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityUrgent   Priority = "urgent"
	PriorityCritical Priority = "critical"
)

// CoordinationEvent represents an event that requires cross-agency coordination
type CoordinationEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Priority  Priority  `json:"priority"`
	Timestamp time.Time `json:"timestamp"`

	// Subject information
	SubjectJMBG string `json:"subject_jmbg"`
	SubjectName string `json:"subject_name,omitempty"`

	// Source information
	SourceSystem    string `json:"source_system"`
	SourceAgency    string `json:"source_agency"`
	SourceReference string `json:"source_reference,omitempty"`

	// Event details
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Details     map[string]any    `json:"details,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Enrichment data (populated by enrichment service)
	Enrichment *EventEnrichment `json:"enrichment,omitempty"`

	// Routing information
	TargetAgencies []string `json:"target_agencies,omitempty"`

	// Status tracking
	Status       EventStatus   `json:"status"`
	Acknowledged []Acknowledge `json:"acknowledged,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// EventStatus represents the status of a coordination event
type EventStatus string

const (
	EventStatusPending      EventStatus = "pending"
	EventStatusRouted       EventStatus = "routed"
	EventStatusAcknowledged EventStatus = "acknowledged"
	EventStatusInProgress   EventStatus = "in_progress"
	EventStatusResolved     EventStatus = "resolved"
	EventStatusEscalated    EventStatus = "escalated"
	EventStatusExpired      EventStatus = "expired"
)

// Acknowledge represents an acknowledgment from an agency
type Acknowledge struct {
	AgencyCode   string    `json:"agency_code"`
	AgencyName   string    `json:"agency_name"`
	WorkerID     string    `json:"worker_id,omitempty"`
	WorkerName   string    `json:"worker_name,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	Notes        string    `json:"notes,omitempty"`
}

// EventEnrichment contains cross-system context for an event
type EventEnrichment struct {
	// Health context
	HealthContext *health.HealthContext `json:"health_context,omitempty"`

	// Social context
	SocialContext *social.SocialContext `json:"social_context,omitempty"`

	// Risk assessment
	RiskLevel       string   `json:"risk_level,omitempty"`
	RiskScore       int      `json:"risk_score,omitempty"`
	RiskFactors     []string `json:"risk_factors,omitempty"`
	VulnerableFlags []string `json:"vulnerable_flags,omitempty"`

	// Related persons
	FamilyMembers []FamilyMemberInfo `json:"family_members,omitempty"`

	// Related cases
	RelatedCases []RelatedCase `json:"related_cases,omitempty"`

	// Recommendations
	RecommendedActions []string `json:"recommended_actions,omitempty"`

	// Enrichment metadata
	EnrichedAt time.Time `json:"enriched_at"`
	Sources    []string  `json:"sources,omitempty"`
}

// FamilyMemberInfo contains basic info about a family member
type FamilyMemberInfo struct {
	JMBG         string `json:"jmbg"`
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	Age          int    `json:"age,omitempty"`
	IsMinor      bool   `json:"is_minor"`
	HasOpenCase  bool   `json:"has_open_case"`
	RiskLevel    string `json:"risk_level,omitempty"`
}

// RelatedCase contains info about a related case
type RelatedCase struct {
	CaseID     string    `json:"case_id"`
	CaseType   string    `json:"case_type"`
	Agency     string    `json:"agency"`
	Status     string    `json:"status"`
	Priority   string    `json:"priority"`
	OpenedAt   time.Time `json:"opened_at"`
	AssignedTo string    `json:"assigned_to,omitempty"`
}

// Protocol represents a coordination protocol
type Protocol struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	TriggerType EventType     `json:"trigger_type"`
	Conditions  []Condition   `json:"conditions"`
	Actions     []Action      `json:"actions"`
	Escalation  *Escalation   `json:"escalation,omitempty"`
	Timeout     time.Duration `json:"timeout"`
	IsActive    bool          `json:"is_active"`
}

// Condition represents a condition for triggering a protocol
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, lt, contains, in
	Value    any    `json:"value"`
}

// Action represents an action in a protocol
type Action struct {
	Type       string         `json:"type"` // notify, create_case, assign, escalate
	Target     string         `json:"target,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Delay      time.Duration  `json:"delay,omitempty"`
}

// Escalation represents escalation rules
type Escalation struct {
	Levels    []EscalationLevel `json:"levels"`
	MaxLevel  int               `json:"max_level"`
}

// EscalationLevel represents a single escalation level
type EscalationLevel struct {
	Level        int           `json:"level"`
	Timeout      time.Duration `json:"timeout"`
	Targets      []string      `json:"targets"`
	Actions      []Action      `json:"actions"`
	Notification string        `json:"notification"`
}

// Notification represents a notification to be sent
type Notification struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	Type        string    `json:"type"` // push, sms, email, in_app
	Priority    Priority  `json:"priority"`
	Recipient   Recipient `json:"recipient"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Data        map[string]any `json:"data,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	Status      string    `json:"status"`
}

// Recipient represents a notification recipient
type Recipient struct {
	Type       string `json:"type"` // agency, worker, group
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	DeviceToken string `json:"device_token,omitempty"`
}

// CoordinationStats represents coordination statistics
type CoordinationStats struct {
	Period             string         `json:"period"`
	TotalEvents        int64          `json:"total_events"`
	EventsByType       map[string]int64 `json:"events_by_type"`
	EventsByPriority   map[string]int64 `json:"events_by_priority"`
	EventsByStatus     map[string]int64 `json:"events_by_status"`
	AverageResponseTime time.Duration  `json:"average_response_time"`
	EscalationRate     float64        `json:"escalation_rate"`
	AcknowledgmentRate float64        `json:"acknowledgment_rate"`
}
