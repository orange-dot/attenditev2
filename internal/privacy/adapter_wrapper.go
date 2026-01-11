package privacy

import (
	"context"
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
	"github.com/serbia-gov/platform/internal/adapters/social"
)

// PrivacyAwareHealthAdapter wraps a health adapter and returns pseudonymized data.
// This adapter runs at the LOCAL facility level and handles the conversion
// between real JMBG (local) and PseudonymID (for central system).
type PrivacyAwareHealthAdapter struct {
	underlying    health.Adapter
	pseudoSvc     *PseudonymizationService
	facilityCode  string
	accessControl *AIAccessController
}

// NewPrivacyAwareHealthAdapter creates a new privacy-aware health adapter.
func NewPrivacyAwareHealthAdapter(
	underlying health.Adapter,
	pseudoSvc *PseudonymizationService,
	accessControl *AIAccessController,
) *PrivacyAwareHealthAdapter {
	return &PrivacyAwareHealthAdapter{
		underlying:    underlying,
		pseudoSvc:     pseudoSvc,
		facilityCode:  pseudoSvc.FacilityCode(),
		accessControl: accessControl,
	}
}

// FetchPseudonymizedHealthContext fetches health data and returns it in pseudonymized form.
// This is what the CENTRAL system receives - no JMBG, names, or addresses.
func (a *PrivacyAwareHealthAdapter) FetchPseudonymizedHealthContext(
	ctx context.Context,
	jmbg string,
) (*PseudonymizedHealthContext, error) {
	// First, get or create pseudonym
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	// Fetch real data from underlying adapter
	record, err := a.underlying.FetchPatientRecord(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch patient record: %w", err)
	}

	// Fetch hospitalizations
	now := time.Now()
	hospitalizations, _ := a.underlying.FetchHospitalizations(ctx, jmbg, now.AddDate(0, -6, 0), now)

	// Fetch prescriptions for chronic condition check
	prescriptions, _ := a.underlying.FetchPrescriptions(ctx, jmbg, true)

	// Calculate summary flags
	hasRecentHospitalization := len(hospitalizations) > 0
	hasChronicCondition := false
	isCurrentlyHospitalized := false
	lastContactDaysAgo := -1

	for _, p := range prescriptions {
		if p.IsChronicMed {
			hasChronicCondition = true
			break
		}
	}

	for _, h := range hospitalizations {
		if h.DischargeDate == nil || h.Status == "active" {
			isCurrentlyHospitalized = true
		}
		// Calculate last contact
		if lastContactDaysAgo < 0 || int(now.Sub(h.AdmissionDate).Hours()/24) < lastContactDaysAgo {
			lastContactDaysAgo = int(now.Sub(h.AdmissionDate).Hours() / 24)
		}
	}

	// Check record last updated for last contact
	if record != nil && !record.LastUpdated.IsZero() {
		recordDaysAgo := int(now.Sub(record.LastUpdated).Hours() / 24)
		if lastContactDaysAgo < 0 || recordDaysAgo < lastContactDaysAgo {
			lastContactDaysAgo = recordDaysAgo
		}
	}

	return NewPseudonymizedHealthContext(
		pseudonymID,
		hasRecentHospitalization,
		hasChronicCondition,
		isCurrentlyHospitalized,
		lastContactDaysAgo,
		a.facilityCode,
	), nil
}

// FetchPseudonymizedPatientInfo fetches patient info in pseudonymized form.
// Returns only age range and municipality - no name, address, phone, email.
func (a *PrivacyAwareHealthAdapter) FetchPseudonymizedPatientInfo(
	ctx context.Context,
	jmbg string,
) (*PseudonymizedSubject, error) {
	// Get or create pseudonym
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	// Fetch real record
	record, err := a.underlying.FetchPatientRecord(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch patient record: %w", err)
	}

	// Extract only non-PII data
	gender := ""
	if record.Gender != "" {
		switch record.Gender {
		case health.GenderMale:
			gender = "M"
		case health.GenderFemale:
			gender = "F"
		}
	}

	return NewPseudonymizedSubject(
		pseudonymID,
		record.DateOfBirth,
		record.City, // Use city as municipality approximation
		a.facilityCode,
		gender,
	), nil
}

// GetPseudonymID returns the pseudonym for a JMBG without fetching any data.
func (a *PrivacyAwareHealthAdapter) GetPseudonymID(ctx context.Context, jmbg string) (PseudonymID, error) {
	return a.pseudoSvc.Pseudonymize(ctx, jmbg)
}

// PseudonymizeAdmissionEvent converts an admission event to pseudonymized form.
func (a *PrivacyAwareHealthAdapter) PseudonymizeAdmissionEvent(
	ctx context.Context,
	event health.AdmissionEvent,
) (*PseudonymizedAdmissionEvent, error) {
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, event.PatientJMBG)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	return &PseudonymizedAdmissionEvent{
		EventID:            event.EventID,
		Timestamp:          event.Timestamp,
		PseudonymID:        pseudonymID,
		Department:         event.Department,
		AdmissionType:      event.AdmissionType,
		DiagnosisCategory:  extractDiagnosisCategory(event.DiagnosisICD),
		SourceFacilityCode: a.facilityCode,
	}, nil
}

// PseudonymizeDischargeEvent converts a discharge event to pseudonymized form.
func (a *PrivacyAwareHealthAdapter) PseudonymizeDischargeEvent(
	ctx context.Context,
	event health.DischargeEvent,
) (*PseudonymizedDischargeEvent, error) {
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, event.PatientJMBG)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	return &PseudonymizedDischargeEvent{
		EventID:            event.EventID,
		Timestamp:          event.Timestamp,
		PseudonymID:        pseudonymID,
		Department:         event.Department,
		DischargeType:      event.DischargeType,
		LengthOfStayDays:   int(event.DischargeDate.Sub(event.AdmissionDate).Hours() / 24),
		DiagnosisCategory:  extractDiagnosisCategory(event.DiagnosisICD),
		FollowUpNeeded:     event.FollowUpNeeded,
		SourceFacilityCode: a.facilityCode,
	}, nil
}

// Underlying returns the underlying adapter for local facility use only.
// This should NEVER be exposed to the central system.
func (a *PrivacyAwareHealthAdapter) Underlying() health.Adapter {
	return a.underlying
}

// PrivacyAwareSocialAdapter wraps a social adapter and returns pseudonymized data.
type PrivacyAwareSocialAdapter struct {
	underlying    social.Adapter
	pseudoSvc     *PseudonymizationService
	facilityCode  string
	accessControl *AIAccessController
}

// NewPrivacyAwareSocialAdapter creates a new privacy-aware social adapter.
func NewPrivacyAwareSocialAdapter(
	underlying social.Adapter,
	pseudoSvc *PseudonymizationService,
	accessControl *AIAccessController,
) *PrivacyAwareSocialAdapter {
	return &PrivacyAwareSocialAdapter{
		underlying:    underlying,
		pseudoSvc:     pseudoSvc,
		facilityCode:  pseudoSvc.FacilityCode(),
		accessControl: accessControl,
	}
}

// FetchPseudonymizedSocialContext fetches social data and returns it in pseudonymized form.
func (a *PrivacyAwareSocialAdapter) FetchPseudonymizedSocialContext(
	ctx context.Context,
	jmbg string,
) (*PseudonymizedSocialContext, error) {
	// Get or create pseudonym
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	// Fetch beneficiary status
	status, _ := a.underlying.FetchBeneficiaryStatus(ctx, jmbg)

	// Fetch open cases
	openCases, _ := a.underlying.FetchOpenCases(ctx, jmbg)

	// Fetch risk assessment
	riskAssessment, _ := a.underlying.FetchRiskAssessment(ctx, jmbg)

	// Calculate summary flags
	isBeneficiary := status != nil &&
		(status.ReceivesCashAssistance ||
			status.ReceivesChildAllowance ||
			status.ReceivesDisabilityBenefit ||
			status.ReceivesElderlyCare)

	hasOpenCases := len(openCases) > 0
	openCaseCount := len(openCases)

	riskLevel := ""
	requiresImmediateAction := false

	if riskAssessment != nil {
		riskLevel = riskAssessment.OverallRisk
		requiresImmediateAction = riskAssessment.RequiresImmediate
	} else if status != nil && status.IsAtRisk {
		riskLevel = status.RiskLevel
	}

	return NewPseudonymizedSocialContext(
		pseudonymID,
		isBeneficiary,
		hasOpenCases,
		openCaseCount,
		riskLevel,
		requiresImmediateAction,
		a.facilityCode,
	), nil
}

// FetchPseudonymizedFamilyMembers fetches family members in pseudonymized form.
func (a *PrivacyAwareSocialAdapter) FetchPseudonymizedFamilyMembers(
	ctx context.Context,
	jmbg string,
) ([]PseudonymizedFamilyMember, error) {
	// Fetch family composition
	familyUnit, err := a.underlying.FetchFamilyComposition(ctx, jmbg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch family composition: %w", err)
	}

	if familyUnit == nil || len(familyUnit.Members) == 0 {
		return nil, nil
	}

	// Pseudonymize each family member
	result := make([]PseudonymizedFamilyMember, 0, len(familyUnit.Members))

	for _, member := range familyUnit.Members {
		// Pseudonymize member's JMBG
		memberPseudonym, err := a.pseudoSvc.Pseudonymize(ctx, member.JMBG)
		if err != nil {
			continue // Skip members we can't pseudonymize
		}

		// Check for open cases for this family member
		memberCases, _ := a.underlying.FetchOpenCases(ctx, member.JMBG)
		hasOpenCase := len(memberCases) > 0

		// Determine risk level from cases
		riskLevel := ""
		for _, c := range memberCases {
			if c.RiskLevel != "" {
				if riskLevel == "" || isHigherRisk(c.RiskLevel, riskLevel) {
					riskLevel = c.RiskLevel
				}
			}
		}

		result = append(result, PseudonymizedFamilyMember{
			PseudonymID:  memberPseudonym,
			Relationship: member.Relationship,
			AgeRange:     calculateAgeRange(member.DateOfBirth),
			IsMinor:      isMinor(member.DateOfBirth),
			HasOpenCase:  hasOpenCase,
			RiskLevel:    riskLevel,
		})
	}

	return result, nil
}

// PseudonymizeCaseUpdateEvent converts a case update event to pseudonymized form.
func (a *PrivacyAwareSocialAdapter) PseudonymizeCaseUpdateEvent(
	ctx context.Context,
	event social.CaseUpdateEvent,
) (*PseudonymizedCaseUpdateEvent, error) {
	pseudonymID, err := a.pseudoSvc.Pseudonymize(ctx, event.ClientJMBG)
	if err != nil {
		return nil, fmt.Errorf("failed to pseudonymize: %w", err)
	}

	return &PseudonymizedCaseUpdateEvent{
		EventID:            event.EventID,
		Timestamp:          event.Timestamp,
		CaseID:             event.CaseID,
		PseudonymID:        pseudonymID,
		UpdateType:         event.UpdateType,
		CSRCode:            event.CSRCode,
		SourceFacilityCode: a.facilityCode,
	}, nil
}

// GetPseudonymID returns the pseudonym for a JMBG without fetching any data.
func (a *PrivacyAwareSocialAdapter) GetPseudonymID(ctx context.Context, jmbg string) (PseudonymID, error) {
	return a.pseudoSvc.Pseudonymize(ctx, jmbg)
}

// Underlying returns the underlying adapter for local facility use only.
// This should NEVER be exposed to the central system.
func (a *PrivacyAwareSocialAdapter) Underlying() social.Adapter {
	return a.underlying
}

// Pseudonymized event types for the central system

// PseudonymizedAdmissionEvent is the privacy-safe version of AdmissionEvent.
type PseudonymizedAdmissionEvent struct {
	EventID            string      `json:"event_id"`
	Timestamp          time.Time   `json:"timestamp"`
	PseudonymID        PseudonymID `json:"pseudonym_id"`
	Department         string      `json:"department"`
	AdmissionType      string      `json:"admission_type"`
	DiagnosisCategory  string      `json:"diagnosis_category,omitempty"` // Category only, not specific ICD
	SourceFacilityCode string      `json:"source_facility_code"`
	// PatientJMBG, PatientName are intentionally excluded
}

// PseudonymizedDischargeEvent is the privacy-safe version of DischargeEvent.
type PseudonymizedDischargeEvent struct {
	EventID            string      `json:"event_id"`
	Timestamp          time.Time   `json:"timestamp"`
	PseudonymID        PseudonymID `json:"pseudonym_id"`
	Department         string      `json:"department"`
	DischargeType      string      `json:"discharge_type"`
	LengthOfStayDays   int         `json:"length_of_stay_days"`
	DiagnosisCategory  string      `json:"diagnosis_category,omitempty"`
	FollowUpNeeded     bool        `json:"follow_up_needed"`
	SourceFacilityCode string      `json:"source_facility_code"`
	// PatientJMBG, PatientName are intentionally excluded
}

// PseudonymizedCaseUpdateEvent is the privacy-safe version of CaseUpdateEvent.
type PseudonymizedCaseUpdateEvent struct {
	EventID            string      `json:"event_id"`
	Timestamp          time.Time   `json:"timestamp"`
	CaseID             string      `json:"case_id"`
	PseudonymID        PseudonymID `json:"pseudonym_id"`
	UpdateType         string      `json:"update_type"`
	CSRCode            string      `json:"csr_code"`
	SourceFacilityCode string      `json:"source_facility_code"`
	// ClientJMBG, WorkerID, Description are intentionally excluded
}

// Helper functions

// extractDiagnosisCategory extracts the category (chapter) from an ICD-10 code.
// Returns only the category letter/number, not the specific diagnosis.
func extractDiagnosisCategory(icd10 string) string {
	if icd10 == "" {
		return ""
	}

	// ICD-10 categories by first letter
	categories := map[byte]string{
		'A': "infectious_diseases",
		'B': "infectious_diseases",
		'C': "neoplasms",
		'D': "blood_disorders",
		'E': "endocrine_metabolic",
		'F': "mental_health",
		'G': "nervous_system",
		'H': "eye_ear",
		'I': "circulatory",
		'J': "respiratory",
		'K': "digestive",
		'L': "skin",
		'M': "musculoskeletal",
		'N': "genitourinary",
		'O': "pregnancy",
		'P': "perinatal",
		'Q': "congenital",
		'R': "symptoms_signs",
		'S': "injury",
		'T': "injury",
		'V': "external_causes",
		'W': "external_causes",
		'X': "external_causes",
		'Y': "external_causes",
		'Z': "health_services",
	}

	if len(icd10) > 0 {
		if category, ok := categories[icd10[0]]; ok {
			return category
		}
	}

	return "other"
}

// isHigherRisk compares two risk levels and returns true if newRisk is higher.
func isHigherRisk(newRisk, currentRisk string) bool {
	riskOrder := map[string]int{
		RiskLevelLow:      1,
		RiskLevelMedium:   2,
		RiskLevelHigh:     3,
		RiskLevelCritical: 4,
	}

	return riskOrder[newRisk] > riskOrder[currentRisk]
}
