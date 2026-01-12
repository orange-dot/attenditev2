package privacy

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// PostgresDepseudonymizationRepository implements DepseudonymizationRepository using PostgreSQL
type PostgresDepseudonymizationRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDepseudonymizationRepository creates a new PostgreSQL-backed depseudonymization repository
func NewPostgresDepseudonymizationRepository(pool *pgxpool.Pool) *PostgresDepseudonymizationRepository {
	return &PostgresDepseudonymizationRepository{pool: pool}
}

// CreateRequest creates a new depseudonymization request
func (r *PostgresDepseudonymizationRepository) CreateRequest(ctx context.Context, req *DepseudonymizationRequest) error {
	query := `
		INSERT INTO privacy.depseudonymization_requests (
			id, pseudonym_id, requestor_id, requestor_agency_id,
			purpose, legal_basis, justification, case_id,
			requested_at, expires_at, status,
			approved_by, approved_at, rejection_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	var approvedBy *string
	if req.ApprovedBy != nil {
		s := string(*req.ApprovedBy)
		approvedBy = &s
	}

	var caseID *string
	if !req.CaseID.IsZero() {
		s := string(req.CaseID)
		caseID = &s
	}

	_, err := r.pool.Exec(ctx, query,
		req.ID,
		req.PseudonymID,
		req.RequestorID,
		req.RequestorAgency,
		req.Purpose,
		req.LegalBasis,
		req.Justification,
		caseID,
		req.RequestedAt,
		req.ExpiresAt,
		req.Status,
		approvedBy,
		req.ApprovedAt,
		req.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("failed to create depseudonymization request: %w", err)
	}

	return nil
}

// GetRequest retrieves a request by ID
func (r *PostgresDepseudonymizationRepository) GetRequest(ctx context.Context, id types.ID) (*DepseudonymizationRequest, error) {
	query := `
		SELECT id, pseudonym_id, requestor_id, requestor_agency_id,
			   purpose, legal_basis, justification, case_id,
			   requested_at, expires_at, status,
			   approved_by, approved_at, rejection_reason
		FROM privacy.depseudonymization_requests
		WHERE id = $1
	`

	var req DepseudonymizationRequest
	var idStr, requestorID, requestorAgency string
	var caseID *string
	var approvedBy *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&idStr,
		&req.PseudonymID,
		&requestorID,
		&requestorAgency,
		&req.Purpose,
		&req.LegalBasis,
		&req.Justification,
		&caseID,
		&req.RequestedAt,
		&req.ExpiresAt,
		&req.Status,
		&approvedBy,
		&req.ApprovedAt,
		&req.RejectionReason,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get depseudonymization request: %w", err)
	}

	req.ID = types.ID(idStr)
	req.RequestorID = types.ID(requestorID)
	req.RequestorAgency = types.ID(requestorAgency)
	if caseID != nil {
		req.CaseID = types.ID(*caseID)
	}
	if approvedBy != nil {
		ab := types.ID(*approvedBy)
		req.ApprovedBy = &ab
	}

	return &req, nil
}

// UpdateRequest updates an existing request
func (r *PostgresDepseudonymizationRepository) UpdateRequest(ctx context.Context, req *DepseudonymizationRequest) error {
	query := `
		UPDATE privacy.depseudonymization_requests
		SET status = $2, approved_by = $3, approved_at = $4, rejection_reason = $5
		WHERE id = $1
	`

	var approvedBy *string
	if req.ApprovedBy != nil {
		s := string(*req.ApprovedBy)
		approvedBy = &s
	}

	_, err := r.pool.Exec(ctx, query,
		req.ID,
		req.Status,
		approvedBy,
		req.ApprovedAt,
		req.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("failed to update depseudonymization request: %w", err)
	}

	return nil
}

// ListPendingRequests returns all pending requests for an agency
func (r *PostgresDepseudonymizationRepository) ListPendingRequests(ctx context.Context, approverAgency types.ID) ([]*DepseudonymizationRequest, error) {
	query := `
		SELECT id, pseudonym_id, requestor_id, requestor_agency_id,
			   purpose, legal_basis, justification, case_id,
			   requested_at, expires_at, status,
			   approved_by, approved_at, rejection_reason
		FROM privacy.depseudonymization_requests
		WHERE status = 'pending' AND expires_at > NOW()
		ORDER BY requested_at ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending requests: %w", err)
	}
	defer rows.Close()

	var requests []*DepseudonymizationRequest
	for rows.Next() {
		var req DepseudonymizationRequest
		var idStr, requestorID, requestorAgency string
		var caseID *string
		var approvedBy *string

		err := rows.Scan(
			&idStr,
			&req.PseudonymID,
			&requestorID,
			&requestorAgency,
			&req.Purpose,
			&req.LegalBasis,
			&req.Justification,
			&caseID,
			&req.RequestedAt,
			&req.ExpiresAt,
			&req.Status,
			&approvedBy,
			&req.ApprovedAt,
			&req.RejectionReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}

		req.ID = types.ID(idStr)
		req.RequestorID = types.ID(requestorID)
		req.RequestorAgency = types.ID(requestorAgency)
		if caseID != nil {
			req.CaseID = types.ID(*caseID)
		}
		if approvedBy != nil {
			ab := types.ID(*approvedBy)
			req.ApprovedBy = &ab
		}

		requests = append(requests, &req)
	}

	return requests, nil
}

// StoreToken stores a new depseudonymization token
func (r *PostgresDepseudonymizationRepository) StoreToken(ctx context.Context, token *DepseudonymizationToken) error {
	query := `
		INSERT INTO privacy.depseudonymization_tokens (
			token, request_id, pseudonym_id,
			issued_at, expires_at, used_count, max_uses,
			last_used_at, revoked_at, revoked_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var revokedBy *string
	if token.RevokedBy != nil {
		s := string(*token.RevokedBy)
		revokedBy = &s
	}

	_, err := r.pool.Exec(ctx, query,
		token.Token,
		token.RequestID,
		token.PseudonymID,
		token.IssuedAt,
		token.ExpiresAt,
		token.UsedCount,
		token.MaxUses,
		token.LastUsedAt,
		token.RevokedAt,
		revokedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to store depseudonymization token: %w", err)
	}

	return nil
}

// GetToken retrieves a token by token string
func (r *PostgresDepseudonymizationRepository) GetToken(ctx context.Context, tokenStr string) (*DepseudonymizationToken, error) {
	query := `
		SELECT token, request_id, pseudonym_id,
			   issued_at, expires_at, used_count, max_uses,
			   last_used_at, revoked_at, revoked_by
		FROM privacy.depseudonymization_tokens
		WHERE token = $1
	`

	var token DepseudonymizationToken
	var requestID string
	var revokedBy *string

	err := r.pool.QueryRow(ctx, query, tokenStr).Scan(
		&token.Token,
		&requestID,
		&token.PseudonymID,
		&token.IssuedAt,
		&token.ExpiresAt,
		&token.UsedCount,
		&token.MaxUses,
		&token.LastUsedAt,
		&token.RevokedAt,
		&revokedBy,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get depseudonymization token: %w", err)
	}

	token.RequestID = types.ID(requestID)
	if revokedBy != nil {
		rb := types.ID(*revokedBy)
		token.RevokedBy = &rb
	}

	return &token, nil
}

// IncrementTokenUsage increments the usage count of a token
func (r *PostgresDepseudonymizationRepository) IncrementTokenUsage(ctx context.Context, tokenStr string) error {
	query := `
		UPDATE privacy.depseudonymization_tokens
		SET used_count = used_count + 1, last_used_at = $2
		WHERE token = $1
	`

	_, err := r.pool.Exec(ctx, query, tokenStr, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to increment token usage: %w", err)
	}

	return nil
}

// RevokeToken revokes a depseudonymization token
func (r *PostgresDepseudonymizationRepository) RevokeToken(ctx context.Context, tokenStr string, revokedBy types.ID) error {
	query := `
		UPDATE privacy.depseudonymization_tokens
		SET revoked_at = $2, revoked_by = $3
		WHERE token = $1
	`

	_, err := r.pool.Exec(ctx, query, tokenStr, time.Now().UTC(), revokedBy)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}
