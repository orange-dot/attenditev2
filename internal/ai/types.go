package ai

import "time"

// SeverityLevel represents the severity of an anomaly
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityWarning  SeverityLevel = "warning"
	SeverityCritical SeverityLevel = "critical"
)

// AnomalyType represents the type of anomaly detected
type AnomalyType string

const (
	AnomalyTypeImpossibleInstruction AnomalyType = "impossible_instruction"
	AnomalyTypeLogicalInconsistency  AnomalyType = "logical_inconsistency"
	AnomalyTypeDataConflict          AnomalyType = "data_conflict"
	AnomalyTypeProtocolViolation     AnomalyType = "protocol_violation"
)

// AnalysisRequest represents a request to analyze a document
type AnalysisRequest struct {
	DocumentText   string            `json:"document_text"`
	DocumentType   string            `json:"document_type,omitempty"`
	PatientContext map[string]any    `json:"patient_context,omitempty"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type              AnomalyType   `json:"type"`
	Severity          SeverityLevel `json:"severity"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Evidence          []string      `json:"evidence"`
	Recommendation    string        `json:"recommendation"`
	ProtocolReference *string       `json:"protocol_reference,omitempty"`
}

// AnalysisResponse represents the response from AI analysis
type AnalysisResponse struct {
	RequestID        string    `json:"request_id"`
	Timestamp        time.Time `json:"timestamp"`
	AnomaliesFound   int       `json:"anomalies_found"`
	Anomalies        []Anomaly `json:"anomalies"`
	ProcessingTimeMs int       `json:"processing_time_ms"`
	ModelUsed        string    `json:"model_used"`
	Confidence       float64   `json:"confidence"`
}

// Example represents a test example document
type Example struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	DocumentText    string  `json:"document_text"`
	ExpectedAnomaly *string `json:"expected_anomaly,omitempty"`
}

// ExamplesResponse represents the response containing test examples
type ExamplesResponse struct {
	Examples []Example `json:"examples"`
}
