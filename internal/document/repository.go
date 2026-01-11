package document

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Repository provides database operations for documents
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new document repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Save saves a new document
func (r *Repository) Save(ctx context.Context, d *Document) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO documents.documents (
			id, document_number, type, status, title, description,
			owner_agency_id, created_by, case_id,
			current_version, shared_with,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = tx.Exec(ctx, query,
		d.ID, d.DocumentNumber, d.Type, d.Status, d.Title, d.Description,
		d.OwnerAgencyID, d.CreatedBy, d.CaseID,
		d.CurrentVersion, d.SharedWith,
		d.CreatedAt, d.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return errors.Conflict("document with this number already exists")
		}
		return errors.Wrap(err, "failed to save document")
	}

	// Save versions
	for _, v := range d.Versions {
		if err := r.saveVersion(ctx, tx, &v); err != nil {
			return err
		}
	}

	// Save signatures
	for _, s := range d.Signatures {
		if err := r.saveSignature(ctx, tx, &s); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// FindByID finds a document by ID
func (r *Repository) FindByID(ctx context.Context, id types.ID) (*Document, error) {
	query := `
		SELECT id, document_number, type, status, title, description,
			owner_agency_id, created_by, case_id,
			current_version, shared_with,
			created_at, updated_at
		FROM documents.documents
		WHERE id = $1`

	d := &Document{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.DocumentNumber, &d.Type, &d.Status, &d.Title, &d.Description,
		&d.OwnerAgencyID, &d.CreatedBy, &d.CaseID,
		&d.CurrentVersion, &d.SharedWith,
		&d.CreatedAt, &d.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("document", id.String())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find document")
	}

	// Load versions
	versions, err := r.getVersions(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Versions = versions

	// Load signatures
	signatures, err := r.getSignatures(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Signatures = signatures

	return d, nil
}

// Update updates a document
func (r *Repository) Update(ctx context.Context, d *Document) error {
	query := `
		UPDATE documents.documents SET
			status = $2, title = $3, description = $4,
			current_version = $5, shared_with = $6, updated_at = $7
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		d.ID, d.Status, d.Title, d.Description,
		d.CurrentVersion, d.SharedWith, d.UpdatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to update document")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("document", d.ID.String())
	}

	return nil
}

// Delete deletes a document
func (r *Repository) Delete(ctx context.Context, id types.ID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM documents.documents WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete document")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("document", id.String())
	}

	return nil
}

// List lists documents with filters
func (r *Repository) List(ctx context.Context, filter ListDocumentsFilter) ([]Document, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argNum))
		args = append(args, *filter.Type)
		argNum++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.CaseID != nil {
		conditions = append(conditions, fmt.Sprintf("case_id = $%d", argNum))
		args = append(args, *filter.CaseID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR document_number ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM documents.documents %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count documents")
	}

	// Limit
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(`
		SELECT id, document_number, type, status, title, description,
			owner_agency_id, created_by, case_id,
			current_version, shared_with,
			created_at, updated_at
		FROM documents.documents
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list documents")
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		err := rows.Scan(
			&d.ID, &d.DocumentNumber, &d.Type, &d.Status, &d.Title, &d.Description,
			&d.OwnerAgencyID, &d.CreatedBy, &d.CaseID,
			&d.CurrentVersion, &d.SharedWith,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan document")
		}
		docs = append(docs, d)
	}

	return docs, total, nil
}

// FindByCase finds documents for a case
func (r *Repository) FindByCase(ctx context.Context, caseID types.ID, filter ListDocumentsFilter) ([]Document, int, error) {
	filter.CaseID = &caseID
	return r.List(ctx, filter)
}

// --- Version operations ---

func (r *Repository) saveVersion(ctx context.Context, tx pgx.Tx, v *DocumentVersion) error {
	query := `
		INSERT INTO documents.versions (
			id, document_id, version, file_path, file_hash,
			file_size, mime_type, created_at, created_by, change_summary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(ctx, query,
		v.ID, v.DocumentID, v.Version, v.FilePath, v.FileHash,
		v.FileSize, v.MimeType, v.CreatedAt, v.CreatedBy, v.ChangeSummary,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save version")
	}

	return nil
}

// AddVersion adds a version to a document
func (r *Repository) AddVersion(ctx context.Context, v *DocumentVersion) error {
	query := `
		INSERT INTO documents.versions (
			id, document_id, version, file_path, file_hash,
			file_size, mime_type, created_at, created_by, change_summary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		v.ID, v.DocumentID, v.Version, v.FilePath, v.FileHash,
		v.FileSize, v.MimeType, v.CreatedAt, v.CreatedBy, v.ChangeSummary,
	)

	if err != nil {
		return errors.Wrap(err, "failed to add version")
	}

	return nil
}

func (r *Repository) getVersions(ctx context.Context, documentID types.ID) ([]DocumentVersion, error) {
	query := `
		SELECT id, document_id, version, file_path, file_hash,
			file_size, mime_type, created_at, created_by, change_summary
		FROM documents.versions
		WHERE document_id = $1
		ORDER BY version DESC`

	rows, err := r.pool.Query(ctx, query, documentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get versions")
	}
	defer rows.Close()

	var versions []DocumentVersion
	for rows.Next() {
		var v DocumentVersion
		err := rows.Scan(
			&v.ID, &v.DocumentID, &v.Version, &v.FilePath, &v.FileHash,
			&v.FileSize, &v.MimeType, &v.CreatedAt, &v.CreatedBy, &v.ChangeSummary,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan version")
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// --- Signature operations ---

func (r *Repository) saveSignature(ctx context.Context, tx pgx.Tx, s *Signature) error {
	query := `
		INSERT INTO documents.signatures (
			id, document_id, version, signer_id, signer_agency_id,
			type, status, signature_data, certificate, timestamp_token,
			reason, location, signed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := tx.Exec(ctx, query,
		s.ID, s.DocumentID, s.Version, s.SignerID, s.SignerAgencyID,
		s.Type, s.Status, s.SignatureData, s.Certificate, s.TimestampToken,
		s.Reason, s.Location, s.SignedAt, s.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save signature")
	}

	return nil
}

// AddSignature adds a signature to a document
func (r *Repository) AddSignature(ctx context.Context, s *Signature) error {
	query := `
		INSERT INTO documents.signatures (
			id, document_id, version, signer_id, signer_agency_id,
			type, status, reason, location, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		s.ID, s.DocumentID, s.Version, s.SignerID, s.SignerAgencyID,
		s.Type, s.Status, s.Reason, s.Location, s.CreatedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to add signature")
	}

	return nil
}

// UpdateSignature updates a signature
func (r *Repository) UpdateSignature(ctx context.Context, s *Signature) error {
	query := `
		UPDATE documents.signatures SET
			status = $2, signature_data = $3, certificate = $4,
			timestamp_token = $5, signed_at = $6
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		s.ID, s.Status, s.SignatureData, s.Certificate,
		s.TimestampToken, s.SignedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to update signature")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("signature", s.ID.String())
	}

	return nil
}

func (r *Repository) getSignatures(ctx context.Context, documentID types.ID) ([]Signature, error) {
	query := `
		SELECT id, document_id, version, signer_id, signer_agency_id,
			type, status, signature_data, certificate, timestamp_token,
			reason, location, signed_at, created_at
		FROM documents.signatures
		WHERE document_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, documentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signatures")
	}
	defer rows.Close()

	var signatures []Signature
	for rows.Next() {
		var s Signature
		err := rows.Scan(
			&s.ID, &s.DocumentID, &s.Version, &s.SignerID, &s.SignerAgencyID,
			&s.Type, &s.Status, &s.SignatureData, &s.Certificate, &s.TimestampToken,
			&s.Reason, &s.Location, &s.SignedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan signature")
		}
		signatures = append(signatures, s)
	}

	return signatures, nil
}
