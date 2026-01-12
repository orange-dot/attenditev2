package privacy

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// PostgresPseudonymRepository implements PseudonymRepository using PostgreSQL
type PostgresPseudonymRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPseudonymRepository creates a new PostgreSQL-backed pseudonym repository
func NewPostgresPseudonymRepository(pool *pgxpool.Pool) *PostgresPseudonymRepository {
	return &PostgresPseudonymRepository{pool: pool}
}

// Store saves a new pseudonym mapping
func (r *PostgresPseudonymRepository) Store(ctx context.Context, mapping *PseudonymMapping) error {
	query := `
		INSERT INTO privacy.pseudonym_mappings (id, jmbg_hash, jmbg_encrypted, pseudonym_id, facility_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (jmbg_hash, facility_code) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query,
		mapping.ID,
		mapping.JMBGHash,
		mapping.JMBGEncrypted,
		mapping.PseudonymID,
		mapping.FacilityCode,
		mapping.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store pseudonym mapping: %w", err)
	}

	return nil
}

// GetByJMBGHash retrieves a mapping by JMBG hash
func (r *PostgresPseudonymRepository) GetByJMBGHash(ctx context.Context, jmbgHash, facilityCode string) (*PseudonymMapping, error) {
	query := `
		SELECT id, jmbg_hash, jmbg_encrypted, pseudonym_id, facility_code, created_at
		FROM privacy.pseudonym_mappings
		WHERE jmbg_hash = $1 AND facility_code = $2
	`

	var mapping PseudonymMapping
	var id string
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, jmbgHash, facilityCode).Scan(
		&id,
		&mapping.JMBGHash,
		&mapping.JMBGEncrypted,
		&mapping.PseudonymID,
		&mapping.FacilityCode,
		&createdAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pseudonym by JMBG hash: %w", err)
	}

	mapping.ID = types.ID(id)
	mapping.CreatedAt = createdAt

	return &mapping, nil
}

// GetByPseudonymID retrieves a mapping by pseudonym ID
func (r *PostgresPseudonymRepository) GetByPseudonymID(ctx context.Context, pseudonymID PseudonymID) (*PseudonymMapping, error) {
	query := `
		SELECT id, jmbg_hash, jmbg_encrypted, pseudonym_id, facility_code, created_at
		FROM privacy.pseudonym_mappings
		WHERE pseudonym_id = $1
	`

	var mapping PseudonymMapping
	var id string
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, query, string(pseudonymID)).Scan(
		&id,
		&mapping.JMBGHash,
		&mapping.JMBGEncrypted,
		&mapping.PseudonymID,
		&mapping.FacilityCode,
		&createdAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pseudonym by ID: %w", err)
	}

	mapping.ID = types.ID(id)
	mapping.CreatedAt = createdAt

	return &mapping, nil
}

// Delete removes a mapping (for GDPR right to erasure)
func (r *PostgresPseudonymRepository) Delete(ctx context.Context, pseudonymID PseudonymID) error {
	query := `DELETE FROM privacy.pseudonym_mappings WHERE pseudonym_id = $1`

	_, err := r.pool.Exec(ctx, query, string(pseudonymID))
	if err != nil {
		return fmt.Errorf("failed to delete pseudonym mapping: %w", err)
	}

	return nil
}
