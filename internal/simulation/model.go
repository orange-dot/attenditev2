package simulation

import (
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// SimulationStep represents a step in the simulation
type SimulationStep struct {
	StepID        string            `json:"step_id"`
	FromInstitution string          `json:"from_institution"`
	ToInstitution   string          `json:"to_institution"`
	Action          string          `json:"action"`
	Description     string          `json:"description"`
	DataExchanged   []string        `json:"data_exchanged"`
	IsResponse      bool            `json:"is_response"`
	Metadata        map[string]any  `json:"metadata,omitempty"`
}

// SimulationRequest represents a request to run a simulation step
type SimulationRequest struct {
	UseCaseID     string          `json:"use_case_id"`
	UseCaseTitle  string          `json:"use_case_title"`
	Step          SimulationStep  `json:"step"`
	SessionID     string          `json:"session_id"`
	CitizenJMBG   string          `json:"citizen_jmbg,omitempty"`
}

// SimulationResponse represents the response from a simulation step
type SimulationResponse struct {
	Success       bool      `json:"success"`
	AuditEntryID  string    `json:"audit_entry_id"`
	Timestamp     time.Time `json:"timestamp"`
	Message       string    `json:"message"`
}

// SimulationSession represents an active simulation session
type SimulationSession struct {
	ID            types.ID  `json:"id"`
	UseCaseID     string    `json:"use_case_id"`
	UseCaseTitle  string    `json:"use_case_title"`
	StartedAt     time.Time `json:"started_at"`
	CurrentStep   int       `json:"current_step"`
	TotalSteps    int       `json:"total_steps"`
	CitizenJMBG   string    `json:"citizen_jmbg,omitempty"`
}

// Predefined institutions for simulation
var Institutions = map[string]Institution{
	"citizen": {
		ID:   "citizen",
		Name: "Građanin",
		Type: "citizen",
		City: "",
	},
	"geronto-kikinda": {
		ID:   "geronto-kikinda",
		Name: "Gerontološki centar Kikinda",
		Type: "local",
		City: "Kikinda",
	},
	"csr-kikinda": {
		ID:   "csr-kikinda",
		Name: "Centar za socijalni rad Kikinda",
		Type: "local",
		City: "Kikinda",
	},
	"mup-kikinda": {
		ID:   "mup-kikinda",
		Name: "Policijska stanica Kikinda",
		Type: "local",
		City: "Kikinda",
	},
	"mup-srbije": {
		ID:   "mup-srbije",
		Name: "MUP Srbije - Centralni registar",
		Type: "central",
		City: "Beograd",
	},
	"poreska": {
		ID:   "poreska",
		Name: "Poreska uprava Srbije",
		Type: "central",
		City: "Beograd",
	},
	"katastar": {
		ID:   "katastar",
		Name: "Republički geodetski zavod",
		Type: "central",
		City: "Beograd",
	},
	"data-centar": {
		ID:   "data-centar",
		Name: "Data centar Vlade Srbije",
		Type: "datacenter",
		City: "Kragujevac",
	},
}

// Institution represents a government institution
type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // local, central, datacenter
	City string `json:"city"`
}

// Event types for simulation
const (
	EventSimulationStarted     = "simulation.started"
	EventSimulationStepExecuted = "simulation.step_executed"
	EventSimulationCompleted   = "simulation.completed"
	EventDataRequest           = "simulation.data_request"
	EventDataResponse          = "simulation.data_response"
)
