package gateway

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/federation/trust"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// --- Mock Repository for Trust Authority ---

type mockRepository struct {
	agencies map[types.ID]*trust.TrustedAgency
	services map[types.ID][]trust.ServiceEndpoint
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		agencies: make(map[types.ID]*trust.TrustedAgency),
		services: make(map[types.ID][]trust.ServiceEndpoint),
	}
}

func (r *mockRepository) SaveAgency(ctx context.Context, agency *trust.TrustedAgency) error {
	r.agencies[agency.ID] = agency
	return nil
}

func (r *mockRepository) GetAgency(ctx context.Context, id types.ID) (*trust.TrustedAgency, error) {
	return r.agencies[id], nil
}

func (r *mockRepository) GetAgencyByCode(ctx context.Context, code string) (*trust.TrustedAgency, error) {
	for _, a := range r.agencies {
		if a.Code == code {
			return a, nil
		}
	}
	return nil, nil
}

func (r *mockRepository) ListAgencies(ctx context.Context) ([]trust.TrustedAgency, error) {
	var result []trust.TrustedAgency
	for _, a := range r.agencies {
		result = append(result, *a)
	}
	return result, nil
}

func (r *mockRepository) UpdateAgency(ctx context.Context, agency *trust.TrustedAgency) error {
	r.agencies[agency.ID] = agency
	return nil
}

func (r *mockRepository) DeleteAgency(ctx context.Context, id types.ID) error {
	delete(r.agencies, id)
	return nil
}

func (r *mockRepository) SaveService(ctx context.Context, service *trust.ServiceEndpoint) error {
	r.services[service.AgencyID] = append(r.services[service.AgencyID], *service)
	return nil
}

func (r *mockRepository) GetServices(ctx context.Context, agencyID types.ID) ([]trust.ServiceEndpoint, error) {
	return r.services[agencyID], nil
}

func (r *mockRepository) GetServicesByType(ctx context.Context, serviceType string) ([]trust.ServiceEndpoint, error) {
	var result []trust.ServiceEndpoint
	for _, services := range r.services {
		for _, s := range services {
			if s.ServiceType == serviceType && s.Active {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// --- Gateway Tests ---

func TestNewGateway(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, err := NewGateway(cfg, authority)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if gateway == nil {
		t.Fatal("Gateway should not be nil")
	}
}

func TestNewGatewayWithoutPrivateKey(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: nil, // No private key
	}

	_, err := NewGateway(cfg, authority)

	if err == nil {
		t.Error("Expected error for missing private key")
	}
}

func TestSignAndVerifyRequest(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)
	ctx := context.Background()

	// Register source agency with authority
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Create gateway with matching keys
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   agency.ID,
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Create a request
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC(),
		SourceAgency: "MUP",
		TargetAgency: "PURS",
		Method:       "POST",
		Path:         "/api/v1/verify",
		Body:         []byte(`{"jmbg":"1234567890123"}`),
	}

	// Sign the request
	err := gateway.signRequest(request)
	if err != nil {
		t.Fatalf("Expected no error signing request, got: %v", err)
	}

	if request.Signature == "" {
		t.Error("Signature should not be empty")
	}

	// Update agency with the gateway's public key for verification
	agency.PublicKey = gateway.publicKey
	repo.SaveAgency(ctx, agency)

	// Verify the request
	err = gateway.VerifyRequest(ctx, request)
	if err != nil {
		t.Errorf("Expected no error verifying request, got: %v", err)
	}
}

func TestVerifyRequestWithInvalidSignature(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)
	ctx := context.Background()

	// Register source agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   agency.ID,
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Update agency public key
	agency.PublicKey = gateway.publicKey
	repo.SaveAgency(ctx, agency)

	// Create a request with invalid signature
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC(),
		SourceAgency: "MUP",
		TargetAgency: "PURS",
		Method:       "POST",
		Path:         "/api/v1/verify",
		Signature:    "aW52YWxpZHNpZ25hdHVyZQ==", // Invalid base64 signature
	}

	err := gateway.VerifyRequest(ctx, request)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}

func TestVerifyRequestWithOldTimestamp(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)
	ctx := context.Background()

	// Register source agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   agency.ID,
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Update agency public key
	agency.PublicKey = gateway.publicKey
	repo.SaveAgency(ctx, agency)

	// Create a request with old timestamp (more than 5 minutes ago)
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC().Add(-10 * time.Minute),
		SourceAgency: "MUP",
		TargetAgency: "PURS",
		Method:       "POST",
		Path:         "/api/v1/verify",
	}

	// Sign the request
	gateway.signRequest(request)

	// Verification should fail due to old timestamp
	err := gateway.VerifyRequest(ctx, request)
	if err == nil {
		t.Error("Expected error for old timestamp")
	}
}

func TestVerifyRequestFromSuspendedAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)
	ctx := context.Background()

	// Register source agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   agency.ID,
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Update agency public key
	agency.PublicKey = gateway.publicKey
	repo.SaveAgency(ctx, agency)

	// Suspend the agency
	authority.SuspendAgency(ctx, agency.ID, "Test suspension")

	// Create and sign request
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC(),
		SourceAgency: "MUP",
		TargetAgency: "PURS",
		Method:       "POST",
		Path:         "/api/v1/verify",
	}
	gateway.signRequest(request)

	// Verification should fail for suspended agency
	err := gateway.VerifyRequest(ctx, request)
	if err == nil {
		t.Error("Expected error for suspended agency")
	}
}

func TestCreateResponse(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	requestID := types.NewID().String()
	responseBody := []byte(`{"status":"verified"}`)

	resp, err := gateway.CreateResponse(requestID, 200, responseBody)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.RequestID != requestID {
		t.Error("Request ID should match")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	if resp.SourceAgency != "MUP" {
		t.Errorf("Expected source agency 'MUP', got '%s'", resp.SourceAgency)
	}

	if resp.Signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestVerifyResponse(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Create a response
	requestID := types.NewID().String()
	responseBody := []byte(`{"status":"verified"}`)
	resp, _ := gateway.CreateResponse(requestID, 200, responseBody)

	// Verify the response
	err := gateway.verifyResponse(resp, gateway.publicKey)
	if err != nil {
		t.Errorf("Expected no error verifying response, got: %v", err)
	}
}

func TestVerifyResponseWithTamperedBody(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Create a response
	requestID := types.NewID().String()
	responseBody := []byte(`{"status":"verified"}`)
	resp, _ := gateway.CreateResponse(requestID, 200, responseBody)

	// Tamper with the body
	resp.Body = []byte(`{"status":"tampered"}`)

	// Verification should fail
	err := gateway.verifyResponse(resp, gateway.publicKey)
	if err == nil {
		t.Error("Expected error for tampered body")
	}
}

func TestSignRequestWithoutBody(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "MUP",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	// Create a request without body (e.g., GET request)
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC(),
		SourceAgency: "MUP",
		TargetAgency: "PURS",
		Method:       "GET",
		Path:         "/api/v1/status",
		Body:         nil,
	}

	err := gateway.signRequest(request)
	if err != nil {
		t.Fatalf("Expected no error signing request without body, got: %v", err)
	}

	if request.Signature == "" {
		t.Error("Signature should not be empty even for bodyless request")
	}
}

func TestSignedRequestFields(t *testing.T) {
	repo := newMockRepository()
	authority, _ := trust.NewAuthority(repo)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	cfg := Config{
		AgencyID:   types.NewID(),
		AgencyCode: "SOURCE",
		PrivateKey: privateKey,
	}

	gateway, _ := NewGateway(cfg, authority)

	request := &SignedRequest{
		ID:            "req-123",
		Timestamp:     time.Now().UTC(),
		SourceAgency:  "SOURCE",
		TargetAgency:  "TARGET",
		Method:        "POST",
		Path:          "/api/test",
		Headers:       map[string]string{"Content-Type": "application/json"},
		Body:          []byte(`{"test":true}`),
		CorrelationID: "corr-456",
	}

	gateway.signRequest(request)

	// Verify all fields are preserved
	if request.ID != "req-123" {
		t.Error("ID should be preserved")
	}

	if request.SourceAgency != "SOURCE" {
		t.Error("Source agency should be preserved")
	}

	if request.TargetAgency != "TARGET" {
		t.Error("Target agency should be preserved")
	}

	if request.Method != "POST" {
		t.Error("Method should be preserved")
	}

	if request.CorrelationID != "corr-456" {
		t.Error("Correlation ID should be preserved")
	}
}
