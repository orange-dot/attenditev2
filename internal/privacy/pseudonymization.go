package privacy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// PseudonymRepository defines storage operations for pseudonym mappings.
// This runs ONLY at the local facility level.
type PseudonymRepository interface {
	// Store saves a new pseudonym mapping.
	Store(ctx context.Context, mapping *PseudonymMapping) error

	// GetByJMBGHash retrieves a mapping by JMBG hash.
	GetByJMBGHash(ctx context.Context, jmbgHash, facilityCode string) (*PseudonymMapping, error)

	// GetByPseudonymID retrieves a mapping by pseudonym ID.
	GetByPseudonymID(ctx context.Context, pseudonymID PseudonymID) (*PseudonymMapping, error)

	// Delete removes a mapping (for GDPR right to erasure).
	Delete(ctx context.Context, pseudonymID PseudonymID) error
}

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	Log(ctx context.Context, action string, resourceType string, resourceID types.ID, data map[string]any) error
}

// PseudonymizationService handles JMBG to PseudonymID conversion.
// This service runs ONLY at the LOCAL facility level - never at central.
type PseudonymizationService struct {
	// HMAC key - unique per facility, should be stored in HSM in production
	hmacKey      []byte
	facilityCode string

	// Local mapping cache (in-memory for performance)
	cache   map[string]PseudonymID
	cacheMu sync.RWMutex

	// Repository for persistent storage
	repo PseudonymRepository

	// Audit logger
	audit AuditLogger
}

// NewPseudonymizationService creates a new pseudonymization service.
// hmacKey should come from HSM or secure key management system.
func NewPseudonymizationService(
	hmacKey []byte,
	facilityCode string,
	repo PseudonymRepository,
	audit AuditLogger,
) *PseudonymizationService {
	return &PseudonymizationService{
		hmacKey:      hmacKey,
		facilityCode: facilityCode,
		cache:        make(map[string]PseudonymID),
		repo:         repo,
		audit:        audit,
	}
}

// Pseudonymize converts a JMBG to a PseudonymID.
// This is a deterministic one-way function using HMAC-SHA256.
// The same JMBG will always produce the same pseudonym (within the same facility).
func (s *PseudonymizationService) Pseudonymize(ctx context.Context, jmbg string) (PseudonymID, error) {
	if jmbg == "" {
		return "", fmt.Errorf("JMBG cannot be empty")
	}

	// Compute JMBG hash for lookup
	jmbgHash := s.hashJMBG(jmbg)

	// Check cache first
	s.cacheMu.RLock()
	if cached, ok := s.cache[jmbgHash]; ok {
		s.cacheMu.RUnlock()
		return cached, nil
	}
	s.cacheMu.RUnlock()

	// Check persistent storage
	existing, err := s.repo.GetByJMBGHash(ctx, jmbgHash, s.facilityCode)
	if err == nil && existing != nil {
		s.cacheMu.Lock()
		s.cache[jmbgHash] = existing.PseudonymID
		s.cacheMu.Unlock()
		return existing.PseudonymID, nil
	}

	// Generate new pseudonym using HMAC-SHA256
	pseudonymID := s.generatePseudonym(jmbg)

	// Create mapping
	mapping := &PseudonymMapping{
		ID:           types.NewID(),
		JMBGHash:     jmbgHash,
		PseudonymID:  pseudonymID,
		FacilityCode: s.facilityCode,
		CreatedAt:    time.Now().UTC(),
	}

	// Store mapping locally (never sent to central)
	if err := s.repo.Store(ctx, mapping); err != nil {
		return "", fmt.Errorf("failed to store mapping: %w", err)
	}

	// Cache for performance
	s.cacheMu.Lock()
	s.cache[jmbgHash] = pseudonymID
	s.cacheMu.Unlock()

	// Audit log
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionPseudonymCreated, "pseudonym", mapping.ID, map[string]any{
			"pseudonym_id":  pseudonymID,
			"facility_code": s.facilityCode,
		})
	}

	return pseudonymID, nil
}

// PseudonymizeMany converts multiple JMBGs efficiently.
func (s *PseudonymizationService) PseudonymizeMany(ctx context.Context, jmbgs []string) (map[string]PseudonymID, error) {
	result := make(map[string]PseudonymID, len(jmbgs))
	for _, jmbg := range jmbgs {
		pseudonym, err := s.Pseudonymize(ctx, jmbg)
		if err != nil {
			return nil, fmt.Errorf("failed to pseudonymize: %w", err)
		}
		result[jmbg] = pseudonym
	}
	return result, nil
}

// GetJMBG retrieves the real JMBG from a pseudonym ID.
// This should only be called after proper authorization.
func (s *PseudonymizationService) GetJMBG(ctx context.Context, pseudonymID PseudonymID) (string, error) {
	mapping, err := s.repo.GetByPseudonymID(ctx, pseudonymID)
	if err != nil {
		return "", fmt.Errorf("mapping not found: %w", err)
	}
	if mapping == nil {
		return "", fmt.Errorf("pseudonym not found")
	}

	// Decrypt JMBG from encrypted storage
	// In a real implementation, this would use the HSM to decrypt
	// For now, we just return an error as encrypted storage is not implemented
	return "", fmt.Errorf("JMBG decryption not implemented - use DepseudonymizationService")
}

// Exists checks if a pseudonym exists.
func (s *PseudonymizationService) Exists(ctx context.Context, pseudonymID PseudonymID) (bool, error) {
	mapping, err := s.repo.GetByPseudonymID(ctx, pseudonymID)
	if err != nil {
		return false, err
	}
	return mapping != nil, nil
}

// Delete removes a pseudonym mapping (for GDPR right to erasure).
func (s *PseudonymizationService) Delete(ctx context.Context, pseudonymID PseudonymID) error {
	// Get mapping first for cache invalidation
	mapping, err := s.repo.GetByPseudonymID(ctx, pseudonymID)
	if err != nil {
		return err
	}
	if mapping == nil {
		return nil // Already deleted
	}

	// Delete from repository
	if err := s.repo.Delete(ctx, pseudonymID); err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}

	// Invalidate cache
	s.cacheMu.Lock()
	delete(s.cache, mapping.JMBGHash)
	s.cacheMu.Unlock()

	return nil
}

// FacilityCode returns the facility code for this service.
func (s *PseudonymizationService) FacilityCode() string {
	return s.facilityCode
}

// hashJMBG creates a SHA-256 hash of the JMBG for lookup purposes.
func (s *PseudonymizationService) hashJMBG(jmbg string) string {
	hash := sha256.Sum256([]byte(jmbg))
	return hex.EncodeToString(hash[:])
}

// generatePseudonym creates a new pseudonym using HMAC-SHA256.
func (s *PseudonymizationService) generatePseudonym(jmbg string) PseudonymID {
	// Include facility code to ensure different facilities generate different pseudonyms
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(s.facilityCode + ":" + jmbg))
	hash := mac.Sum(nil)

	// Format: PSE-<first 32 hex chars of hash>
	return PseudonymID("PSE-" + hex.EncodeToString(hash)[:32])
}

// MaskJMBG returns a masked version of JMBG for display purposes.
// Shows first 7 digits (birthdate + region) and hides the rest.
func MaskJMBG(jmbg string) string {
	if len(jmbg) < 13 {
		return "***********"
	}
	return jmbg[:7] + "******"
}

// MaskPhone returns a masked version of a phone number.
func MaskPhone(phone string) string {
	if len(phone) < 4 {
		return "****"
	}
	return "***-***-" + phone[len(phone)-4:]
}

// MaskEmail returns a masked version of an email address.
func MaskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***@***"
	}
	if at <= 2 {
		return email[:1] + "***@" + email[at+1:]
	}
	return email[:2] + "***@" + email[at+1:]
}

// MaskName returns a masked version of a name.
func MaskName(name string) string {
	if len(name) <= 1 {
		return "*"
	}
	return string(name[0]) + "." + " ***"
}
