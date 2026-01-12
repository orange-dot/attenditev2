package trust

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed trust authority repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// SaveAgency saves a trusted agency
func (r *PostgresRepository) SaveAgency(ctx context.Context, agency *TrustedAgency) error {
	query := `
		INSERT INTO federation.trusted_agencies (
			id, name, code, gateway_url, public_key, certificate,
			status, registered_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (code) DO UPDATE SET
			name = EXCLUDED.name,
			gateway_url = EXCLUDED.gateway_url,
			public_key = EXCLUDED.public_key,
			certificate = EXCLUDED.certificate,
			status = EXCLUDED.status,
			last_seen_at = EXCLUDED.last_seen_at
	`

	_, err := r.pool.Exec(ctx, query,
		agency.ID,
		agency.Name,
		agency.Code,
		agency.GatewayURL,
		agency.PublicKey,
		agency.Certificate,
		agency.Status,
		agency.RegisteredAt,
		agency.LastSeenAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save trusted agency: %w", err)
	}

	return nil
}

// GetAgency retrieves an agency by ID
func (r *PostgresRepository) GetAgency(ctx context.Context, id types.ID) (*TrustedAgency, error) {
	query := `
		SELECT id, name, code, gateway_url, public_key, certificate,
			   status, registered_at, last_seen_at
		FROM federation.trusted_agencies
		WHERE id = $1
	`

	var agency TrustedAgency
	var idStr string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&idStr,
		&agency.Name,
		&agency.Code,
		&agency.GatewayURL,
		&agency.PublicKey,
		&agency.Certificate,
		&agency.Status,
		&agency.RegisteredAt,
		&agency.LastSeenAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get trusted agency: %w", err)
	}

	agency.ID = types.ID(idStr)
	return &agency, nil
}

// GetAgencyByCode retrieves an agency by code
func (r *PostgresRepository) GetAgencyByCode(ctx context.Context, code string) (*TrustedAgency, error) {
	query := `
		SELECT id, name, code, gateway_url, public_key, certificate,
			   status, registered_at, last_seen_at
		FROM federation.trusted_agencies
		WHERE code = $1
	`

	var agency TrustedAgency
	var idStr string

	err := r.pool.QueryRow(ctx, query, code).Scan(
		&idStr,
		&agency.Name,
		&agency.Code,
		&agency.GatewayURL,
		&agency.PublicKey,
		&agency.Certificate,
		&agency.Status,
		&agency.RegisteredAt,
		&agency.LastSeenAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get trusted agency by code: %w", err)
	}

	agency.ID = types.ID(idStr)
	return &agency, nil
}

// ListAgencies lists all registered agencies
func (r *PostgresRepository) ListAgencies(ctx context.Context) ([]TrustedAgency, error) {
	query := `
		SELECT id, name, code, gateway_url, public_key, certificate,
			   status, registered_at, last_seen_at
		FROM federation.trusted_agencies
		ORDER BY name ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list trusted agencies: %w", err)
	}
	defer rows.Close()

	var agencies []TrustedAgency
	for rows.Next() {
		var agency TrustedAgency
		var idStr string

		err := rows.Scan(
			&idStr,
			&agency.Name,
			&agency.Code,
			&agency.GatewayURL,
			&agency.PublicKey,
			&agency.Certificate,
			&agency.Status,
			&agency.RegisteredAt,
			&agency.LastSeenAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trusted agency: %w", err)
		}

		agency.ID = types.ID(idStr)
		agencies = append(agencies, agency)
	}

	return agencies, nil
}

// UpdateAgency updates an existing agency
func (r *PostgresRepository) UpdateAgency(ctx context.Context, agency *TrustedAgency) error {
	query := `
		UPDATE federation.trusted_agencies
		SET name = $2, gateway_url = $3, public_key = $4, certificate = $5,
			status = $6, last_seen_at = $7
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		agency.ID,
		agency.Name,
		agency.GatewayURL,
		agency.PublicKey,
		agency.Certificate,
		agency.Status,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to update trusted agency: %w", err)
	}

	return nil
}

// DeleteAgency deletes an agency by ID
func (r *PostgresRepository) DeleteAgency(ctx context.Context, id types.ID) error {
	query := `DELETE FROM federation.trusted_agencies WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete trusted agency: %w", err)
	}

	return nil
}

// SaveService saves a service endpoint
func (r *PostgresRepository) SaveService(ctx context.Context, service *ServiceEndpoint) error {
	query := `
		INSERT INTO federation.service_endpoints (
			id, agency_id, service_type, path, version, active
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (agency_id, service_type, version) DO UPDATE SET
			path = EXCLUDED.path,
			active = EXCLUDED.active
	`

	_, err := r.pool.Exec(ctx, query,
		service.ID,
		service.AgencyID,
		service.ServiceType,
		service.Path,
		service.Version,
		service.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to save service endpoint: %w", err)
	}

	return nil
}

// GetServices gets all services for an agency
func (r *PostgresRepository) GetServices(ctx context.Context, agencyID types.ID) ([]ServiceEndpoint, error) {
	query := `
		SELECT id, agency_id, service_type, path, version, active
		FROM federation.service_endpoints
		WHERE agency_id = $1
		ORDER BY service_type, version
	`

	rows, err := r.pool.Query(ctx, query, agencyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	defer rows.Close()

	var services []ServiceEndpoint
	for rows.Next() {
		var service ServiceEndpoint
		var idStr, agencyIDStr string

		err := rows.Scan(
			&idStr,
			&agencyIDStr,
			&service.ServiceType,
			&service.Path,
			&service.Version,
			&service.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service endpoint: %w", err)
		}

		service.ID = types.ID(idStr)
		service.AgencyID = types.ID(agencyIDStr)
		services = append(services, service)
	}

	return services, nil
}

// GetServicesByType gets all services of a specific type
func (r *PostgresRepository) GetServicesByType(ctx context.Context, serviceType string) ([]ServiceEndpoint, error) {
	query := `
		SELECT id, agency_id, service_type, path, version, active
		FROM federation.service_endpoints
		WHERE service_type = $1 AND active = true
		ORDER BY agency_id, version
	`

	rows, err := r.pool.Query(ctx, query, serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get services by type: %w", err)
	}
	defer rows.Close()

	var services []ServiceEndpoint
	for rows.Next() {
		var service ServiceEndpoint
		var idStr, agencyIDStr string

		err := rows.Scan(
			&idStr,
			&agencyIDStr,
			&service.ServiceType,
			&service.Path,
			&service.Version,
			&service.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service endpoint: %w", err)
		}

		service.ID = types.ID(idStr)
		service.AgencyID = types.ID(agencyIDStr)
		services = append(services, service)
	}

	return services, nil
}
