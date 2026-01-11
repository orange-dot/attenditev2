package agency

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Repository provides database operations for agencies and workers
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new agency repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Agency Operations ---

// CreateAgency creates a new agency
func (r *Repository) CreateAgency(ctx context.Context, agency *Agency) error {
	query := `
		INSERT INTO identity.agencies (
			id, code, name, type, parent_id, status,
			address_street, address_city, address_postal_code, address_country,
			address_lat, address_lng,
			contact_email, contact_phone, contact_mobile
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15
		)`

	_, err := r.pool.Exec(ctx, query,
		agency.ID, agency.Code, agency.Name, agency.Type, agency.ParentID, agency.Status,
		agency.Address.Street, agency.Address.City, agency.Address.PostalCode, agency.Address.Country,
		agency.Address.Lat, agency.Address.Lng,
		agency.Contact.Email, agency.Contact.Phone, agency.Contact.Mobile,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return errors.Conflict("agency with this code already exists")
		}
		return errors.Wrap(err, "failed to create agency")
	}

	return nil
}

// GetAgency retrieves an agency by ID
func (r *Repository) GetAgency(ctx context.Context, id types.ID) (*Agency, error) {
	query := `
		SELECT id, code, name, type, parent_id, status,
			address_street, address_city, address_postal_code, address_country,
			address_lat, address_lng,
			contact_email, contact_phone, contact_mobile,
			created_at, updated_at
		FROM identity.agencies
		WHERE id = $1`

	agency := &Agency{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&agency.ID, &agency.Code, &agency.Name, &agency.Type, &agency.ParentID, &agency.Status,
		&agency.Address.Street, &agency.Address.City, &agency.Address.PostalCode, &agency.Address.Country,
		&agency.Address.Lat, &agency.Address.Lng,
		&agency.Contact.Email, &agency.Contact.Phone, &agency.Contact.Mobile,
		&agency.CreatedAt, &agency.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("agency", id.String())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agency")
	}

	return agency, nil
}

// GetAgencyByCode retrieves an agency by code
func (r *Repository) GetAgencyByCode(ctx context.Context, code string) (*Agency, error) {
	query := `
		SELECT id, code, name, type, parent_id, status,
			address_street, address_city, address_postal_code, address_country,
			address_lat, address_lng,
			contact_email, contact_phone, contact_mobile,
			created_at, updated_at
		FROM identity.agencies
		WHERE code = $1`

	agency := &Agency{}
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&agency.ID, &agency.Code, &agency.Name, &agency.Type, &agency.ParentID, &agency.Status,
		&agency.Address.Street, &agency.Address.City, &agency.Address.PostalCode, &agency.Address.Country,
		&agency.Address.Lat, &agency.Address.Lng,
		&agency.Contact.Email, &agency.Contact.Phone, &agency.Contact.Mobile,
		&agency.CreatedAt, &agency.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("agency", code)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get agency by code")
	}

	return agency, nil
}

// UpdateAgency updates an agency
func (r *Repository) UpdateAgency(ctx context.Context, agency *Agency) error {
	query := `
		UPDATE identity.agencies SET
			name = $2, parent_id = $3, status = $4,
			address_street = $5, address_city = $6, address_postal_code = $7,
			address_country = $8, address_lat = $9, address_lng = $10,
			contact_email = $11, contact_phone = $12, contact_mobile = $13
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		agency.ID, agency.Name, agency.ParentID, agency.Status,
		agency.Address.Street, agency.Address.City, agency.Address.PostalCode,
		agency.Address.Country, agency.Address.Lat, agency.Address.Lng,
		agency.Contact.Email, agency.Contact.Phone, agency.Contact.Mobile,
	)

	if err != nil {
		return errors.Wrap(err, "failed to update agency")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("agency", agency.ID.String())
	}

	return nil
}

// DeleteAgency deletes an agency
func (r *Repository) DeleteAgency(ctx context.Context, id types.ID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM identity.agencies WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete agency")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("agency", id.String())
	}

	return nil
}

// ListAgencies lists agencies with optional filters
func (r *Repository) ListAgencies(ctx context.Context, filter ListAgenciesFilter) ([]Agency, int, error) {
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

	if filter.ParentID != nil {
		conditions = append(conditions, fmt.Sprintf("parent_id = $%d", argNum))
		args = append(args, *filter.ParentID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM identity.agencies %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count agencies")
	}

	// Get agencies
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(`
		SELECT id, code, name, type, parent_id, status,
			address_street, address_city, address_postal_code, address_country,
			address_lat, address_lng,
			contact_email, contact_phone, contact_mobile,
			created_at, updated_at
		FROM identity.agencies
		%s
		ORDER BY name
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list agencies")
	}
	defer rows.Close()

	var agencies []Agency
	for rows.Next() {
		var agency Agency
		err := rows.Scan(
			&agency.ID, &agency.Code, &agency.Name, &agency.Type, &agency.ParentID, &agency.Status,
			&agency.Address.Street, &agency.Address.City, &agency.Address.PostalCode, &agency.Address.Country,
			&agency.Address.Lat, &agency.Address.Lng,
			&agency.Contact.Email, &agency.Contact.Phone, &agency.Contact.Mobile,
			&agency.CreatedAt, &agency.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan agency")
		}
		agencies = append(agencies, agency)
	}

	return agencies, total, nil
}

// --- Worker Operations ---

// CreateWorker creates a new worker
func (r *Repository) CreateWorker(ctx context.Context, worker *Worker) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO identity.workers (
			id, agency_id, citizen_id, employee_id,
			first_name, last_name, email, position, department, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = tx.Exec(ctx, query,
		worker.ID, worker.AgencyID, worker.CitizenID, worker.EmployeeID,
		worker.FirstName, worker.LastName, worker.Email, worker.Position, worker.Department, worker.Status,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return errors.Conflict("worker with this email or employee_id already exists in agency")
		}
		return errors.Wrap(err, "failed to create worker")
	}

	// Add roles
	for _, role := range worker.Roles {
		roleQuery := `
			INSERT INTO identity.worker_roles (id, worker_id, role, scope, granted_by)
			VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(ctx, roleQuery, role.ID, worker.ID, role.Role, role.Scope, role.GrantedBy)
		if err != nil {
			return errors.Wrap(err, "failed to add worker role")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// GetWorker retrieves a worker by ID
func (r *Repository) GetWorker(ctx context.Context, id types.ID) (*Worker, error) {
	query := `
		SELECT id, agency_id, citizen_id, employee_id,
			first_name, last_name, email, position, department, status,
			created_at, updated_at
		FROM identity.workers
		WHERE id = $1`

	worker := &Worker{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&worker.ID, &worker.AgencyID, &worker.CitizenID, &worker.EmployeeID,
		&worker.FirstName, &worker.LastName, &worker.Email, &worker.Position, &worker.Department, &worker.Status,
		&worker.CreatedAt, &worker.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("worker", id.String())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get worker")
	}

	// Get roles
	roles, err := r.getWorkerRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	worker.Roles = roles

	return worker, nil
}

// getWorkerRoles retrieves roles for a worker
func (r *Repository) getWorkerRoles(ctx context.Context, workerID types.ID) ([]WorkerRole, error) {
	query := `
		SELECT id, worker_id, role, scope, granted_at, granted_by
		FROM identity.worker_roles
		WHERE worker_id = $1`

	rows, err := r.pool.Query(ctx, query, workerID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get worker roles")
	}
	defer rows.Close()

	var roles []WorkerRole
	for rows.Next() {
		var role WorkerRole
		if err := rows.Scan(&role.ID, &role.WorkerID, &role.Role, &role.Scope, &role.GrantedAt, &role.GrantedBy); err != nil {
			return nil, errors.Wrap(err, "failed to scan worker role")
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// UpdateWorker updates a worker
func (r *Repository) UpdateWorker(ctx context.Context, worker *Worker) error {
	query := `
		UPDATE identity.workers SET
			first_name = $2, last_name = $3, email = $4,
			position = $5, department = $6, status = $7
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		worker.ID, worker.FirstName, worker.LastName, worker.Email,
		worker.Position, worker.Department, worker.Status,
	)

	if err != nil {
		return errors.Wrap(err, "failed to update worker")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("worker", worker.ID.String())
	}

	return nil
}

// DeleteWorker deletes a worker
func (r *Repository) DeleteWorker(ctx context.Context, id types.ID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM identity.workers WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete worker")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("worker", id.String())
	}

	return nil
}

// ListWorkers lists workers with optional filters
func (r *Repository) ListWorkers(ctx context.Context, filter ListWorkersFilter) ([]Worker, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.AgencyID != nil {
		conditions = append(conditions, fmt.Sprintf("w.agency_id = $%d", argNum))
		args = append(args, *filter.AgencyID)
		argNum++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("w.status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(w.first_name ILIKE $%d OR w.last_name ILIKE $%d OR w.email ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM identity.workers w %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count workers")
	}

	// Get workers
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(`
		SELECT w.id, w.agency_id, w.citizen_id, w.employee_id,
			w.first_name, w.last_name, w.email, w.position, w.department, w.status,
			w.created_at, w.updated_at
		FROM identity.workers w
		%s
		ORDER BY w.last_name, w.first_name
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list workers")
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		var worker Worker
		err := rows.Scan(
			&worker.ID, &worker.AgencyID, &worker.CitizenID, &worker.EmployeeID,
			&worker.FirstName, &worker.LastName, &worker.Email, &worker.Position, &worker.Department, &worker.Status,
			&worker.CreatedAt, &worker.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan worker")
		}
		workers = append(workers, worker)
	}

	return workers, total, nil
}
