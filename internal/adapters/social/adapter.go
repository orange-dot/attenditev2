package social

import (
	"context"
	"time"
)

// Adapter defines the interface for social protection system adapters.
// Implementations connect to SOZIS, Socijalna Karta, or other social systems.
type Adapter interface {
	// Beneficiary data from Socijalna Karta
	FetchBeneficiaryStatus(ctx context.Context, jmbg string) (*BeneficiaryStatus, error)
	FetchFamilyComposition(ctx context.Context, jmbg string) (*FamilyUnit, error)
	FetchPropertyData(ctx context.Context, jmbg string) (*PropertyData, error)
	FetchIncomeData(ctx context.Context, jmbg string) (*IncomeData, error)

	// CSR case data (from SOZIS)
	FetchOpenCases(ctx context.Context, jmbg string) ([]SocialCase, error)
	FetchCaseHistory(ctx context.Context, jmbg string) ([]SocialCase, error)
	FetchRiskAssessment(ctx context.Context, jmbg string) (*RiskAssessment, error)

	// Real-time subscriptions
	SubscribeCaseUpdates(ctx context.Context, handler CaseUpdateHandler) error
	SubscribeEmergencyInterventions(ctx context.Context, handler InterventionHandler) error

	// Notifications to CSR
	NotifyCSR(ctx context.Context, agencyCode string, notification Notification) error
	RequestIntervention(ctx context.Context, request InterventionRequest) (*InterventionResponse, error)

	// Adapter metadata
	SourceSystem() string
	IsConnected() bool

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

// CaseUpdateHandler is called when a CSR case is updated
type CaseUpdateHandler func(event CaseUpdateEvent)

// InterventionHandler is called for emergency interventions
type InterventionHandler func(event InterventionEvent)

// CaseUpdateEvent represents a case update from SOZIS
type CaseUpdateEvent struct {
	EventID      string    `json:"event_id"`
	Timestamp    time.Time `json:"timestamp"`
	CaseID       string    `json:"case_id"`
	CaseNumber   string    `json:"case_number"`
	ClientJMBG   string    `json:"client_jmbg"`
	UpdateType   string    `json:"update_type"` // created, updated, closed, escalated
	CSRCode      string    `json:"csr_code"`
	WorkerID     string    `json:"worker_id,omitempty"`
	Description  string    `json:"description,omitempty"`
	SourceSystem string    `json:"source_system"`
}

// InterventionEvent represents an emergency intervention
type InterventionEvent struct {
	EventID           string    `json:"event_id"`
	Timestamp         time.Time `json:"timestamp"`
	InterventionID    string    `json:"intervention_id"`
	ClientJMBG        string    `json:"client_jmbg"`
	InterventionType  string    `json:"intervention_type"` // domestic_violence, child_protection, elder_abuse
	Priority          string    `json:"priority"`          // critical, urgent, standard
	CSRCode           string    `json:"csr_code"`
	RequestedBy       string    `json:"requested_by"` // police, hospital, citizen
	RequestedBySystem string    `json:"requested_by_system,omitempty"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	SourceSystem      string    `json:"source_system"`
}

// Notification represents a notification to CSR
type Notification struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // info, warning, urgent, critical
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	ClientJMBG  string                 `json:"client_jmbg,omitempty"`
	CaseID      string                 `json:"case_id,omitempty"`
	Source      string                 `json:"source"`
	SourceType  string                 `json:"source_type"` // hospital, police, platform
	Data        map[string]interface{} `json:"data,omitempty"`
	RequiresAck bool                   `json:"requires_ack"`
	AckDeadline *time.Time             `json:"ack_deadline,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// InterventionRequest represents a request for CSR intervention
type InterventionRequest struct {
	ID               string                 `json:"id"`
	ClientJMBG       string                 `json:"client_jmbg"`
	InterventionType string                 `json:"intervention_type"`
	Priority         string                 `json:"priority"`
	RequestedBy      string                 `json:"requested_by"`
	RequestedByType  string                 `json:"requested_by_type"` // hospital, police, citizen
	Description      string                 `json:"description"`
	Location         string                 `json:"location,omitempty"`
	ContactPhone     string                 `json:"contact_phone,omitempty"`
	Data             map[string]interface{} `json:"data,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

// InterventionResponse represents the response to an intervention request
type InterventionResponse struct {
	RequestID      string    `json:"request_id"`
	InterventionID string    `json:"intervention_id"`
	Status         string    `json:"status"` // accepted, rejected, pending
	AssignedCSR    string    `json:"assigned_csr,omitempty"`
	AssignedWorker string    `json:"assigned_worker,omitempty"`
	EstimatedTime  string    `json:"estimated_time,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Config holds configuration for social adapters
type Config struct {
	// Socijalna Karta API
	SocialCardURL    string `json:"social_card_url"`
	SocialCardCert   string `json:"social_card_cert"`
	SocialCardKey    string `json:"social_card_key"`
	SocialCardCACert string `json:"social_card_ca_cert"`

	// SOZIS connection (if direct DB access)
	SOZISHost     string `json:"sozis_host,omitempty"`
	SOZISPort     int    `json:"sozis_port,omitempty"`
	SOZISDatabase string `json:"sozis_database,omitempty"`
	SOZISUser     string `json:"sozis_user,omitempty"`
	SOZISPassword string `json:"sozis_password,omitempty"`

	// CSR identification
	CSRCode string `json:"csr_code"`
	CSRName string `json:"csr_name"`

	// Polling configuration
	PollInterval  time.Duration `json:"poll_interval"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`
	Timeout       time.Duration `json:"timeout"`

	// Event publishing
	EventBufferSize int `json:"event_buffer_size"`
}

// DefaultConfig returns default social adapter configuration
func DefaultConfig() Config {
	return Config{
		PollInterval:    30 * time.Second,
		RetryAttempts:   3,
		RetryDelay:      5 * time.Second,
		Timeout:         30 * time.Second,
		EventBufferSize: 1000,
	}
}
