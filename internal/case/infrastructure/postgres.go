package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serbia-gov/platform/internal/case/domain"
	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// PostgresRepository implements domain.Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Save saves a new case
func (r *PostgresRepository) Save(ctx context.Context, c *domain.Case) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Convert access levels to JSON
	accessLevelsJSON, err := json.Marshal(c.AccessLevels)
	if err != nil {
		return errors.Wrap(err, "failed to marshal access levels")
	}

	query := `
		INSERT INTO cases.cases (
			id, case_number, type, status, priority, title, description,
			owning_agency_id, lead_worker_id,
			sla_deadline, sla_status,
			shared_with, access_levels,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`

	_, err = tx.Exec(ctx, query,
		c.ID, c.CaseNumber, c.Type, c.Status, c.Priority, c.Title, c.Description,
		c.OwningAgencyID, c.LeadWorkerID,
		c.SLADeadline, c.SLAStatus,
		c.SharedWith, accessLevelsJSON,
		c.CreatedAt, c.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return errors.Conflict("case with this number already exists")
		}
		return errors.Wrap(err, "failed to save case")
	}

	// Save participants
	for _, p := range c.Participants {
		if err := r.saveParticipant(ctx, tx, &p); err != nil {
			return err
		}
	}

	// Save assignments
	for _, a := range c.Assignments {
		if err := r.saveAssignment(ctx, tx, &a); err != nil {
			return err
		}
	}

	// Save events
	for _, e := range c.Events {
		if err := r.saveEvent(ctx, tx, &e); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// FindByID finds a case by ID
func (r *PostgresRepository) FindByID(ctx context.Context, id types.ID) (*domain.Case, error) {
	query := `
		SELECT id, case_number, type, status, priority, title, description,
			owning_agency_id, lead_worker_id,
			sla_deadline, sla_status,
			shared_with, access_levels,
			created_at, updated_at, closed_at
		FROM cases.cases
		WHERE id = $1`

	c := &domain.Case{}
	var accessLevelsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.CaseNumber, &c.Type, &c.Status, &c.Priority, &c.Title, &c.Description,
		&c.OwningAgencyID, &c.LeadWorkerID,
		&c.SLADeadline, &c.SLAStatus,
		&c.SharedWith, &accessLevelsJSON,
		&c.CreatedAt, &c.UpdatedAt, &c.ClosedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("case", id.String())
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find case")
	}

	// Parse access levels
	if err := json.Unmarshal(accessLevelsJSON, &c.AccessLevels); err != nil {
		c.AccessLevels = make(map[string]domain.AccessLevel)
	}

	// Load participants
	participants, err := r.getParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Participants = participants

	// Load assignments
	assignments, err := r.getAssignments(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Assignments = assignments

	return c, nil
}

// FindByCaseNumber finds a case by case number
func (r *PostgresRepository) FindByCaseNumber(ctx context.Context, caseNumber string) (*domain.Case, error) {
	var id types.ID
	err := r.pool.QueryRow(ctx, `SELECT id FROM cases.cases WHERE case_number = $1`, caseNumber).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, errors.NotFound("case", caseNumber)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to find case by number")
	}

	return r.FindByID(ctx, id)
}

// Update updates an existing case
func (r *PostgresRepository) Update(ctx context.Context, c *domain.Case) error {
	accessLevelsJSON, err := json.Marshal(c.AccessLevels)
	if err != nil {
		return errors.Wrap(err, "failed to marshal access levels")
	}

	query := `
		UPDATE cases.cases SET
			status = $2, priority = $3, title = $4, description = $5,
			owning_agency_id = $6, lead_worker_id = $7,
			sla_deadline = $8, sla_status = $9,
			shared_with = $10, access_levels = $11,
			updated_at = $12, closed_at = $13
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		c.ID, c.Status, c.Priority, c.Title, c.Description,
		c.OwningAgencyID, c.LeadWorkerID,
		c.SLADeadline, c.SLAStatus,
		c.SharedWith, accessLevelsJSON,
		c.UpdatedAt, c.ClosedAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to update case")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("case", c.ID.String())
	}

	return nil
}

// Delete deletes a case
func (r *PostgresRepository) Delete(ctx context.Context, id types.ID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM cases.cases WHERE id = $1`, id)
	if err != nil {
		return errors.Wrap(err, "failed to delete case")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("case", id.String())
	}

	return nil
}

// List lists cases with filters
func (r *PostgresRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Case, int, error) {
	return r.listCases(ctx, filter, "", nil)
}

// FindByAgency finds cases owned by an agency
func (r *PostgresRepository) FindByAgency(ctx context.Context, agencyID types.ID, filter domain.ListFilter) ([]domain.Case, int, error) {
	return r.listCases(ctx, filter, "owning_agency_id = $%d", []interface{}{agencyID})
}

// FindByWorker finds cases assigned to a worker
func (r *PostgresRepository) FindByWorker(ctx context.Context, workerID types.ID, filter domain.ListFilter) ([]domain.Case, int, error) {
	// Subquery to find cases with assignments to this worker
	return r.listCases(ctx, filter,
		"id IN (SELECT case_id FROM cases.assignments WHERE worker_id = $%d AND status = 'active')",
		[]interface{}{workerID})
}

// FindSharedWith finds cases shared with an agency
func (r *PostgresRepository) FindSharedWith(ctx context.Context, agencyID types.ID, filter domain.ListFilter) ([]domain.Case, int, error) {
	return r.listCases(ctx, filter, "$%d = ANY(shared_with)", []interface{}{agencyID})
}

func (r *PostgresRepository) listCases(ctx context.Context, filter domain.ListFilter, extraCondition string, extraArgs []interface{}) ([]domain.Case, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	// Add extra condition first
	if extraCondition != "" {
		for _, arg := range extraArgs {
			args = append(args, arg)
			extraCondition = fmt.Sprintf(extraCondition, argNum)
			argNum++
		}
		conditions = append(conditions, extraCondition)
	}

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

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argNum))
		args = append(args, *filter.Priority)
		argNum++
	}

	if filter.SLAStatus != nil {
		conditions = append(conditions, fmt.Sprintf("sla_status = $%d", argNum))
		args = append(args, *filter.SLAStatus)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR case_number ILIKE $%d OR description ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM cases.cases %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count cases")
	}

	// Order
	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	orderDir := "ASC"
	if filter.OrderDesc {
		orderDir = "DESC"
	}

	// Limit
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(`
		SELECT id, case_number, type, status, priority, title, description,
			owning_agency_id, lead_worker_id,
			sla_deadline, sla_status,
			shared_with, access_levels,
			created_at, updated_at, closed_at
		FROM cases.cases
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, orderDir, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list cases")
	}
	defer rows.Close()

	var cases []domain.Case
	for rows.Next() {
		var c domain.Case
		var accessLevelsJSON []byte

		err := rows.Scan(
			&c.ID, &c.CaseNumber, &c.Type, &c.Status, &c.Priority, &c.Title, &c.Description,
			&c.OwningAgencyID, &c.LeadWorkerID,
			&c.SLADeadline, &c.SLAStatus,
			&c.SharedWith, &accessLevelsJSON,
			&c.CreatedAt, &c.UpdatedAt, &c.ClosedAt,
		)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed to scan case")
		}

		if err := json.Unmarshal(accessLevelsJSON, &c.AccessLevels); err != nil {
			c.AccessLevels = make(map[string]domain.AccessLevel)
		}

		cases = append(cases, c)
	}

	return cases, total, nil
}

// --- Participant operations ---

func (r *PostgresRepository) saveParticipant(ctx context.Context, tx pgx.Tx, p *domain.Participant) error {
	query := `
		INSERT INTO cases.participants (
			id, case_id, citizen_id, role, name,
			contact_email, contact_phone, notes, added_at, added_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(ctx, query,
		p.ID, p.CaseID, p.CitizenID, p.Role, p.Name,
		p.ContactEmail, p.ContactPhone, p.Notes, p.AddedAt, p.AddedBy,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save participant")
	}

	return nil
}

func (r *PostgresRepository) AddParticipant(ctx context.Context, caseID types.ID, p *domain.Participant) error {
	p.CaseID = caseID
	query := `
		INSERT INTO cases.participants (
			id, case_id, citizen_id, role, name,
			contact_email, contact_phone, notes, added_at, added_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		p.ID, p.CaseID, p.CitizenID, p.Role, p.Name,
		p.ContactEmail, p.ContactPhone, p.Notes, p.AddedAt, p.AddedBy,
	)

	if err != nil {
		return errors.Wrap(err, "failed to add participant")
	}

	return nil
}

func (r *PostgresRepository) RemoveParticipant(ctx context.Context, caseID, participantID types.ID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM cases.participants WHERE id = $1 AND case_id = $2`,
		participantID, caseID)

	if err != nil {
		return errors.Wrap(err, "failed to remove participant")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("participant", participantID.String())
	}

	return nil
}

func (r *PostgresRepository) getParticipants(ctx context.Context, caseID types.ID) ([]domain.Participant, error) {
	query := `
		SELECT id, case_id, citizen_id, role, name,
			contact_email, contact_phone, notes, added_at, added_by
		FROM cases.participants
		WHERE case_id = $1
		ORDER BY added_at`

	rows, err := r.pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get participants")
	}
	defer rows.Close()

	var participants []domain.Participant
	for rows.Next() {
		var p domain.Participant
		err := rows.Scan(
			&p.ID, &p.CaseID, &p.CitizenID, &p.Role, &p.Name,
			&p.ContactEmail, &p.ContactPhone, &p.Notes, &p.AddedAt, &p.AddedBy,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan participant")
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// --- Assignment operations ---

func (r *PostgresRepository) saveAssignment(ctx context.Context, tx pgx.Tx, a *domain.Assignment) error {
	query := `
		INSERT INTO cases.assignments (
			id, case_id, agency_id, worker_id, role, status,
			assigned_at, assigned_by, completed_at, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(ctx, query,
		a.ID, a.CaseID, a.AgencyID, a.WorkerID, a.Role, a.Status,
		a.AssignedAt, a.AssignedBy, a.CompletedAt, a.Notes,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save assignment")
	}

	return nil
}

func (r *PostgresRepository) AddAssignment(ctx context.Context, caseID types.ID, a *domain.Assignment) error {
	a.CaseID = caseID
	query := `
		INSERT INTO cases.assignments (
			id, case_id, agency_id, worker_id, role, status,
			assigned_at, assigned_by, completed_at, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		a.ID, a.CaseID, a.AgencyID, a.WorkerID, a.Role, a.Status,
		a.AssignedAt, a.AssignedBy, a.CompletedAt, a.Notes,
	)

	if err != nil {
		return errors.Wrap(err, "failed to add assignment")
	}

	return nil
}

func (r *PostgresRepository) UpdateAssignment(ctx context.Context, a *domain.Assignment) error {
	query := `
		UPDATE cases.assignments SET
			status = $2, completed_at = $3, notes = $4
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, a.ID, a.Status, a.CompletedAt, a.Notes)
	if err != nil {
		return errors.Wrap(err, "failed to update assignment")
	}

	if result.RowsAffected() == 0 {
		return errors.NotFound("assignment", a.ID.String())
	}

	return nil
}

func (r *PostgresRepository) getAssignments(ctx context.Context, caseID types.ID) ([]domain.Assignment, error) {
	query := `
		SELECT id, case_id, agency_id, worker_id, role, status,
			assigned_at, assigned_by, completed_at, notes
		FROM cases.assignments
		WHERE case_id = $1
		ORDER BY assigned_at`

	rows, err := r.pool.Query(ctx, query, caseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get assignments")
	}
	defer rows.Close()

	var assignments []domain.Assignment
	for rows.Next() {
		var a domain.Assignment
		err := rows.Scan(
			&a.ID, &a.CaseID, &a.AgencyID, &a.WorkerID, &a.Role, &a.Status,
			&a.AssignedAt, &a.AssignedBy, &a.CompletedAt, &a.Notes,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan assignment")
		}
		assignments = append(assignments, a)
	}

	return assignments, nil
}

// --- Event operations ---

func (r *PostgresRepository) saveEvent(ctx context.Context, tx pgx.Tx, e *domain.CaseEvent) error {
	dataJSON, err := json.Marshal(e.Data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event data")
	}

	query := `
		INSERT INTO cases.case_events (
			id, case_id, type, actor_id, actor_agency_id, description, data, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = tx.Exec(ctx, query,
		e.ID, e.CaseID, e.Type, e.ActorID, e.ActorAgencyID, e.Description, dataJSON, e.Timestamp,
	)

	if err != nil {
		return errors.Wrap(err, "failed to save event")
	}

	return nil
}

func (r *PostgresRepository) AddEvent(ctx context.Context, caseID types.ID, e *domain.CaseEvent) error {
	e.CaseID = caseID
	dataJSON, err := json.Marshal(e.Data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal event data")
	}

	query := `
		INSERT INTO cases.case_events (
			id, case_id, type, actor_id, actor_agency_id, description, data, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.pool.Exec(ctx, query,
		e.ID, e.CaseID, e.Type, e.ActorID, e.ActorAgencyID, e.Description, dataJSON, e.Timestamp,
	)

	if err != nil {
		return errors.Wrap(err, "failed to add event")
	}

	return nil
}

func (r *PostgresRepository) GetEvents(ctx context.Context, caseID types.ID, limit, offset int) ([]domain.CaseEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, case_id, type, actor_id, actor_agency_id, description, data, timestamp
		FROM cases.case_events
		WHERE case_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, caseID, limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get events")
	}
	defer rows.Close()

	var events []domain.CaseEvent
	for rows.Next() {
		var e domain.CaseEvent
		var dataJSON []byte

		err := rows.Scan(
			&e.ID, &e.CaseID, &e.Type, &e.ActorID, &e.ActorAgencyID, &e.Description, &dataJSON, &e.Timestamp,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan event")
		}

		if err := json.Unmarshal(dataJSON, &e.Data); err != nil {
			e.Data = nil
		}

		events = append(events, e)
	}

	return events, nil
}
