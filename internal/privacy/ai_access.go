package privacy

import (
	"context"
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// AIAccessRepository stores AI access requests.
type AIAccessRepository interface {
	CreateRequest(ctx context.Context, req *AIAccessRequest) error
	GetRequest(ctx context.Context, id types.ID) (*AIAccessRequest, error)
	GetActiveAccess(ctx context.Context, aiSystemID string) (*AIAccessRequest, error)
	UpdateRequest(ctx context.Context, req *AIAccessRequest) error
	ListPendingRequests(ctx context.Context) ([]*AIAccessRequest, error)
}

// AIAccessController manages AI access to data.
type AIAccessController struct {
	repo         AIAccessRepository
	audit        AuditLogger
	defaultLevel DataAccessLevel
	defaultTTL   time.Duration
}

// AIAccessConfig holds configuration for AI access control.
type AIAccessConfig struct {
	DefaultLevel       DataAccessLevel
	DefaultTTLHours    int
	MaxRecordsLevel1   int
	MaxRecordsLevel2   int
}

// DefaultAIAccessConfig returns default configuration.
func DefaultAIAccessConfig() AIAccessConfig {
	return AIAccessConfig{
		DefaultLevel:     DataAccessLevelAggregated,
		DefaultTTLHours:  24,
		MaxRecordsLevel1: 1000,
		MaxRecordsLevel2: 100,
	}
}

// NewAIAccessController creates a new AI access controller.
func NewAIAccessController(repo AIAccessRepository, audit AuditLogger, cfg AIAccessConfig) *AIAccessController {
	return &AIAccessController{
		repo:         repo,
		audit:        audit,
		defaultLevel: cfg.DefaultLevel,
		defaultTTL:   time.Duration(cfg.DefaultTTLHours) * time.Hour,
	}
}

// GetAccessLevel returns the current access level for an AI system.
func (c *AIAccessController) GetAccessLevel(ctx context.Context, aiSystemID string) (DataAccessLevel, error) {
	activeAccess, err := c.repo.GetActiveAccess(ctx, aiSystemID)
	if err != nil {
		return c.defaultLevel, nil
	}
	if activeAccess == nil {
		return c.defaultLevel, nil
	}

	if !activeAccess.IsActive() {
		return c.defaultLevel, nil
	}

	return activeAccess.RequestedLevel, nil
}

// RequestElevatedAccess requests higher access level for an AI system.
func (c *AIAccessController) RequestElevatedAccess(
	ctx context.Context,
	aiSystemID string,
	level DataAccessLevel,
	purpose string,
	scope DataScope,
	requestorID types.ID,
) (*AIAccessRequest, error) {
	if level == DataAccessLevelAggregated {
		return nil, fmt.Errorf("no request needed for aggregated access")
	}

	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	now := time.Now().UTC()
	req := &AIAccessRequest{
		ID:             types.NewID(),
		AISystemID:     aiSystemID,
		RequestedLevel: level,
		Purpose:        purpose,
		DataScope:      scope,
		RequestedBy:    requestorID,
		RequestedAt:    now,
		ExpiresAt:      now.Add(c.defaultTTL),
		Status:         RequestStatusPending,
	}

	if err := c.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Audit log
	if c.audit != nil {
		c.audit.Log(ctx, AuditActionAIAccessRequested, "ai_access", req.ID, map[string]any{
			"ai_system_id":    aiSystemID,
			"requested_level": level.String(),
			"requestor_id":    requestorID,
		})
	}

	return req, nil
}

// ApproveAccess approves an AI access request.
func (c *AIAccessController) ApproveAccess(
	ctx context.Context,
	requestID types.ID,
	approverID types.ID,
) error {
	req, err := c.repo.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	if req.Status != RequestStatusPending {
		return fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	now := time.Now().UTC()
	req.Status = RequestStatusApproved
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now

	if err := c.repo.UpdateRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Audit log
	if c.audit != nil {
		c.audit.Log(ctx, AuditActionAIAccessGranted, "ai_access", req.ID, map[string]any{
			"ai_system_id":   req.AISystemID,
			"granted_level":  req.RequestedLevel.String(),
			"approver_id":    approverID,
			"expires_at":     req.ExpiresAt,
		})
	}

	return nil
}

// DenyAccess denies an AI access request.
func (c *AIAccessController) DenyAccess(
	ctx context.Context,
	requestID types.ID,
	denierID types.ID,
	reason string,
) error {
	req, err := c.repo.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	if req.Status != RequestStatusPending {
		return fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	now := time.Now().UTC()
	req.Status = RequestStatusRejected
	req.ApprovedBy = &denierID
	req.ApprovedAt = &now

	if err := c.repo.UpdateRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Audit log
	if c.audit != nil {
		c.audit.Log(ctx, AuditActionAIAccessDenied, "ai_access", req.ID, map[string]any{
			"ai_system_id": req.AISystemID,
			"denier_id":    denierID,
			"reason":       reason,
		})
	}

	return nil
}

// RevokeAccess revokes an approved AI access.
func (c *AIAccessController) RevokeAccess(
	ctx context.Context,
	aiSystemID string,
	revokerID types.ID,
	reason string,
) error {
	activeAccess, err := c.repo.GetActiveAccess(ctx, aiSystemID)
	if err != nil || activeAccess == nil {
		return fmt.Errorf("no active access found")
	}

	activeAccess.Status = RequestStatusRevoked

	if err := c.repo.UpdateRequest(ctx, activeAccess); err != nil {
		return fmt.Errorf("failed to revoke access: %w", err)
	}

	// Audit log
	if c.audit != nil {
		c.audit.Log(ctx, AuditActionAIAccessDenied, "ai_access", activeAccess.ID, map[string]any{
			"ai_system_id": aiSystemID,
			"revoker_id":   revokerID,
			"reason":       reason,
			"action":       "revoked",
		})
	}

	return nil
}

// ListPendingRequests returns all pending AI access requests.
func (c *AIAccessController) ListPendingRequests(ctx context.Context) ([]*AIAccessRequest, error) {
	return c.repo.ListPendingRequests(ctx)
}

// FilterDataForAI filters data based on AI access level.
// Returns the data with appropriate fields removed/masked based on access level.
func (c *AIAccessController) FilterDataForAI(
	ctx context.Context,
	aiSystemID string,
	data map[string]any,
) (map[string]any, error) {
	level, err := c.GetAccessLevel(ctx, aiSystemID)
	if err != nil {
		level = c.defaultLevel
	}

	switch level {
	case DataAccessLevelAggregated:
		return c.aggregateData(data), nil
	case DataAccessLevelPseudonymized:
		return c.pseudonymizeData(data), nil
	case DataAccessLevelLinkable:
		// Full access but PII should still be pseudonymized
		return c.pseudonymizeData(data), nil
	default:
		return c.aggregateData(data), nil
	}
}

// aggregateData returns only statistical summaries.
func (c *AIAccessController) aggregateData(data map[string]any) map[string]any {
	// Remove all individual identifiers
	result := make(map[string]any)

	for key, value := range data {
		// Skip PII fields
		if isPIIField(key) {
			continue
		}

		// Keep aggregated/statistical fields
		if isAggregateField(key) {
			result[key] = value
		}
	}

	result["_access_level"] = "aggregated"
	result["_filtered"] = true

	return result
}

// pseudonymizeData returns data with PII replaced by pseudonyms.
func (c *AIAccessController) pseudonymizeData(data map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range data {
		// Replace PII fields with pseudonymized values
		if isPIIField(key) {
			switch key {
			case "jmbg", "personal_id":
				result["pseudonym_id"] = "[PSEUDONYMIZED]"
			case "name", "first_name", "last_name":
				result[key+"_redacted"] = true
			case "email", "phone", "address":
				result[key+"_redacted"] = true
			default:
				// Skip other PII
			}
			continue
		}

		result[key] = value
	}

	result["_access_level"] = "pseudonymized"
	result["_filtered"] = true

	return result
}

// isPIIField checks if a field name typically contains PII.
func isPIIField(field string) bool {
	piiFields := map[string]bool{
		"jmbg":        true,
		"personal_id": true,
		"name":        true,
		"first_name":  true,
		"last_name":   true,
		"email":       true,
		"phone":       true,
		"mobile":      true,
		"address":     true,
		"street":      true,
		"city":        true,
		"postal_code": true,
		"lbo":         true,
		"citizen_id":  true,
	}
	return piiFields[field]
}

// isAggregateField checks if a field is suitable for aggregated access.
func isAggregateField(field string) bool {
	aggregateFields := map[string]bool{
		"count":            true,
		"total":            true,
		"average":          true,
		"sum":              true,
		"min":              true,
		"max":              true,
		"type":             true,
		"status":           true,
		"priority":         true,
		"created_at":       true,
		"updated_at":       true,
		"agency_id":        true,
		"case_type":        true,
		"risk_level":       true,
		"region":           true,
		"age_group":        true,
		"gender":           true,
		"is_minor":         true,
		"has_open_cases":   true,
		"is_beneficiary":   true,
	}
	return aggregateFields[field]
}
