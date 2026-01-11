package privacy

import (
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// PseudonymizedSubject replaces Citizen in the central system.
// It contains no PII - only pseudonymized identifier and aggregated demographics.
type PseudonymizedSubject struct {
	PseudonymID        PseudonymID `json:"pseudonym_id"`
	AgeRange           string      `json:"age_range"`            // "0-10", "11-17", "18-30", "31-50", "51-65", "65+"
	Municipality       string      `json:"municipality"`         // Oblast only, not exact address
	IsMinor            bool        `json:"is_minor"`
	SourceFacilityCode string      `json:"source_facility_code"` // For routing de-pseudonymization requests
	Gender             string      `json:"gender,omitempty"`     // "M", "F", or empty if not relevant
}

// NewPseudonymizedSubject creates a new pseudonymized subject from raw data.
func NewPseudonymizedSubject(
	pseudonymID PseudonymID,
	birthDate time.Time,
	municipality string,
	facilityCode string,
	gender string,
) *PseudonymizedSubject {
	return &PseudonymizedSubject{
		PseudonymID:        pseudonymID,
		AgeRange:           calculateAgeRange(birthDate),
		Municipality:       municipality,
		IsMinor:            isMinor(birthDate),
		SourceFacilityCode: facilityCode,
		Gender:             gender,
	}
}

// PseudonymizedParticipant replaces Participant with CitizenID in the central system.
// Contains no name, email, phone - only pseudonymized reference and role.
type PseudonymizedParticipant struct {
	PseudonymID        *PseudonymID `json:"pseudonym_id,omitempty"`
	Role               string       `json:"role"`
	SourceFacilityCode string       `json:"source_facility_code,omitempty"`
	// Name, ContactEmail, ContactPhone are intentionally excluded
}

// PseudonymizedFamilyMember replaces FamilyMemberInfo in the central system.
// Contains pseudonymized reference and aggregated data, no JMBG or name.
type PseudonymizedFamilyMember struct {
	PseudonymID  PseudonymID `json:"pseudonym_id"`
	Relationship string      `json:"relationship"` // parent, child, spouse, sibling, etc.
	AgeRange     string      `json:"age_range"`
	IsMinor      bool        `json:"is_minor"`
	HasOpenCase  bool        `json:"has_open_case"`
	RiskLevel    string      `json:"risk_level,omitempty"` // low, medium, high, critical
	// Name, JMBG are intentionally excluded
}

// PseudonymizedHealthContext replaces HealthContext for the central system.
// Contains only aggregated health indicators, no patient records or diagnoses.
type PseudonymizedHealthContext struct {
	PseudonymID              PseudonymID `json:"pseudonym_id"`
	HasRecentHospitalization bool        `json:"has_recent_hospitalization"`
	HasChronicCondition      bool        `json:"has_chronic_condition"`
	IsCurrentlyHospitalized  bool        `json:"is_currently_hospitalized"`
	LastContactDaysAgo       int         `json:"last_contact_days_ago,omitempty"`
	SourceFacilityCode       string      `json:"source_facility_code"`
	// PatientRecord, diagnoses, prescriptions are intentionally excluded
}

// NewPseudonymizedHealthContext creates a pseudonymized health context.
func NewPseudonymizedHealthContext(
	pseudonymID PseudonymID,
	hasRecentHospitalization bool,
	hasChronicCondition bool,
	isCurrentlyHospitalized bool,
	lastContactDaysAgo int,
	facilityCode string,
) *PseudonymizedHealthContext {
	return &PseudonymizedHealthContext{
		PseudonymID:              pseudonymID,
		HasRecentHospitalization: hasRecentHospitalization,
		HasChronicCondition:      hasChronicCondition,
		IsCurrentlyHospitalized:  isCurrentlyHospitalized,
		LastContactDaysAgo:       lastContactDaysAgo,
		SourceFacilityCode:       facilityCode,
	}
}

// PseudonymizedSocialContext replaces SocialContext for the central system.
// Contains only aggregated social welfare indicators, no detailed beneficiary data.
type PseudonymizedSocialContext struct {
	PseudonymID             PseudonymID `json:"pseudonym_id"`
	IsBeneficiary           bool        `json:"is_beneficiary"`
	HasOpenCases            bool        `json:"has_open_cases"`
	OpenCaseCount           int         `json:"open_case_count,omitempty"`
	RiskLevel               string      `json:"risk_level,omitempty"`
	RequiresImmediateAction bool        `json:"requires_immediate_action"`
	SourceFacilityCode      string      `json:"source_facility_code"`
	// BeneficiaryStatus details, FamilyUnit with JMBGs are intentionally excluded
}

// NewPseudonymizedSocialContext creates a pseudonymized social context.
func NewPseudonymizedSocialContext(
	pseudonymID PseudonymID,
	isBeneficiary bool,
	hasOpenCases bool,
	openCaseCount int,
	riskLevel string,
	requiresImmediateAction bool,
	facilityCode string,
) *PseudonymizedSocialContext {
	return &PseudonymizedSocialContext{
		PseudonymID:             pseudonymID,
		IsBeneficiary:           isBeneficiary,
		HasOpenCases:            hasOpenCases,
		OpenCaseCount:           openCaseCount,
		RiskLevel:               riskLevel,
		RequiresImmediateAction: requiresImmediateAction,
		SourceFacilityCode:      facilityCode,
	}
}

// PseudonymizedEnrichment is the privacy-safe version of enrichment data
// that gets sent to the central system.
type PseudonymizedEnrichment struct {
	Subject       *PseudonymizedSubject       `json:"subject"`
	FamilyMembers []PseudonymizedFamilyMember `json:"family_members,omitempty"`
	HealthContext *PseudonymizedHealthContext `json:"health_context,omitempty"`
	SocialContext *PseudonymizedSocialContext `json:"social_context,omitempty"`
	EnrichedAt    time.Time                   `json:"enriched_at"`
	AccessLevel   DataAccessLevel             `json:"access_level"`
}

// PseudonymizedCase represents a case with pseudonymized participant data.
type PseudonymizedCase struct {
	ID           types.ID                   `json:"id"`
	CaseType     string                     `json:"case_type"`
	Status       string                     `json:"status"`
	Priority     string                     `json:"priority,omitempty"`
	RiskLevel    string                     `json:"risk_level,omitempty"`
	Participants []PseudonymizedParticipant `json:"participants"`
	AgencyID     types.ID                   `json:"agency_id"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	// Description, internal notes, etc. may contain PII - handle carefully
}

// AggregatedStatistics contains only statistical data for AI analysis.
// This is the default level of access for AI systems.
type AggregatedStatistics struct {
	TotalCases       int                   `json:"total_cases"`
	CasesByType      map[string]int        `json:"cases_by_type"`
	CasesByStatus    map[string]int        `json:"cases_by_status"`
	CasesByPriority  map[string]int        `json:"cases_by_priority"`
	CasesByRiskLevel map[string]int        `json:"cases_by_risk_level"`
	CasesByAgency    map[types.ID]int      `json:"cases_by_agency"`
	CasesByRegion    map[string]int        `json:"cases_by_region"`
	AgeDistribution  map[string]int        `json:"age_distribution"`
	TimeRange        TimeRange             `json:"time_range"`
	GeneratedAt      time.Time             `json:"generated_at"`
}

// TimeRange represents a time period for statistics.
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// calculateAgeRange returns an age range string from birthdate.
func calculateAgeRange(birthDate time.Time) string {
	if birthDate.IsZero() {
		return "unknown"
	}

	age := calculateAge(birthDate)

	switch {
	case age < 0:
		return "unknown"
	case age <= 10:
		return "0-10"
	case age <= 17:
		return "11-17"
	case age <= 30:
		return "18-30"
	case age <= 50:
		return "31-50"
	case age <= 65:
		return "51-65"
	default:
		return "65+"
	}
}

// calculateAge returns the age in years from a birthdate.
func calculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred this year
	if now.YearDay() < birthDate.YearDay() {
		years--
	}

	return years
}

// isMinor returns true if the person is under 18.
func isMinor(birthDate time.Time) bool {
	if birthDate.IsZero() {
		return false
	}
	return calculateAge(birthDate) < 18
}

// ConvertToAgeRange converts an exact age to age range.
func ConvertToAgeRange(age int) string {
	switch {
	case age < 0:
		return "unknown"
	case age <= 10:
		return "0-10"
	case age <= 17:
		return "11-17"
	case age <= 30:
		return "18-30"
	case age <= 50:
		return "31-50"
	case age <= 65:
		return "51-65"
	default:
		return "65+"
	}
}

// RiskLevel constants for standardized risk assessment.
const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// RelationshipType constants for family relationships.
const (
	RelationshipParent   = "parent"
	RelationshipChild    = "child"
	RelationshipSpouse   = "spouse"
	RelationshipSibling  = "sibling"
	RelationshipGuardian = "guardian"
	RelationshipOther    = "other"
)
