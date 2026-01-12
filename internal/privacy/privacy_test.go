package privacy

import (
	"context"
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// --- PseudonymID Tests ---

func TestPseudonymIDIsZero(t *testing.T) {
	var empty PseudonymID
	if !empty.IsZero() {
		t.Error("Empty PseudonymID should be zero")
	}

	nonEmpty := PseudonymID("PSE-abc123")
	if nonEmpty.IsZero() {
		t.Error("Non-empty PseudonymID should not be zero")
	}
}

func TestPseudonymIDString(t *testing.T) {
	p := PseudonymID("PSE-abc123")
	if p.String() != "PSE-abc123" {
		t.Errorf("Expected 'PSE-abc123', got '%s'", p.String())
	}
}

// --- LegalBasis Tests ---

func TestLegalBasisRequiresManualApproval(t *testing.T) {
	tests := []struct {
		basis           LegalBasis
		requiresApproval bool
	}{
		{LegalBasisCourtOrder, false},      // Auto-approved
		{LegalBasisLifeThreat, false},      // Auto-approved
		{LegalBasisChildProtection, true},  // Requires approval
		{LegalBasisLawEnforcement, true},   // Requires approval
		{LegalBasisSubjectConsent, true},   // Requires approval
	}

	for _, tt := range tests {
		t.Run(string(tt.basis), func(t *testing.T) {
			if tt.basis.RequiresManualApproval() != tt.requiresApproval {
				t.Errorf("Expected RequiresManualApproval=%v for %s", tt.requiresApproval, tt.basis)
			}
		})
	}
}

// --- DataAccessLevel Tests ---

func TestDataAccessLevelString(t *testing.T) {
	tests := []struct {
		level    DataAccessLevel
		expected string
	}{
		{DataAccessLevelAggregated, "aggregated"},
		{DataAccessLevelPseudonymized, "pseudonymized"},
		{DataAccessLevelLinkable, "linkable"},
		{DataAccessLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.level.String())
			}
		})
	}
}

// --- DepseudonymizationToken Tests ---

func TestTokenIsValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		token    DepseudonymizationToken
		expected bool
	}{
		{
			name: "Valid token",
			token: DepseudonymizationToken{
				ExpiresAt: now.Add(1 * time.Hour),
				UsedCount: 0,
				MaxUses:   3,
			},
			expected: true,
		},
		{
			name: "Expired token",
			token: DepseudonymizationToken{
				ExpiresAt: now.Add(-1 * time.Hour),
				UsedCount: 0,
				MaxUses:   3,
			},
			expected: false,
		},
		{
			name: "Used up token",
			token: DepseudonymizationToken{
				ExpiresAt: now.Add(1 * time.Hour),
				UsedCount: 3,
				MaxUses:   3,
			},
			expected: false,
		},
		{
			name: "Revoked token",
			token: DepseudonymizationToken{
				ExpiresAt: now.Add(1 * time.Hour),
				UsedCount: 0,
				MaxUses:   3,
				RevokedAt: &now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token.IsValid() != tt.expected {
				t.Errorf("Expected IsValid=%v for %s", tt.expected, tt.name)
			}
		})
	}
}

// --- DepseudonymizationRequest Tests ---

func TestRequestIsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		req      DepseudonymizationRequest
		expected bool
	}{
		{
			name: "Active approved request",
			req: DepseudonymizationRequest{
				Status:    RequestStatusApproved,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "Expired approved request",
			req: DepseudonymizationRequest{
				Status:    RequestStatusApproved,
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Pending request",
			req: DepseudonymizationRequest{
				Status:    RequestStatusPending,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Rejected request",
			req: DepseudonymizationRequest{
				Status:    RequestStatusRejected,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.IsActive() != tt.expected {
				t.Errorf("Expected IsActive=%v for %s", tt.expected, tt.name)
			}
		})
	}
}

// --- AIAccessRequest Tests ---

func TestAIAccessRequestIsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		req      AIAccessRequest
		expected bool
	}{
		{
			name: "Active approved request",
			req: AIAccessRequest{
				Status:    RequestStatusApproved,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "Expired approved request",
			req: AIAccessRequest{
				Status:    RequestStatusApproved,
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Pending request",
			req: AIAccessRequest{
				Status:    RequestStatusPending,
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.IsActive() != tt.expected {
				t.Errorf("Expected IsActive=%v for %s", tt.expected, tt.name)
			}
		})
	}
}

// --- Masking Function Tests ---

func TestMaskJMBG(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890123", "1234567******"},
		{"0101990123456", "0101990******"},
		{"short", "***********"},
		{"", "***********"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MaskJMBG(tt.input)
			if result != tt.expected {
				t.Errorf("MaskJMBG(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+381641234567", "***-***-4567"},
		{"0641234567", "***-***-4567"},
		{"123", "****"},
		{"", "****"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MaskPhone(tt.input)
			if result != tt.expected {
				t.Errorf("MaskPhone(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john.doe@example.com", "jo***@example.com"},
		{"ab@test.com", "a***@test.com"},
		{"x@y.com", "x***@y.com"},
		{"invalid", "***@***"},
		{"@test.com", "***@***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MaskEmail(tt.input)
			if result != tt.expected {
				t.Errorf("MaskEmail(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Marko", "M. ***"},
		{"Ana", "A. ***"},
		{"X", "*"},
		{"", "*"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := MaskName(tt.input)
			if result != tt.expected {
				t.Errorf("MaskName(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Mock Repository ---

type mockPseudonymRepo struct {
	mappings map[string]*PseudonymMapping
}

func newMockPseudonymRepo() *mockPseudonymRepo {
	return &mockPseudonymRepo{
		mappings: make(map[string]*PseudonymMapping),
	}
}

func (r *mockPseudonymRepo) Store(ctx context.Context, mapping *PseudonymMapping) error {
	r.mappings[mapping.JMBGHash] = mapping
	return nil
}

func (r *mockPseudonymRepo) GetByJMBGHash(ctx context.Context, jmbgHash, facilityCode string) (*PseudonymMapping, error) {
	if m, ok := r.mappings[jmbgHash]; ok && m.FacilityCode == facilityCode {
		return m, nil
	}
	return nil, nil
}

func (r *mockPseudonymRepo) GetByPseudonymID(ctx context.Context, pseudonymID PseudonymID) (*PseudonymMapping, error) {
	for _, m := range r.mappings {
		if m.PseudonymID == pseudonymID {
			return m, nil
		}
	}
	return nil, nil
}

func (r *mockPseudonymRepo) Delete(ctx context.Context, pseudonymID PseudonymID) error {
	for k, m := range r.mappings {
		if m.PseudonymID == pseudonymID {
			delete(r.mappings, k)
			return nil
		}
	}
	return nil
}

// --- PseudonymizationService Tests ---

func TestPseudonymizationService_Pseudonymize(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()
	jmbg := "0101990123456"

	// First call - should create new pseudonym
	pseudonym1, err := svc.Pseudonymize(ctx, jmbg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pseudonym1.IsZero() {
		t.Error("Pseudonym should not be empty")
	}

	if len(pseudonym1) < 10 {
		t.Error("Pseudonym should have sufficient length")
	}

	// Second call - should return same pseudonym (deterministic)
	pseudonym2, err := svc.Pseudonymize(ctx, jmbg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if pseudonym1 != pseudonym2 {
		t.Errorf("Pseudonymization should be deterministic: %s != %s", pseudonym1, pseudonym2)
	}
}

func TestPseudonymizationService_DifferentJMBGs(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()

	// Different JMBGs should produce different pseudonyms
	p1, _ := svc.Pseudonymize(ctx, "0101990123456")
	p2, _ := svc.Pseudonymize(ctx, "0202985654321")

	if p1 == p2 {
		t.Error("Different JMBGs should produce different pseudonyms")
	}
}

func TestPseudonymizationService_DifferentFacilities(t *testing.T) {
	repo1 := newMockPseudonymRepo()
	repo2 := newMockPseudonymRepo()

	key := []byte("test-hmac-key-32-bytes-long!!!!")

	svc1 := NewPseudonymizationService(key, "CSW-BG-01", repo1, nil)
	svc2 := NewPseudonymizationService(key, "CSW-NS-01", repo2, nil)

	ctx := context.Background()
	jmbg := "0101990123456"

	// Same JMBG with different facility codes should produce different pseudonyms
	p1, _ := svc1.Pseudonymize(ctx, jmbg)
	p2, _ := svc2.Pseudonymize(ctx, jmbg)

	if p1 == p2 {
		t.Error("Same JMBG at different facilities should produce different pseudonyms")
	}
}

func TestPseudonymizationService_EmptyJMBG(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()

	_, err := svc.Pseudonymize(ctx, "")
	if err == nil {
		t.Error("Expected error for empty JMBG")
	}
}

func TestPseudonymizationService_PseudonymizeMany(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()
	jmbgs := []string{"0101990123456", "0202985654321", "0303980111222"}

	result, err := svc.PseudonymizeMany(ctx, jmbgs)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}

	// All pseudonyms should be different
	seen := make(map[PseudonymID]bool)
	for _, p := range result {
		if seen[p] {
			t.Error("Duplicate pseudonym found")
		}
		seen[p] = true
	}
}

func TestPseudonymizationService_Exists(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()
	jmbg := "0101990123456"

	// Create pseudonym
	pseudonym, _ := svc.Pseudonymize(ctx, jmbg)

	// Should exist
	exists, err := svc.Exists(ctx, pseudonym)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !exists {
		t.Error("Pseudonym should exist")
	}

	// Non-existent should not exist
	exists, err = svc.Exists(ctx, PseudonymID("PSE-nonexistent"))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if exists {
		t.Error("Non-existent pseudonym should not exist")
	}
}

func TestPseudonymizationService_Delete(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	ctx := context.Background()
	jmbg := "0101990123456"

	// Create pseudonym
	pseudonym, _ := svc.Pseudonymize(ctx, jmbg)

	// Delete
	err := svc.Delete(ctx, pseudonym)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should no longer exist
	exists, _ := svc.Exists(ctx, pseudonym)
	if exists {
		t.Error("Deleted pseudonym should not exist")
	}
}

func TestPseudonymizationService_FacilityCode(t *testing.T) {
	repo := newMockPseudonymRepo()
	svc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	if svc.FacilityCode() != "CSW-BG-01" {
		t.Errorf("Expected 'CSW-BG-01', got '%s'", svc.FacilityCode())
	}
}

// --- Mock Depseudonymization Repository ---

type mockDepseudoRepo struct {
	requests map[types.ID]*DepseudonymizationRequest
	tokens   map[string]*DepseudonymizationToken
}

func newMockDepseudoRepo() *mockDepseudoRepo {
	return &mockDepseudoRepo{
		requests: make(map[types.ID]*DepseudonymizationRequest),
		tokens:   make(map[string]*DepseudonymizationToken),
	}
}

func (r *mockDepseudoRepo) CreateRequest(ctx context.Context, req *DepseudonymizationRequest) error {
	r.requests[req.ID] = req
	return nil
}

func (r *mockDepseudoRepo) GetRequest(ctx context.Context, id types.ID) (*DepseudonymizationRequest, error) {
	return r.requests[id], nil
}

func (r *mockDepseudoRepo) UpdateRequest(ctx context.Context, req *DepseudonymizationRequest) error {
	r.requests[req.ID] = req
	return nil
}

func (r *mockDepseudoRepo) ListPendingRequests(ctx context.Context, approverAgency types.ID) ([]*DepseudonymizationRequest, error) {
	var result []*DepseudonymizationRequest
	for _, req := range r.requests {
		if req.Status == RequestStatusPending {
			result = append(result, req)
		}
	}
	return result, nil
}

func (r *mockDepseudoRepo) StoreToken(ctx context.Context, token *DepseudonymizationToken) error {
	r.tokens[token.Token] = token
	return nil
}

func (r *mockDepseudoRepo) GetToken(ctx context.Context, tokenStr string) (*DepseudonymizationToken, error) {
	return r.tokens[tokenStr], nil
}

func (r *mockDepseudoRepo) IncrementTokenUsage(ctx context.Context, tokenStr string) error {
	if t, ok := r.tokens[tokenStr]; ok {
		t.UsedCount++
	}
	return nil
}

func (r *mockDepseudoRepo) RevokeToken(ctx context.Context, tokenStr string, revokedBy types.ID) error {
	if t, ok := r.tokens[tokenStr]; ok {
		now := time.Now()
		t.RevokedAt = &now
		t.RevokedBy = &revokedBy
	}
	return nil
}

// --- DepseudonymizationService Tests ---

func TestDepseudonymizationService_RequestValidation(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()
	requestorID := types.NewID()
	agencyID := types.NewID()
	caseID := types.NewID()

	// Create a pseudonym first
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")

	// Valid request
	req, err := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		requestorID,
		agencyID,
		"child welfare investigation",
		LegalBasisChildProtection,
		"Need to verify identity for child welfare case due to urgent safety concerns",
		caseID,
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if req.ID.IsZero() {
		t.Error("Request ID should be set")
	}

	if req.Status != RequestStatusPending {
		t.Errorf("Expected pending status, got %s", req.Status)
	}
}

func TestDepseudonymizationService_AutoApproval(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()
	requestorID := types.NewID()
	agencyID := types.NewID()
	caseID := types.NewID()

	// Create a pseudonym first
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")

	// Life threat should be auto-approved
	req, err := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		requestorID,
		agencyID,
		"emergency medical situation",
		LegalBasisLifeThreat,
		"Immediate life threat - need to identify patient for emergency treatment",
		caseID,
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if req.Status != RequestStatusApproved {
		t.Errorf("Life threat request should be auto-approved, got %s", req.Status)
	}

	if req.ApprovedBy == nil || *req.ApprovedBy != types.ID("system") {
		t.Error("Auto-approved request should have system as approver")
	}
}

func TestDepseudonymizationService_RequestWithInvalidPseudonym(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()

	_, err := svc.RequestDepseudonymization(
		ctx,
		PseudonymID("PSE-nonexistent"),
		types.NewID(),
		types.NewID(),
		"test",
		LegalBasisChildProtection,
		"This is a test justification that is long enough",
		types.NewID(),
	)

	if err == nil {
		t.Error("Expected error for non-existent pseudonym")
	}
}

func TestDepseudonymizationService_ApproveRequest(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()
	requestorID := types.NewID()
	agencyID := types.NewID()
	approverID := types.NewID()
	caseID := types.NewID()

	// Create a pseudonym
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")

	// Create pending request
	req, _ := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		requestorID,
		agencyID,
		"investigation",
		LegalBasisChildProtection,
		"This is a detailed justification for the request",
		caseID,
	)

	// Approve
	token, err := svc.ApproveRequest(ctx, req.ID, approverID)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if token == nil {
		t.Fatal("Token should not be nil")
	}

	if token.Token == "" {
		t.Error("Token string should not be empty")
	}

	if token.IsValid() != true {
		t.Error("Newly created token should be valid")
	}

	// Request should be updated
	updated, _ := svc.GetRequest(ctx, req.ID)
	if updated.Status != RequestStatusApproved {
		t.Errorf("Request should be approved, got %s", updated.Status)
	}
}

func TestDepseudonymizationService_RejectRequest(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()
	requestorID := types.NewID()
	agencyID := types.NewID()
	rejectorID := types.NewID()
	caseID := types.NewID()

	// Create a pseudonym
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")

	// Create pending request
	req, _ := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		requestorID,
		agencyID,
		"investigation",
		LegalBasisChildProtection,
		"This is a detailed justification for the request",
		caseID,
	)

	// Reject
	err := svc.RejectRequest(ctx, req.ID, rejectorID, "Insufficient justification")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Request should be updated
	updated, _ := svc.GetRequest(ctx, req.ID)
	if updated.Status != RequestStatusRejected {
		t.Errorf("Request should be rejected, got %s", updated.Status)
	}

	if updated.RejectionReason != "Insufficient justification" {
		t.Error("Rejection reason should be set")
	}
}

func TestDepseudonymizationService_RevokeToken(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()
	requestorID := types.NewID()
	agencyID := types.NewID()
	approverID := types.NewID()
	revokerID := types.NewID()
	caseID := types.NewID()

	// Create and approve
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")
	req, _ := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		requestorID,
		agencyID,
		"investigation",
		LegalBasisChildProtection,
		"This is a detailed justification for the request",
		caseID,
	)
	token, _ := svc.ApproveRequest(ctx, req.ID, approverID)

	// Token should be valid before revocation
	if !token.IsValid() {
		t.Error("Token should be valid before revocation")
	}

	// Revoke
	err := svc.RevokeToken(ctx, token.Token, revokerID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Get updated token and check
	updatedToken, _ := depseudoRepo.GetToken(ctx, token.Token)
	if updatedToken.IsValid() {
		t.Error("Token should be invalid after revocation")
	}
}

func TestDepseudonymizationService_ShortJustification(t *testing.T) {
	pseudoRepo := newMockPseudonymRepo()
	pseudoSvc := NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		pseudoRepo,
		nil,
	)

	depseudoRepo := newMockDepseudoRepo()
	cfg := DefaultDepseudonymizationConfig()
	svc := NewDepseudonymizationService(pseudoSvc, depseudoRepo, nil, cfg)

	ctx := context.Background()

	// Create a pseudonym first
	pseudonymID, _ := pseudoSvc.Pseudonymize(ctx, "0101990123456")

	_, err := svc.RequestDepseudonymization(
		ctx,
		pseudonymID,
		types.NewID(),
		types.NewID(),
		"test",
		LegalBasisChildProtection,
		"Too short", // Less than 20 characters
		types.NewID(),
	)

	if err == nil {
		t.Error("Expected error for short justification")
	}
}
