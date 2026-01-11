package coordination

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/adapters/health"
	"github.com/serbia-gov/platform/internal/adapters/social"
)

// EnrichmentService enriches coordination events with cross-system context
type EnrichmentService struct {
	healthAdapter health.Adapter
	socialAdapter social.Adapter

	// Cache for recent enrichments to avoid repeated lookups
	cache     map[string]*cachedEnrichment
	cacheMu   sync.RWMutex
	cacheTTL  time.Duration

	// Configuration
	config EnrichmentConfig
}

type cachedEnrichment struct {
	data      *EventEnrichment
	expiresAt time.Time
}

// EnrichmentConfig holds configuration for the enrichment service
type EnrichmentConfig struct {
	// Enable/disable specific enrichment sources
	EnableHealthEnrichment bool
	EnableSocialEnrichment bool
	EnableFamilyLookup     bool
	EnableCaseLookup       bool

	// Timeouts
	HealthTimeout time.Duration
	SocialTimeout time.Duration

	// Cache settings
	CacheTTL time.Duration

	// Lookback period for historical data
	HealthLookbackDays int
	CaseLookbackDays   int
}

// DefaultEnrichmentConfig returns sensible defaults
func DefaultEnrichmentConfig() EnrichmentConfig {
	return EnrichmentConfig{
		EnableHealthEnrichment: true,
		EnableSocialEnrichment: true,
		EnableFamilyLookup:     true,
		EnableCaseLookup:       true,
		HealthTimeout:          5 * time.Second,
		SocialTimeout:          5 * time.Second,
		CacheTTL:               5 * time.Minute,
		HealthLookbackDays:     365,
		CaseLookbackDays:       730, // 2 years
	}
}

// NewEnrichmentService creates a new enrichment service
func NewEnrichmentService(
	healthAdapter health.Adapter,
	socialAdapter social.Adapter,
	config EnrichmentConfig,
) *EnrichmentService {
	return &EnrichmentService{
		healthAdapter: healthAdapter,
		socialAdapter: socialAdapter,
		cache:         make(map[string]*cachedEnrichment),
		cacheTTL:      config.CacheTTL,
		config:        config,
	}
}

// Enrich adds cross-system context to a coordination event
func (s *EnrichmentService) Enrich(ctx context.Context, event *CoordinationEvent) error {
	if event.SubjectJMBG == "" {
		return fmt.Errorf("event has no subject JMBG")
	}

	// Check cache first
	if cached := s.getCached(event.SubjectJMBG); cached != nil {
		event.Enrichment = cached
		return nil
	}

	enrichment := &EventEnrichment{
		EnrichedAt: time.Now(),
		Sources:    make([]string, 0),
	}

	// Collect enrichment data concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// Health enrichment
	if s.config.EnableHealthEnrichment && s.healthAdapter != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			healthCtx, err := s.enrichHealth(ctx, event.SubjectJMBG)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("health enrichment: %w", err))
			} else if healthCtx != nil {
				enrichment.HealthContext = healthCtx
				enrichment.Sources = append(enrichment.Sources, "health")
			}
		}()
	}

	// Social enrichment
	if s.config.EnableSocialEnrichment && s.socialAdapter != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			socialCtx, err := s.enrichSocial(ctx, event.SubjectJMBG)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("social enrichment: %w", err))
			} else if socialCtx != nil {
				enrichment.SocialContext = socialCtx
				enrichment.Sources = append(enrichment.Sources, "social")
			}
		}()
	}

	// Family lookup
	if s.config.EnableFamilyLookup && s.socialAdapter != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			family, err := s.enrichFamily(ctx, event.SubjectJMBG)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("family enrichment: %w", err))
			} else if family != nil {
				enrichment.FamilyMembers = family
				enrichment.Sources = append(enrichment.Sources, "family")
			}
		}()
	}

	// Related cases lookup
	if s.config.EnableCaseLookup && s.socialAdapter != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cases, err := s.enrichCases(ctx, event.SubjectJMBG)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("case enrichment: %w", err))
			} else if cases != nil {
				enrichment.RelatedCases = cases
				enrichment.Sources = append(enrichment.Sources, "cases")
			}
		}()
	}

	wg.Wait()

	// Calculate composite risk assessment
	s.calculateRisk(enrichment, event)

	// Generate recommendations
	s.generateRecommendations(enrichment, event)

	// Store enrichment
	event.Enrichment = enrichment

	// Update cache
	s.setCache(event.SubjectJMBG, enrichment)

	return nil
}

// enrichHealth fetches health context
func (s *EnrichmentService) enrichHealth(ctx context.Context, jmbg string) (*health.HealthContext, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.HealthTimeout)
	defer cancel()

	healthCtx := health.NewHealthContext(jmbg)

	// Fetch patient record
	patient, err := s.healthAdapter.FetchPatientRecord(ctx, jmbg)
	if err == nil && patient != nil {
		healthCtx.PatientRecord = patient
	}

	// Fetch recent hospitalizations
	to := time.Now()
	from := to.AddDate(0, 0, -s.config.HealthLookbackDays)
	hospitalizations, err := s.healthAdapter.FetchHospitalizations(ctx, jmbg, from, to)
	if err == nil {
		healthCtx.Hospitalizations = hospitalizations
	}

	// Fetch active prescriptions
	prescriptions, err := s.healthAdapter.FetchPrescriptions(ctx, jmbg, true)
	if err == nil {
		healthCtx.Prescriptions = prescriptions
	}

	healthCtx.UpdateFlags()

	return healthCtx, nil
}

// enrichSocial fetches social context
func (s *EnrichmentService) enrichSocial(ctx context.Context, jmbg string) (*social.SocialContext, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.SocialTimeout)
	defer cancel()

	socialCtx := social.NewSocialContext(jmbg)

	// Fetch beneficiary status
	status, err := s.socialAdapter.FetchBeneficiaryStatus(ctx, jmbg)
	if err == nil && status != nil {
		socialCtx.BeneficiaryStatus = status
	}

	// Fetch family composition
	family, err := s.socialAdapter.FetchFamilyComposition(ctx, jmbg)
	if err == nil && family != nil {
		socialCtx.FamilyUnit = family
	}

	// Fetch open cases
	cases, err := s.socialAdapter.FetchOpenCases(ctx, jmbg)
	if err == nil {
		socialCtx.OpenCases = cases
	}

	// Fetch risk assessment
	risk, err := s.socialAdapter.FetchRiskAssessment(ctx, jmbg)
	if err == nil && risk != nil {
		socialCtx.RiskAssessment = risk
	}

	socialCtx.UpdateFlags()

	return socialCtx, nil
}

// enrichFamily fetches family member information
func (s *EnrichmentService) enrichFamily(ctx context.Context, jmbg string) ([]FamilyMemberInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.SocialTimeout)
	defer cancel()

	family, err := s.socialAdapter.FetchFamilyComposition(ctx, jmbg)
	if err != nil || family == nil {
		return nil, err
	}

	members := make([]FamilyMemberInfo, 0, len(family.Members))
	for _, m := range family.Members {
		age := calculateAge(m.DateOfBirth)
		info := FamilyMemberInfo{
			JMBG:         m.JMBG,
			Name:         fmt.Sprintf("%s %s", m.FirstName, m.LastName),
			Relationship: m.Relationship,
			Age:          age,
			IsMinor:      age < 18,
		}

		// Check if family member has open cases
		memberCases, err := s.socialAdapter.FetchOpenCases(ctx, m.JMBG)
		if err == nil && len(memberCases) > 0 {
			info.HasOpenCase = true
			// Get highest risk level from cases
			for _, c := range memberCases {
				if isHigherRisk(c.RiskLevel, info.RiskLevel) {
					info.RiskLevel = c.RiskLevel
				}
			}
		}

		members = append(members, info)
	}

	return members, nil
}

// enrichCases fetches related cases
func (s *EnrichmentService) enrichCases(ctx context.Context, jmbg string) ([]RelatedCase, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.SocialTimeout)
	defer cancel()

	openCases, err := s.socialAdapter.FetchOpenCases(ctx, jmbg)
	if err != nil {
		return nil, err
	}

	related := make([]RelatedCase, 0, len(openCases))
	for _, c := range openCases {
		related = append(related, RelatedCase{
			CaseID:     c.ID,
			CaseType:   c.CaseType,
			Agency:     c.CSRName,
			Status:     c.Status,
			Priority:   c.Priority,
			OpenedAt:   c.OpenedAt,
			AssignedTo: c.AssignedWorker,
		})
	}

	return related, nil
}

// calculateRisk computes a composite risk score
func (s *EnrichmentService) calculateRisk(enrichment *EventEnrichment, event *CoordinationEvent) {
	riskScore := 0
	var riskFactors []string
	var vulnerableFlags []string

	// Health-based risk factors
	if enrichment.HealthContext != nil {
		hc := enrichment.HealthContext

		if hc.HasChronicCondition {
			riskScore += 10
			riskFactors = append(riskFactors, "chronic_condition")
		}

		if hc.HasRecentHospitalization {
			riskScore += 15
			riskFactors = append(riskFactors, "recent_hospitalization")
		}

		if hc.HasActiveTreatment {
			riskScore += 5
			riskFactors = append(riskFactors, "active_treatment")
		}

		if hc.RequiresContinuousCare {
			riskScore += 20
			riskFactors = append(riskFactors, "continuous_care_needed")
			vulnerableFlags = append(vulnerableFlags, "health_dependent")
		}
	}

	// Social-based risk factors
	if enrichment.SocialContext != nil {
		sc := enrichment.SocialContext

		if sc.IsBeneficiary {
			riskScore += 10
			riskFactors = append(riskFactors, "social_beneficiary")
		}

		if sc.HasOpenCases {
			riskScore += 15
			riskFactors = append(riskFactors, "open_social_cases")
		}

		if sc.RiskAssessment != nil {
			switch sc.RiskAssessment.OverallRisk {
			case "critical":
				riskScore += 40
			case "high":
				riskScore += 30
			case "medium":
				riskScore += 15
			}
			// Extract factor descriptions from RiskFactor structs
			for _, rf := range sc.RiskAssessment.RiskFactors {
				riskFactors = append(riskFactors, rf.Factor)
			}
		}

		if sc.RequiresImmediateAction {
			riskScore += 25
			vulnerableFlags = append(vulnerableFlags, "immediate_action_required")
		}
	}

	// Family-based risk factors
	for _, member := range enrichment.FamilyMembers {
		if member.IsMinor && member.HasOpenCase {
			riskScore += 20
			riskFactors = append(riskFactors, fmt.Sprintf("minor_with_case_%s", member.JMBG))
			vulnerableFlags = append(vulnerableFlags, "minor_at_risk")
		}
	}

	// Event-type specific risk
	switch event.Type {
	case EventTypeEmergency:
		riskScore += 30
	case EventTypeChildProtection:
		riskScore += 40
		vulnerableFlags = append(vulnerableFlags, "child_protection_concern")
	case EventTypeDomesticViolence:
		riskScore += 35
		vulnerableFlags = append(vulnerableFlags, "domestic_violence_concern")
	case EventTypeVulnerablePerson:
		riskScore += 25
		vulnerableFlags = append(vulnerableFlags, "vulnerable_person")
	}

	// Cap at 100
	if riskScore > 100 {
		riskScore = 100
	}

	// Determine risk level
	var riskLevel string
	switch {
	case riskScore >= 70:
		riskLevel = "critical"
	case riskScore >= 50:
		riskLevel = "high"
	case riskScore >= 30:
		riskLevel = "medium"
	default:
		riskLevel = "low"
	}

	enrichment.RiskScore = riskScore
	enrichment.RiskLevel = riskLevel
	enrichment.RiskFactors = riskFactors
	enrichment.VulnerableFlags = vulnerableFlags
}

// generateRecommendations generates action recommendations
func (s *EnrichmentService) generateRecommendations(enrichment *EventEnrichment, event *CoordinationEvent) {
	var recommendations []string

	// Based on risk level
	switch enrichment.RiskLevel {
	case "critical":
		recommendations = append(recommendations, "Immediate response required - escalate to supervisor")
		recommendations = append(recommendations, "Consider multi-agency coordination meeting")
	case "high":
		recommendations = append(recommendations, "Priority handling - respond within 2 hours")
		recommendations = append(recommendations, "Review all related cases")
	}

	// Based on event type
	switch event.Type {
	case EventTypeAdmission:
		if enrichment.SocialContext != nil && enrichment.SocialContext.HasOpenCases {
			recommendations = append(recommendations, "Notify assigned social worker of hospitalization")
		}
	case EventTypeDischarge:
		if enrichment.HealthContext != nil && enrichment.HealthContext.RequiresContinuousCare {
			recommendations = append(recommendations, "Ensure follow-up care is arranged before discharge")
			recommendations = append(recommendations, "Coordinate with home care services if needed")
		}
	case EventTypeChildProtection:
		recommendations = append(recommendations, "Apply child protection protocol")
		recommendations = append(recommendations, "Document all interactions")
		recommendations = append(recommendations, "Consider family assessment")
	case EventTypeDomesticViolence:
		recommendations = append(recommendations, "Apply domestic violence protocol")
		recommendations = append(recommendations, "Ensure victim safety first")
		recommendations = append(recommendations, "Consider emergency shelter if needed")
	}

	// Based on family situation
	for _, member := range enrichment.FamilyMembers {
		if member.IsMinor && member.HasOpenCase {
			recommendations = append(recommendations,
				fmt.Sprintf("Minor %s has open case - coordinate with their worker", member.JMBG))
		}
	}

	// Based on social context
	if enrichment.SocialContext != nil {
		if enrichment.SocialContext.IsBeneficiary {
			recommendations = append(recommendations, "Check if benefits are affected by this event")
		}
	}

	enrichment.RecommendedActions = recommendations
}

// getCached returns cached enrichment if still valid
func (s *EnrichmentService) getCached(jmbg string) *EventEnrichment {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	cached, ok := s.cache[jmbg]
	if !ok || time.Now().After(cached.expiresAt) {
		return nil
	}
	return cached.data
}

// setCache stores enrichment in cache
func (s *EnrichmentService) setCache(jmbg string, enrichment *EventEnrichment) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache[jmbg] = &cachedEnrichment{
		data:      enrichment,
		expiresAt: time.Now().Add(s.cacheTTL),
	}
}

// ClearCache clears the enrichment cache
func (s *EnrichmentService) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = make(map[string]*cachedEnrichment)
}

// Helper functions

func calculateAge(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

func isHigherRisk(newRisk, currentRisk string) bool {
	riskOrder := map[string]int{
		"":         0,
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}
	return riskOrder[newRisk] > riskOrder[currentRisk]
}
