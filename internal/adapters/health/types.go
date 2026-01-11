package health

import (
	"time"
)

// PatientRecord represents a patient from the health system
type PatientRecord struct {
	// Identifiers
	LocalID string `json:"local_id"`          // ID in source system
	JMBG    string `json:"jmbg"`              // Unique citizen number
	LBO     string `json:"lbo,omitempty"`     // Health insurance number

	// Demographics
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	MiddleName  string     `json:"middle_name,omitempty"`
	DateOfBirth time.Time  `json:"date_of_birth"`
	Gender      Gender     `json:"gender"`
	Deceased    bool       `json:"deceased"`
	DeceasedAt  *time.Time `json:"deceased_at,omitempty"`

	// Contact
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`

	// Insurance
	InsuranceStatus string `json:"insurance_status,omitempty"`
	InsuranceType   string `json:"insurance_type,omitempty"`

	// Metadata
	SourceSystem      string    `json:"source_system"`
	SourceInstitution string    `json:"source_institution"`
	LastUpdated       time.Time `json:"last_updated"`
}

// Gender represents patient gender
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderOther   Gender = "other"
	GenderUnknown Gender = "unknown"
)

// Hospitalization represents a hospital stay
type Hospitalization struct {
	ID                string     `json:"id"`
	PatientJMBG       string     `json:"patient_jmbg"`
	AdmissionDate     time.Time  `json:"admission_date"`
	DischargeDate     *time.Time `json:"discharge_date,omitempty"`
	Department        string     `json:"department"`
	DepartmentCode    string     `json:"department_code,omitempty"`
	Room              string     `json:"room,omitempty"`
	Bed               string     `json:"bed,omitempty"`
	AdmissionType     string     `json:"admission_type"` // emergency, planned, transfer
	DischargeType     string     `json:"discharge_type,omitempty"`
	AttendingDoctor   string     `json:"attending_doctor,omitempty"`
	AttendingDoctorID string     `json:"attending_doctor_id,omitempty"`

	// Diagnoses
	PrimaryDiagnosis    *Diagnosis  `json:"primary_diagnosis,omitempty"`
	SecondaryDiagnoses  []Diagnosis `json:"secondary_diagnoses,omitempty"`
	DischargeDiagnosis  *Diagnosis  `json:"discharge_diagnosis,omitempty"`

	// Procedures
	Procedures []Procedure `json:"procedures,omitempty"`

	// Status
	Status string `json:"status"` // active, discharged, transferred

	// Metadata
	SourceSystem      string    `json:"source_system"`
	SourceInstitution string    `json:"source_institution"`
	LastUpdated       time.Time `json:"last_updated"`
}

// Diagnosis represents a medical diagnosis
type Diagnosis struct {
	ID          string     `json:"id,omitempty"`
	ICD10Code   string     `json:"icd10_code"`
	Description string     `json:"description"`
	Type        string     `json:"type"` // primary, secondary, admission, discharge
	DiagnosedAt time.Time  `json:"diagnosed_at"`
	DiagnosedBy string     `json:"diagnosed_by,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

// Procedure represents a medical procedure
type Procedure struct {
	ID          string    `json:"id,omitempty"`
	Code        string    `json:"code"` // Procedure code
	CodeSystem  string    `json:"code_system,omitempty"`
	Description string    `json:"description"`
	PerformedAt time.Time `json:"performed_at"`
	PerformedBy string    `json:"performed_by,omitempty"`
	Department  string    `json:"department,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

// LabResult represents a laboratory test result
type LabResult struct {
	ID            string    `json:"id"`
	PatientJMBG   string    `json:"patient_jmbg"`
	TestCode      string    `json:"test_code"`
	TestName      string    `json:"test_name"`
	LOINCCode     string    `json:"loinc_code,omitempty"`
	Value         string    `json:"value"`
	Unit          string    `json:"unit,omitempty"`
	ReferenceMin  string    `json:"reference_min,omitempty"`
	ReferenceMax  string    `json:"reference_max,omitempty"`
	Interpretation string   `json:"interpretation,omitempty"` // normal, low, high, critical
	CollectedAt   time.Time `json:"collected_at"`
	ReportedAt    time.Time `json:"reported_at"`
	OrderedBy     string    `json:"ordered_by,omitempty"`
	Laboratory    string    `json:"laboratory,omitempty"`
	Notes         string    `json:"notes,omitempty"`

	// Metadata
	SourceSystem      string    `json:"source_system"`
	SourceInstitution string    `json:"source_institution"`
	LastUpdated       time.Time `json:"last_updated"`
}

// Prescription represents a medication prescription
type Prescription struct {
	ID              string     `json:"id"`
	PatientJMBG     string     `json:"patient_jmbg"`
	MedicationName  string     `json:"medication_name"`
	MedicationCode  string     `json:"medication_code,omitempty"`
	ATCCode         string     `json:"atc_code,omitempty"`
	Dosage          string     `json:"dosage"`
	DosageUnit      string     `json:"dosage_unit,omitempty"`
	Frequency       string     `json:"frequency"`
	Route           string     `json:"route,omitempty"` // oral, iv, im, etc.
	Duration        string     `json:"duration,omitempty"`
	Quantity        int        `json:"quantity,omitempty"`
	Refills         int        `json:"refills,omitempty"`
	PrescribedAt    time.Time  `json:"prescribed_at"`
	PrescribedBy    string     `json:"prescribed_by"`
	PrescribedByID  string     `json:"prescribed_by_id,omitempty"`
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
	DispensedAt     *time.Time `json:"dispensed_at,omitempty"`
	DispensedBy     string     `json:"dispensed_by,omitempty"`
	Status          string     `json:"status"` // active, dispensed, expired, cancelled
	Instructions    string     `json:"instructions,omitempty"`
	DiagnosisICD10  string     `json:"diagnosis_icd10,omitempty"`
	IsChronicMed    bool       `json:"is_chronic_med"`

	// Metadata
	SourceSystem      string    `json:"source_system"`
	SourceInstitution string    `json:"source_institution"`
	LastUpdated       time.Time `json:"last_updated"`
}

// HealthContext aggregates health data for a person
type HealthContext struct {
	PatientJMBG string `json:"patient_jmbg"`

	// Current patient record
	PatientRecord *PatientRecord `json:"patient,omitempty"`

	// Recent clinical data
	Hospitalizations []Hospitalization `json:"recent_hospitalizations,omitempty"`
	Prescriptions    []Prescription    `json:"active_prescriptions,omitempty"`
	LabResults       []LabResult       `json:"recent_lab_results,omitempty"`
	Diagnoses        []Diagnosis       `json:"recent_diagnoses,omitempty"`

	// Summary flags
	HasRecentHospitalization bool `json:"has_recent_hospitalization"`
	HasChronicCondition      bool `json:"has_chronic_condition"`
	HasActiveTreatment       bool `json:"has_active_treatment"`
	IsCurrentlyHospitalized  bool `json:"is_currently_hospitalized"`
	RequiresContinuousCare   bool `json:"requires_continuous_care"`

	// Timestamps
	CollectedAt time.Time `json:"collected_at"`
}

// NewHealthContext creates a new health context for a patient
func NewHealthContext(jmbg string) *HealthContext {
	return &HealthContext{
		PatientJMBG:      jmbg,
		Hospitalizations: make([]Hospitalization, 0),
		Prescriptions:    make([]Prescription, 0),
		LabResults:       make([]LabResult, 0),
		Diagnoses:        make([]Diagnosis, 0),
		CollectedAt:      time.Now(),
	}
}

// UpdateFlags updates summary flags based on collected data
func (hc *HealthContext) UpdateFlags() {
	hc.HasRecentHospitalization = len(hc.Hospitalizations) > 0
	hc.HasActiveTreatment = len(hc.Prescriptions) > 0

	// Check if currently hospitalized
	for _, h := range hc.Hospitalizations {
		if h.DischargeDate == nil || h.Status == "active" {
			hc.IsCurrentlyHospitalized = true
			break
		}
	}

	// Check for chronic conditions (prescriptions marked as chronic)
	chronicCount := 0
	for _, p := range hc.Prescriptions {
		if p.IsChronicMed {
			hc.HasChronicCondition = true
			chronicCount++
		}
	}

	// Requires continuous care if multiple chronic medications or currently hospitalized
	hc.RequiresContinuousCare = chronicCount >= 3 || hc.IsCurrentlyHospitalized
}
