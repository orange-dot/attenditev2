package trust

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// TrustedAgency represents an agency registered with the Trust Authority
type TrustedAgency struct {
	ID           types.ID  `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"` // e.g., "MUP", "APR"
	GatewayURL   string    `json:"gateway_url"`
	PublicKey    []byte    `json:"public_key"`
	Certificate  []byte    `json:"certificate"`
	Status       string    `json:"status"` // active, suspended, revoked
	RegisteredAt time.Time `json:"registered_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

// ServiceEndpoint represents a service offered by an agency
type ServiceEndpoint struct {
	ID          types.ID `json:"id"`
	AgencyID    types.ID `json:"agency_id"`
	ServiceType string   `json:"service_type"` // e.g., "case.share", "document.verify"
	Path        string   `json:"path"`
	Version     string   `json:"version"`
	Active      bool     `json:"active"`
}

// Authority manages trust relationships between agencies
type Authority struct {
	mu         sync.RWMutex
	rootKey    ed25519.PrivateKey
	rootCert   *x509.Certificate
	agencies   map[types.ID]*TrustedAgency
	services   map[types.ID][]ServiceEndpoint
	repository Repository
}

// Repository interface for Trust Authority persistence
type Repository interface {
	SaveAgency(ctx context.Context, agency *TrustedAgency) error
	GetAgency(ctx context.Context, id types.ID) (*TrustedAgency, error)
	GetAgencyByCode(ctx context.Context, code string) (*TrustedAgency, error)
	ListAgencies(ctx context.Context) ([]TrustedAgency, error)
	UpdateAgency(ctx context.Context, agency *TrustedAgency) error
	DeleteAgency(ctx context.Context, id types.ID) error

	SaveService(ctx context.Context, service *ServiceEndpoint) error
	GetServices(ctx context.Context, agencyID types.ID) ([]ServiceEndpoint, error)
	GetServicesByType(ctx context.Context, serviceType string) ([]ServiceEndpoint, error)
}

// NewAuthority creates a new Trust Authority
func NewAuthority(repo Repository) (*Authority, error) {
	// Generate root CA keypair for MVP (in production, use HSM)
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate root key: %w", err)
	}

	// Create self-signed root certificate
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Serbia Government"},
			Country:       []string{"RS"},
			Province:      []string{"Belgrade"},
			Locality:      []string{"Belgrade"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "Serbia Gov Interoperability Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		MaxPathLen:            2,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pubKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root certificate: %w", err)
	}

	return &Authority{
		rootKey:    privKey,
		rootCert:   cert,
		agencies:   make(map[types.ID]*TrustedAgency),
		services:   make(map[types.ID][]ServiceEndpoint),
		repository: repo,
	}, nil
}

// RegisterAgency registers a new agency and issues a certificate
func (a *Authority) RegisterAgency(ctx context.Context, name, code, gatewayURL string) (*TrustedAgency, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if agency code already exists
	if a.repository != nil {
		existing, _ := a.repository.GetAgencyByCode(ctx, code)
		if existing != nil {
			return nil, fmt.Errorf("agency with code %s already registered", code)
		}
	}

	// Generate keypair for agency
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agency key: %w", err)
	}

	// Issue certificate for agency
	cert, err := a.issueCertificate(name, code, pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	agency := &TrustedAgency{
		ID:           types.NewID(),
		Name:         name,
		Code:         code,
		GatewayURL:   gatewayURL,
		PublicKey:    pubKey,
		Certificate:  cert,
		Status:       "active",
		RegisteredAt: time.Now(),
		LastSeenAt:   time.Now(),
	}

	a.agencies[agency.ID] = agency

	if a.repository != nil {
		if err := a.repository.SaveAgency(ctx, agency); err != nil {
			return nil, fmt.Errorf("failed to save agency: %w", err)
		}
	}

	return agency, nil
}

// issueCertificate issues a certificate for an agency
func (a *Authority) issueCertificate(name, code string, pubKey ed25519.PublicKey) ([]byte, error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Serbia Government - " + name},
			Country:      []string{"RS"},
			CommonName:   code + ".gov.rs",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // 1 year validity
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, a.rootCert, pubKey, a.rootKey)
	if err != nil {
		return nil, err
	}

	// Encode as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return certPEM, nil
}

// GetAgency retrieves an agency by ID
func (a *Authority) GetAgency(ctx context.Context, id types.ID) (*TrustedAgency, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if agency, ok := a.agencies[id]; ok {
		return agency, nil
	}

	if a.repository != nil {
		return a.repository.GetAgency(ctx, id)
	}

	return nil, fmt.Errorf("agency not found: %s", id)
}

// GetAgencyByCode retrieves an agency by code
func (a *Authority) GetAgencyByCode(ctx context.Context, code string) (*TrustedAgency, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, agency := range a.agencies {
		if agency.Code == code {
			return agency, nil
		}
	}

	if a.repository != nil {
		return a.repository.GetAgencyByCode(ctx, code)
	}

	return nil, fmt.Errorf("agency not found: %s", code)
}

// ListAgencies lists all registered agencies
func (a *Authority) ListAgencies(ctx context.Context) ([]TrustedAgency, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.repository != nil {
		return a.repository.ListAgencies(ctx)
	}

	agencies := make([]TrustedAgency, 0, len(a.agencies))
	for _, agency := range a.agencies {
		agencies = append(agencies, *agency)
	}
	return agencies, nil
}

// SuspendAgency suspends an agency's trust status
func (a *Authority) SuspendAgency(ctx context.Context, id types.ID, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agency, ok := a.agencies[id]
	if !ok {
		if a.repository != nil {
			var err error
			agency, err = a.repository.GetAgency(ctx, id)
			if err != nil {
				return fmt.Errorf("agency not found: %s", id)
			}
		} else {
			return fmt.Errorf("agency not found: %s", id)
		}
	}

	agency.Status = "suspended"

	if a.repository != nil {
		return a.repository.UpdateAgency(ctx, agency)
	}

	return nil
}

// RevokeAgency permanently revokes an agency's trust
func (a *Authority) RevokeAgency(ctx context.Context, id types.ID, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agency, ok := a.agencies[id]
	if !ok {
		if a.repository != nil {
			var err error
			agency, err = a.repository.GetAgency(ctx, id)
			if err != nil {
				return fmt.Errorf("agency not found: %s", id)
			}
		} else {
			return fmt.Errorf("agency not found: %s", id)
		}
	}

	agency.Status = "revoked"

	if a.repository != nil {
		return a.repository.UpdateAgency(ctx, agency)
	}

	return nil
}

// RegisterService registers a service endpoint for an agency
func (a *Authority) RegisterService(ctx context.Context, agencyID types.ID, serviceType, path, version string) (*ServiceEndpoint, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	service := &ServiceEndpoint{
		ID:          types.NewID(),
		AgencyID:    agencyID,
		ServiceType: serviceType,
		Path:        path,
		Version:     version,
		Active:      true,
	}

	a.services[agencyID] = append(a.services[agencyID], *service)

	if a.repository != nil {
		if err := a.repository.SaveService(ctx, service); err != nil {
			return nil, err
		}
	}

	return service, nil
}

// GetServices gets all services for an agency
func (a *Authority) GetServices(ctx context.Context, agencyID types.ID) ([]ServiceEndpoint, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.repository != nil {
		return a.repository.GetServices(ctx, agencyID)
	}

	return a.services[agencyID], nil
}

// FindService finds agencies providing a specific service
func (a *Authority) FindService(ctx context.Context, serviceType string) ([]ServiceEndpoint, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.repository != nil {
		return a.repository.GetServicesByType(ctx, serviceType)
	}

	var results []ServiceEndpoint
	for _, services := range a.services {
		for _, svc := range services {
			if svc.ServiceType == serviceType && svc.Active {
				results = append(results, svc)
			}
		}
	}
	return results, nil
}

// VerifyCertificate verifies an agency's certificate
func (a *Authority) VerifyCertificate(certPEM []byte) error {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Verify against root CA
	roots := x509.NewCertPool()
	roots.AddCert(a.rootCert)

	_, err = cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})

	return err
}

// GetRootCertificatePEM returns the root CA certificate in PEM format
func (a *Authority) GetRootCertificatePEM() []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: a.rootCert.Raw,
	})
}
