package social

import (
	"time"
)

// BeneficiaryStatus represents social assistance status from Socijalna Karta
type BeneficiaryStatus struct {
	JMBG string `json:"jmbg"`

	// Benefits
	ReceivesCashAssistance bool       `json:"receives_cash_assistance"`
	CashAssistanceAmount   float64    `json:"cash_assistance_amount,omitempty"`
	CashAssistanceSince    *time.Time `json:"cash_assistance_since,omitempty"`
	CashAssistanceUntil    *time.Time `json:"cash_assistance_until,omitempty"`

	ReceivesChildAllowance   bool       `json:"receives_child_allowance"`
	ChildAllowanceAmount     float64    `json:"child_allowance_amount,omitempty"`
	ChildAllowanceSince      *time.Time `json:"child_allowance_since,omitempty"`

	ReceivesDisabilityBenefit bool       `json:"receives_disability_benefit"`
	DisabilityBenefitAmount   float64    `json:"disability_benefit_amount,omitempty"`
	DisabilityBenefitSince    *time.Time `json:"disability_benefit_since,omitempty"`

	ReceivesElderlyCare      bool       `json:"receives_elderly_care"`
	ElderlyCareType          string     `json:"elderly_care_type,omitempty"`
	ElderlyCareSince         *time.Time `json:"elderly_care_since,omitempty"`

	// Status flags
	IsEmployed           bool   `json:"is_employed"`
	EmploymentStatus     string `json:"employment_status,omitempty"`
	IsRetired            bool   `json:"is_retired"`
	HasHealthInsurance   bool   `json:"has_health_insurance"`
	HealthInsuranceType  string `json:"health_insurance_type,omitempty"`

	// Risk indicators
	IsAtRisk           bool   `json:"is_at_risk"`
	RiskLevel          string `json:"risk_level,omitempty"` // low, medium, high, critical
	RiskFactors        []string `json:"risk_factors,omitempty"`

	// Metadata
	LastUpdated  time.Time `json:"last_updated"`
	DataSource   string    `json:"data_source"`
}

// FamilyUnit represents family composition from Socijalna Karta
type FamilyUnit struct {
	HouseholdID   string         `json:"household_id"`
	HeadOfFamily  string         `json:"head_of_family_jmbg"`
	Address       string         `json:"address"`
	Municipality  string         `json:"municipality"`
	Members       []FamilyMember `json:"members"`
	TotalIncome   float64        `json:"total_income,omitempty"`
	IncomePerCapita float64      `json:"income_per_capita,omitempty"`
	HousingType   string         `json:"housing_type,omitempty"`
	HousingStatus string         `json:"housing_status,omitempty"` // owned, rented, social_housing
	LastUpdated   time.Time      `json:"last_updated"`
}

// FamilyMember represents a family member
type FamilyMember struct {
	JMBG           string     `json:"jmbg"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	DateOfBirth    time.Time  `json:"date_of_birth"`
	Relationship   string     `json:"relationship"` // head, spouse, child, parent, other
	Gender         string     `json:"gender"`
	IsEmployed     bool       `json:"is_employed"`
	IsStudent      bool       `json:"is_student"`
	HasDisability  bool       `json:"has_disability"`
	DisabilityType string     `json:"disability_type,omitempty"`
	Income         float64    `json:"income,omitempty"`
}

// PropertyData represents property records
type PropertyData struct {
	JMBG             string     `json:"jmbg"`
	OwnsRealEstate   bool       `json:"owns_real_estate"`
	Properties       []Property `json:"properties,omitempty"`
	OwnsVehicle      bool       `json:"owns_vehicle"`
	Vehicles         []Vehicle  `json:"vehicles,omitempty"`
	HasSavings       bool       `json:"has_savings"`
	SavingsRange     string     `json:"savings_range,omitempty"` // categorical, not exact
	LastUpdated      time.Time  `json:"last_updated"`
	DataSource       string     `json:"data_source"`
}

// Property represents real estate
type Property struct {
	Type         string  `json:"type"` // apartment, house, land, commercial
	Location     string  `json:"location"`
	SizeM2       float64 `json:"size_m2,omitempty"`
	OwnershipPct float64 `json:"ownership_pct"`
	Value        float64 `json:"value,omitempty"`
	Encumbered   bool    `json:"encumbered"` // has mortgage or other burden
}

// Vehicle represents a registered vehicle
type Vehicle struct {
	Type              string `json:"type"` // car, motorcycle, truck
	Brand             string `json:"brand,omitempty"`
	YearOfManufacture int    `json:"year_of_manufacture,omitempty"`
	OwnershipPct      float64 `json:"ownership_pct"`
}

// IncomeData represents income information
type IncomeData struct {
	JMBG             string      `json:"jmbg"`
	Sources          []IncomeSource `json:"sources"`
	TotalMonthly     float64     `json:"total_monthly"`
	TotalYearly      float64     `json:"total_yearly"`
	LastUpdated      time.Time   `json:"last_updated"`
	DataSource       string      `json:"data_source"`
}

// IncomeSource represents a source of income
type IncomeSource struct {
	Type         string  `json:"type"` // salary, pension, benefit, rental, other
	Amount       float64 `json:"amount"`
	Frequency    string  `json:"frequency"` // monthly, yearly
	Employer     string  `json:"employer,omitempty"`
	Since        *time.Time `json:"since,omitempty"`
}

// SocialCase represents a case from CSR/SOZIS
type SocialCase struct {
	ID           string    `json:"id"`
	CaseNumber   string    `json:"case_number"`
	ClientJMBG   string    `json:"client_jmbg"`
	ClientName   string    `json:"client_name"`
	CaseType     string    `json:"case_type"`
	Category     string    `json:"category"` // child_protection, elderly_care, disability, poverty, etc.
	Status       string    `json:"status"`   // open, in_progress, closed, suspended
	Priority     string    `json:"priority"` // low, normal, high, urgent, critical

	// Assignment
	CSRCode        string `json:"csr_code"`
	CSRName        string `json:"csr_name"`
	AssignedWorker string `json:"assigned_worker,omitempty"`
	WorkerID       string `json:"worker_id,omitempty"`
	SupervisorID   string `json:"supervisor_id,omitempty"`

	// Timeline
	OpenedAt      time.Time  `json:"opened_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	NextReviewAt  *time.Time `json:"next_review_at,omitempty"`

	// Details
	Description    string   `json:"description,omitempty"`
	Services       []string `json:"services,omitempty"`
	RiskLevel      string   `json:"risk_level,omitempty"`
	RiskFactors    []string `json:"risk_factors,omitempty"`
	Notes          string   `json:"notes,omitempty"`

	// Related persons
	FamilyMembers []string `json:"family_members_jmbg,omitempty"`

	// Metadata
	SourceSystem string    `json:"source_system"`
	LastUpdated  time.Time `json:"last_updated"`
}

// RiskAssessment represents a risk assessment for a person
type RiskAssessment struct {
	JMBG            string     `json:"jmbg"`
	AssessmentID    string     `json:"assessment_id"`
	AssessmentDate  time.Time  `json:"assessment_date"`
	AssessedBy      string     `json:"assessed_by"`

	// Overall risk
	OverallRisk     string     `json:"overall_risk"` // low, medium, high, critical
	RiskScore       int        `json:"risk_score,omitempty"` // 0-100

	// Risk categories
	ChildSafetyRisk      string `json:"child_safety_risk,omitempty"`
	DomesticViolenceRisk string `json:"domestic_violence_risk,omitempty"`
	SelfHarmRisk         string `json:"self_harm_risk,omitempty"`
	NeglectRisk          string `json:"neglect_risk,omitempty"`
	PovertyRisk          string `json:"poverty_risk,omitempty"`
	HomelessnessRisk     string `json:"homelessness_risk,omitempty"`
	SubstanceAbuseRisk   string `json:"substance_abuse_risk,omitempty"`
	MentalHealthRisk     string `json:"mental_health_risk,omitempty"`

	// Contributing factors
	RiskFactors     []RiskFactor `json:"risk_factors,omitempty"`
	ProtectiveFactors []string   `json:"protective_factors,omitempty"`

	// Recommendations
	RecommendedActions []string `json:"recommended_actions,omitempty"`
	RequiresImmediate  bool     `json:"requires_immediate_action"`

	// Metadata
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
	SourceSystem string     `json:"source_system"`
	LastUpdated  time.Time  `json:"last_updated"`
}

// RiskFactor represents a specific risk factor
type RiskFactor struct {
	Category    string `json:"category"`
	Factor      string `json:"factor"`
	Severity    string `json:"severity"` // low, medium, high
	Description string `json:"description,omitempty"`
}

// SocialContext aggregates social data for a person
type SocialContext struct {
	PersonJMBG string `json:"person_jmbg"`

	// Core data
	BeneficiaryStatus *BeneficiaryStatus `json:"beneficiary_status,omitempty"`
	FamilyUnit        *FamilyUnit        `json:"family_unit,omitempty"`
	PropertyData      *PropertyData      `json:"property_data,omitempty"`
	IncomeData        *IncomeData        `json:"income_data,omitempty"`

	// Case data
	OpenCases    []SocialCase    `json:"open_cases,omitempty"`
	CaseHistory  []SocialCase    `json:"case_history,omitempty"`
	RiskAssessment *RiskAssessment `json:"risk_assessment,omitempty"`

	// Summary flags
	IsBeneficiary          bool   `json:"is_beneficiary"`
	HasOpenCases           bool   `json:"has_open_cases"`
	IsAtRisk               bool   `json:"is_at_risk"`
	RiskLevel              string `json:"risk_level,omitempty"`
	RequiresImmediateAction bool  `json:"requires_immediate_action"`

	// Timestamps
	CollectedAt time.Time `json:"collected_at"`
}

// NewSocialContext creates a new social context for a person
func NewSocialContext(jmbg string) *SocialContext {
	return &SocialContext{
		PersonJMBG:  jmbg,
		OpenCases:   make([]SocialCase, 0),
		CaseHistory: make([]SocialCase, 0),
		CollectedAt: time.Now(),
	}
}

// UpdateFlags updates summary flags based on collected data
func (sc *SocialContext) UpdateFlags() {
	sc.IsBeneficiary = sc.BeneficiaryStatus != nil &&
		(sc.BeneficiaryStatus.ReceivesCashAssistance ||
			sc.BeneficiaryStatus.ReceivesChildAllowance ||
			sc.BeneficiaryStatus.ReceivesDisabilityBenefit ||
			sc.BeneficiaryStatus.ReceivesElderlyCare)

	sc.HasOpenCases = len(sc.OpenCases) > 0

	if sc.RiskAssessment != nil {
		sc.IsAtRisk = sc.RiskAssessment.OverallRisk != "low"
		sc.RiskLevel = sc.RiskAssessment.OverallRisk
		sc.RequiresImmediateAction = sc.RiskAssessment.RequiresImmediate
	} else if sc.BeneficiaryStatus != nil {
		sc.IsAtRisk = sc.BeneficiaryStatus.IsAtRisk
		sc.RiskLevel = sc.BeneficiaryStatus.RiskLevel
	}
}
