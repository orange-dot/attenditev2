package privacy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// DepseudonymizationRepository stores depseudonymization requests and tokens.
type DepseudonymizationRepository interface {
	// Request operations
	CreateRequest(ctx context.Context, req *DepseudonymizationRequest) error
	GetRequest(ctx context.Context, id types.ID) (*DepseudonymizationRequest, error)
	UpdateRequest(ctx context.Context, req *DepseudonymizationRequest) error
	ListPendingRequests(ctx context.Context, approverAgency types.ID) ([]*DepseudonymizationRequest, error)

	// Token operations
	StoreToken(ctx context.Context, token *DepseudonymizationToken) error
	GetToken(ctx context.Context, tokenStr string) (*DepseudonymizationToken, error)
	IncrementTokenUsage(ctx context.Context, tokenStr string) error
	RevokeToken(ctx context.Context, tokenStr string, revokedBy types.ID) error
}

// DepseudonymizationService handles controlled identity revelation.
type DepseudonymizationService struct {
	pseudoSvc   *PseudonymizationService
	repo        DepseudonymizationRepository
	audit       AuditLogger

	// Configuration
	defaultTokenTTL    time.Duration
	maxTokenUses       int
	approvalTimeoutHrs int
}

// DepseudonymizationConfig holds configuration for the service.
type DepseudonymizationConfig struct {
	TokenTTLHours        int
	MaxTokenUses         int
	ApprovalTimeoutHours int
}

// DefaultDepseudonymizationConfig returns default configuration.
func DefaultDepseudonymizationConfig() DepseudonymizationConfig {
	return DepseudonymizationConfig{
		TokenTTLHours:        1,
		MaxTokenUses:         3,
		ApprovalTimeoutHours: 24,
	}
}

// NewDepseudonymizationService creates a new depseudonymization service.
func NewDepseudonymizationService(
	pseudoSvc *PseudonymizationService,
	repo DepseudonymizationRepository,
	audit AuditLogger,
	cfg DepseudonymizationConfig,
) *DepseudonymizationService {
	return &DepseudonymizationService{
		pseudoSvc:          pseudoSvc,
		repo:               repo,
		audit:              audit,
		defaultTokenTTL:    time.Duration(cfg.TokenTTLHours) * time.Hour,
		maxTokenUses:       cfg.MaxTokenUses,
		approvalTimeoutHrs: cfg.ApprovalTimeoutHours,
	}
}

// RequestDepseudonymization creates a request for identity revelation.
func (s *DepseudonymizationService) RequestDepseudonymization(
	ctx context.Context,
	pseudonymID PseudonymID,
	requestorID, requestorAgency types.ID,
	purpose string,
	legalBasis LegalBasis,
	justification string,
	caseID types.ID,
) (*DepseudonymizationRequest, error) {
	// Validate inputs
	if pseudonymID.IsZero() {
		return nil, fmt.Errorf("pseudonym ID is required")
	}
	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}
	if justification == "" {
		return nil, fmt.Errorf("justification is required")
	}
	if len(justification) < 20 {
		return nil, fmt.Errorf("justification must be at least 20 characters")
	}

	// Verify pseudonym exists
	exists, err := s.pseudoSvc.Exists(ctx, pseudonymID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify pseudonym: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("pseudonym not found")
	}

	now := time.Now().UTC()
	req := &DepseudonymizationRequest{
		ID:              types.NewID(),
		PseudonymID:     pseudonymID,
		RequestorID:     requestorID,
		RequestorAgency: requestorAgency,
		Purpose:         purpose,
		LegalBasis:      legalBasis,
		Justification:   justification,
		CaseID:          caseID,
		RequestedAt:     now,
		ExpiresAt:       now.Add(time.Duration(s.approvalTimeoutHrs) * time.Hour),
		Status:          RequestStatusPending,
	}

	// Auto-approve for emergencies
	if !legalBasis.RequiresManualApproval() {
		req.Status = RequestStatusApproved
		systemID := types.ID("system")
		req.ApprovedBy = &systemID
		req.ApprovedAt = &now
	}

	// Store request
	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Audit log
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionDepseudoRequested, "depseudo_request", req.ID, map[string]any{
			"pseudonym_id":  pseudonymID,
			"requestor_id":  requestorID,
			"legal_basis":   legalBasis,
			"auto_approved": !legalBasis.RequiresManualApproval(),
		})
	}

	return req, nil
}

// ApproveRequest approves a pending depseudonymization request.
func (s *DepseudonymizationService) ApproveRequest(
	ctx context.Context,
	requestID types.ID,
	approverID types.ID,
) (*DepseudonymizationToken, error) {
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("request not found: %w", err)
	}

	if req.Status != RequestStatusPending {
		return nil, fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = RequestStatusExpired
		s.repo.UpdateRequest(ctx, req)
		return nil, fmt.Errorf("request has expired")
	}

	// Update request
	now := time.Now().UTC()
	req.Status = RequestStatusApproved
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now

	if err := s.repo.UpdateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to update request: %w", err)
	}

	// Generate token
	token, err := s.generateToken(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Audit log
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionDepseudoApproved, "depseudo_request", req.ID, map[string]any{
			"pseudonym_id": req.PseudonymID,
			"approver_id":  approverID,
			"token_id":     token.Token[:16] + "...",
		})
	}

	return token, nil
}

// RejectRequest rejects a pending depseudonymization request.
func (s *DepseudonymizationService) RejectRequest(
	ctx context.Context,
	requestID types.ID,
	rejectorID types.ID,
	reason string,
) error {
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	if req.Status != RequestStatusPending {
		return fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	now := time.Now().UTC()
	req.Status = RequestStatusRejected
	req.ApprovedBy = &rejectorID
	req.ApprovedAt = &now
	req.RejectionReason = reason

	if err := s.repo.UpdateRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Audit log
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionDepseudoRejected, "depseudo_request", req.ID, map[string]any{
			"pseudonym_id": req.PseudonymID,
			"rejector_id":  rejectorID,
			"reason":       reason,
		})
	}

	return nil
}

// Depseudonymize reveals the real JMBG using a valid token.
func (s *DepseudonymizationService) Depseudonymize(
	ctx context.Context,
	tokenStr string,
) (string, error) {
	token, err := s.repo.GetToken(ctx, tokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	if !token.IsValid() {
		if token.RevokedAt != nil {
			return "", fmt.Errorf("token has been revoked")
		}
		if time.Now().After(token.ExpiresAt) {
			return "", fmt.Errorf("token has expired")
		}
		if token.UsedCount >= token.MaxUses {
			return "", fmt.Errorf("token usage limit exceeded")
		}
		return "", fmt.Errorf("token is not valid")
	}

	// Increment usage
	if err := s.repo.IncrementTokenUsage(ctx, tokenStr); err != nil {
		return "", fmt.Errorf("failed to update token usage: %w", err)
	}

	// Get real JMBG from local mapping
	mapping, err := s.pseudoSvc.repo.GetByPseudonymID(ctx, token.PseudonymID)
	if err != nil || mapping == nil {
		return "", fmt.Errorf("mapping not found")
	}

	// Decrypt JMBG
	var jmbg string
	if s.pseudoSvc.encryptor != nil && len(mapping.JMBGEncrypted) > 0 {
		// Dev mode: decrypt using AES-GCM
		jmbg, err = s.pseudoSvc.encryptor.Decrypt(mapping.JMBGEncrypted)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt JMBG: %w", err)
		}
	} else if len(mapping.JMBGEncrypted) > 0 {
		// Production mode: would use HSM here
		// TODO: Implement HSM-based decryption for production
		return "", fmt.Errorf("JMBG decryption requires HSM integration - enable dev mode encryptor for testing")
	} else {
		// No encrypted JMBG stored (legacy mapping or test data)
		return "", fmt.Errorf("no encrypted JMBG available for this mapping")
	}

	// Audit log - ALWAYS log JMBG access
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionDepseudoUsed, "depseudo_token", types.ID(tokenStr[:32]), map[string]any{
			"pseudonym_id": token.PseudonymID,
			"request_id":   token.RequestID,
			"usage_count":  token.UsedCount + 1,
			"jmbg_masked":  MaskJMBG(jmbg), // Log masked version for audit trail
		})
	}

	return jmbg, nil
}

// RevokeToken revokes a depseudonymization token.
func (s *DepseudonymizationService) RevokeToken(
	ctx context.Context,
	tokenStr string,
	revokedBy types.ID,
) error {
	if err := s.repo.RevokeToken(ctx, tokenStr, revokedBy); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Audit log
	if s.audit != nil {
		s.audit.Log(ctx, AuditActionDepseudoRevoked, "depseudo_token", types.ID(tokenStr[:32]), map[string]any{
			"revoked_by": revokedBy,
		})
	}

	return nil
}

// ListPendingRequests returns all pending requests for an agency.
func (s *DepseudonymizationService) ListPendingRequests(
	ctx context.Context,
	approverAgency types.ID,
) ([]*DepseudonymizationRequest, error) {
	return s.repo.ListPendingRequests(ctx, approverAgency)
}

// GetRequest retrieves a request by ID.
func (s *DepseudonymizationService) GetRequest(
	ctx context.Context,
	requestID types.ID,
) (*DepseudonymizationRequest, error) {
	return s.repo.GetRequest(ctx, requestID)
}

// generateToken creates a secure depseudonymization token.
func (s *DepseudonymizationService) generateToken(ctx context.Context, req *DepseudonymizationRequest) (*DepseudonymizationToken, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	now := time.Now().UTC()
	token := &DepseudonymizationToken{
		Token:       "DPT-" + hex.EncodeToString(tokenBytes),
		RequestID:   req.ID,
		PseudonymID: req.PseudonymID,
		IssuedAt:    now,
		ExpiresAt:   now.Add(s.defaultTokenTTL),
		UsedCount:   0,
		MaxUses:     s.maxTokenUses,
	}

	if err := s.repo.StoreToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return token, nil
}

// GenerateTokenForApprovedRequest generates a token for an already approved request.
func (s *DepseudonymizationService) GenerateTokenForApprovedRequest(
	ctx context.Context,
	requestID types.ID,
) (*DepseudonymizationToken, error) {
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("request not found: %w", err)
	}

	if req.Status != RequestStatusApproved {
		return nil, fmt.Errorf("request is not approved (status: %s)", req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		return nil, fmt.Errorf("request has expired")
	}

	return s.generateToken(ctx, req)
}
