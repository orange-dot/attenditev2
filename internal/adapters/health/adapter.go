package health

import (
	"context"
	"time"
)

// Adapter defines the interface for health system adapters.
// Implementations connect to specific HIS systems (Heliant, InfoMedis, etc.)
// and provide a unified API for the platform.
type Adapter interface {
	// Patient data retrieval
	FetchPatientRecord(ctx context.Context, jmbg string) (*PatientRecord, error)
	FetchPatientByLBO(ctx context.Context, lbo string) (*PatientRecord, error)

	// Clinical data retrieval
	FetchHospitalizations(ctx context.Context, jmbg string, from, to time.Time) ([]Hospitalization, error)
	FetchLabResults(ctx context.Context, jmbg string, from, to time.Time) ([]LabResult, error)
	FetchPrescriptions(ctx context.Context, jmbg string, activeOnly bool) ([]Prescription, error)
	FetchDiagnoses(ctx context.Context, jmbg string, from, to time.Time) ([]Diagnosis, error)

	// Real-time event subscriptions
	SubscribeAdmissions(ctx context.Context, handler AdmissionHandler) error
	SubscribeDischarges(ctx context.Context, handler DischargeHandler) error
	SubscribeEmergencies(ctx context.Context, handler EmergencyHandler) error

	// Adapter metadata
	SourceSystem() string
	SourceInstitution() string
	IsConnected() bool

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

// AdmissionHandler is called when a new patient admission is detected
type AdmissionHandler func(event AdmissionEvent)

// DischargeHandler is called when a patient is discharged
type DischargeHandler func(event DischargeEvent)

// EmergencyHandler is called for emergency cases
type EmergencyHandler func(event EmergencyEvent)

// AdmissionEvent represents a patient admission
type AdmissionEvent struct {
	EventID       string    `json:"event_id"`
	Timestamp     time.Time `json:"timestamp"`
	PatientJMBG   string    `json:"patient_jmbg"`
	PatientName   string    `json:"patient_name"`
	Department    string    `json:"department"`
	AdmissionType string    `json:"admission_type"` // emergency, planned, transfer
	DiagnosisICD  string    `json:"diagnosis_icd,omitempty"`
	SourceSystem  string    `json:"source_system"`
	SourceInst    string    `json:"source_institution"`
}

// DischargeEvent represents a patient discharge
type DischargeEvent struct {
	EventID        string     `json:"event_id"`
	Timestamp      time.Time  `json:"timestamp"`
	PatientJMBG    string     `json:"patient_jmbg"`
	PatientName    string     `json:"patient_name"`
	Department     string     `json:"department"`
	DischargeType  string     `json:"discharge_type"` // home, transfer, deceased
	AdmissionDate  time.Time  `json:"admission_date"`
	DischargeDate  time.Time  `json:"discharge_date"`
	DiagnosisICD   string     `json:"diagnosis_icd,omitempty"`
	FollowUpNeeded bool       `json:"follow_up_needed"`
	FollowUpDate   *time.Time `json:"follow_up_date,omitempty"`
	SourceSystem   string     `json:"source_system"`
	SourceInst     string     `json:"source_institution"`
}

// EmergencyEvent represents an emergency case
type EmergencyEvent struct {
	EventID       string    `json:"event_id"`
	Timestamp     time.Time `json:"timestamp"`
	PatientJMBG   string    `json:"patient_jmbg"`
	PatientName   string    `json:"patient_name"`
	EmergencyType string    `json:"emergency_type"` // trauma, cardiac, psychiatric, etc.
	Severity      string    `json:"severity"`       // critical, urgent, standard
	Description   string    `json:"description,omitempty"`
	SourceSystem  string    `json:"source_system"`
	SourceInst    string    `json:"source_institution"`
}

// Config holds common configuration for health adapters
type Config struct {
	// Database connection
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode"`

	// Institution info
	InstitutionCode string `json:"institution_code"`
	InstitutionName string `json:"institution_name"`

	// Polling configuration
	PollInterval    time.Duration `json:"poll_interval"`
	BatchSize       int           `json:"batch_size"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	ConnectionRetry time.Duration `json:"connection_retry"`

	// Event publishing
	EventBufferSize int `json:"event_buffer_size"`
}

// DefaultConfig returns default adapter configuration
func DefaultConfig() Config {
	return Config{
		Port:            1433, // SQL Server default
		SSLMode:         "disable",
		PollInterval:    30 * time.Second,
		BatchSize:       100,
		RetryAttempts:   3,
		RetryDelay:      5 * time.Second,
		ConnectionRetry: 30 * time.Second,
		EventBufferSize: 1000,
	}
}
