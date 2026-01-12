package trust

import (
	"context"
	"testing"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// --- Mock Repository ---

type mockRepository struct {
	agencies map[types.ID]*TrustedAgency
	services map[types.ID][]ServiceEndpoint
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		agencies: make(map[types.ID]*TrustedAgency),
		services: make(map[types.ID][]ServiceEndpoint),
	}
}

func (r *mockRepository) SaveAgency(ctx context.Context, agency *TrustedAgency) error {
	r.agencies[agency.ID] = agency
	return nil
}

func (r *mockRepository) GetAgency(ctx context.Context, id types.ID) (*TrustedAgency, error) {
	return r.agencies[id], nil
}

func (r *mockRepository) GetAgencyByCode(ctx context.Context, code string) (*TrustedAgency, error) {
	for _, a := range r.agencies {
		if a.Code == code {
			return a, nil
		}
	}
	return nil, nil
}

func (r *mockRepository) ListAgencies(ctx context.Context) ([]TrustedAgency, error) {
	var result []TrustedAgency
	for _, a := range r.agencies {
		result = append(result, *a)
	}
	return result, nil
}

func (r *mockRepository) UpdateAgency(ctx context.Context, agency *TrustedAgency) error {
	r.agencies[agency.ID] = agency
	return nil
}

func (r *mockRepository) DeleteAgency(ctx context.Context, id types.ID) error {
	delete(r.agencies, id)
	return nil
}

func (r *mockRepository) SaveService(ctx context.Context, service *ServiceEndpoint) error {
	r.services[service.AgencyID] = append(r.services[service.AgencyID], *service)
	return nil
}

func (r *mockRepository) GetServices(ctx context.Context, agencyID types.ID) ([]ServiceEndpoint, error) {
	return r.services[agencyID], nil
}

func (r *mockRepository) GetServicesByType(ctx context.Context, serviceType string) ([]ServiceEndpoint, error) {
	var result []ServiceEndpoint
	for _, services := range r.services {
		for _, s := range services {
			if s.ServiceType == serviceType && s.Active {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// --- Trust Authority Tests ---

func TestNewAuthority(t *testing.T) {
	repo := newMockRepository()
	authority, err := NewAuthority(repo)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if authority == nil {
		t.Fatal("Authority should not be nil")
	}

	if authority.rootKey == nil {
		t.Error("Root key should be generated")
	}

	if authority.rootCert == nil {
		t.Error("Root certificate should be created")
	}
}

func TestRegisterAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	agency, err := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if agency.ID.IsZero() {
		t.Error("Agency ID should be set")
	}

	if agency.Name != "Ministry of Interior" {
		t.Errorf("Expected name 'Ministry of Interior', got '%s'", agency.Name)
	}

	if agency.Code != "MUP" {
		t.Errorf("Expected code 'MUP', got '%s'", agency.Code)
	}

	if agency.GatewayURL != "https://mup.gov.rs/gateway" {
		t.Errorf("Expected gateway URL 'https://mup.gov.rs/gateway', got '%s'", agency.GatewayURL)
	}

	if agency.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", agency.Status)
	}

	if len(agency.PublicKey) == 0 {
		t.Error("Public key should be generated")
	}

	if len(agency.Certificate) == 0 {
		t.Error("Certificate should be issued")
	}
}

func TestRegisterDuplicateAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register first agency
	_, err := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")
	if err != nil {
		t.Fatalf("Expected no error for first registration, got: %v", err)
	}

	// Try to register with same code
	_, err = authority.RegisterAgency(ctx, "Different Name", "MUP", "https://different.gov.rs/gateway")
	if err == nil {
		t.Error("Expected error for duplicate agency code")
	}
}

func TestGetAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	registered, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Get by ID
	agency, err := authority.GetAgency(ctx, registered.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if agency.Code != "MUP" {
		t.Errorf("Expected code 'MUP', got '%s'", agency.Code)
	}
}

func TestGetAgencyByCode(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Get by code
	agency, err := authority.GetAgencyByCode(ctx, "MUP")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if agency.Name != "Ministry of Interior" {
		t.Errorf("Expected name 'Ministry of Interior', got '%s'", agency.Name)
	}
}

func TestListAgencies(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register multiple agencies
	authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")
	authority.RegisterAgency(ctx, "Tax Administration", "PURS", "https://purs.gov.rs/gateway")
	authority.RegisterAgency(ctx, "Social Welfare", "CSW", "https://csw.gov.rs/gateway")

	// List agencies
	agencies, err := authority.ListAgencies(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(agencies) != 3 {
		t.Errorf("Expected 3 agencies, got %d", len(agencies))
	}
}

func TestSuspendAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Suspend
	err := authority.SuspendAgency(ctx, agency.ID, "Security concern")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify status
	updated, _ := authority.GetAgency(ctx, agency.ID)
	if updated.Status != "suspended" {
		t.Errorf("Expected status 'suspended', got '%s'", updated.Status)
	}
}

func TestRevokeAgency(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Revoke
	err := authority.RevokeAgency(ctx, agency.ID, "Permanent removal")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify status
	updated, _ := authority.GetAgency(ctx, agency.ID)
	if updated.Status != "revoked" {
		t.Errorf("Expected status 'revoked', got '%s'", updated.Status)
	}
}

func TestRegisterService(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Register service
	service, err := authority.RegisterService(ctx, agency.ID, "citizen.verify", "/api/v1/citizen/verify", "1.0")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if service.ID.IsZero() {
		t.Error("Service ID should be set")
	}

	if service.ServiceType != "citizen.verify" {
		t.Errorf("Expected service type 'citizen.verify', got '%s'", service.ServiceType)
	}

	if !service.Active {
		t.Error("Service should be active by default")
	}
}

func TestGetServices(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Register multiple services
	authority.RegisterService(ctx, agency.ID, "citizen.verify", "/api/v1/citizen/verify", "1.0")
	authority.RegisterService(ctx, agency.ID, "document.check", "/api/v1/document/check", "1.0")

	// Get services
	services, err := authority.GetServices(ctx, agency.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

func TestFindService(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register multiple agencies with services
	agency1, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")
	agency2, _ := authority.RegisterAgency(ctx, "Tax Administration", "PURS", "https://purs.gov.rs/gateway")

	authority.RegisterService(ctx, agency1.ID, "citizen.verify", "/api/v1/citizen/verify", "1.0")
	authority.RegisterService(ctx, agency2.ID, "citizen.verify", "/api/v1/citizen/verify", "2.0")
	authority.RegisterService(ctx, agency1.ID, "document.check", "/api/v1/document/check", "1.0")

	// Find services by type
	services, err := authority.FindService(ctx, "citizen.verify")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 citizen.verify services, got %d", len(services))
	}
}

func TestVerifyCertificate(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)
	ctx := context.Background()

	// Register agency
	agency, _ := authority.RegisterAgency(ctx, "Ministry of Interior", "MUP", "https://mup.gov.rs/gateway")

	// Verify certificate
	err := authority.VerifyCertificate(agency.Certificate)
	if err != nil {
		t.Errorf("Expected certificate to be valid, got: %v", err)
	}
}

func TestVerifyInvalidCertificate(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)

	// Try to verify invalid certificate
	err := authority.VerifyCertificate([]byte("invalid certificate"))
	if err == nil {
		t.Error("Expected error for invalid certificate")
	}
}

func TestGetRootCertificatePEM(t *testing.T) {
	repo := newMockRepository()
	authority, _ := NewAuthority(repo)

	rootPEM := authority.GetRootCertificatePEM()

	if len(rootPEM) == 0 {
		t.Error("Root certificate PEM should not be empty")
	}

	// Check PEM format
	if string(rootPEM[:27]) != "-----BEGIN CERTIFICATE-----" {
		t.Error("Root certificate should be in PEM format")
	}
}
