package agency

import (
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// AgencyType defines the type of government agency
type AgencyType string

const (
	AgencyTypePolice         AgencyType = "POLICE"
	AgencyTypeHealthcare     AgencyType = "HEALTHCARE"
	AgencyTypeSocialServices AgencyType = "SOCIAL_SERVICES"
	AgencyTypeJudiciary      AgencyType = "JUDICIARY"
	AgencyTypeTax            AgencyType = "TAX"
	AgencyTypeLocalGov       AgencyType = "LOCAL_GOVERNMENT"
	AgencyTypeEducation      AgencyType = "EDUCATION"
	AgencyTypeEmergency      AgencyType = "EMERGENCY"
	AgencyTypeOther          AgencyType = "OTHER"
)

// AgencyStatus defines the status of an agency
type AgencyStatus string

const (
	AgencyStatusActive   AgencyStatus = "active"
	AgencyStatusInactive AgencyStatus = "inactive"
	AgencyStatusPending  AgencyStatus = "pending"
)

// Agency represents a government agency
type Agency struct {
	ID       types.ID     `json:"id"`
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Type     AgencyType   `json:"type"`
	ParentID *types.ID    `json:"parent_id,omitempty"`
	Status   AgencyStatus `json:"status"`

	Address types.Address     `json:"address"`
	Contact types.ContactInfo `json:"contact"`

	FederationCert []byte `json:"-"` // Not exposed in JSON

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkerStatus defines the status of a worker
type WorkerStatus string

const (
	WorkerStatusActive     WorkerStatus = "active"
	WorkerStatusOnLeave    WorkerStatus = "on_leave"
	WorkerStatusSuspended  WorkerStatus = "suspended"
	WorkerStatusTerminated WorkerStatus = "terminated"
)

// Worker represents an employee of an agency
type Worker struct {
	ID         types.ID     `json:"id"`
	AgencyID   types.ID     `json:"agency_id"`
	CitizenID  *types.ID    `json:"citizen_id,omitempty"`
	EmployeeID string       `json:"employee_id"`

	FirstName  string       `json:"first_name"`
	LastName   string       `json:"last_name"`
	Email      string       `json:"email"`
	Position   string       `json:"position"`
	Department string       `json:"department"`

	Roles  []WorkerRole `json:"roles"`
	Status WorkerStatus `json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FullName returns the worker's full name
func (w Worker) FullName() string {
	return w.FirstName + " " + w.LastName
}

// WorkerRole represents a role assigned to a worker
type WorkerRole struct {
	ID        types.ID  `json:"id"`
	WorkerID  types.ID  `json:"worker_id"`
	Role      string    `json:"role"`
	Scope     string    `json:"scope"`
	GrantedAt time.Time `json:"granted_at"`
	GrantedBy types.ID  `json:"granted_by"`
}

// CreateAgencyRequest is the request to create an agency
type CreateAgencyRequest struct {
	Code     string            `json:"code" validate:"required,min=2,max=50"`
	Name     string            `json:"name" validate:"required,min=2,max=255"`
	Type     AgencyType        `json:"type" validate:"required"`
	ParentID *types.ID         `json:"parent_id,omitempty"`
	Address  types.Address     `json:"address"`
	Contact  types.ContactInfo `json:"contact"`
}

// UpdateAgencyRequest is the request to update an agency
type UpdateAgencyRequest struct {
	Name     *string            `json:"name,omitempty"`
	ParentID *types.ID          `json:"parent_id,omitempty"`
	Status   *AgencyStatus      `json:"status,omitempty"`
	Address  *types.Address     `json:"address,omitempty"`
	Contact  *types.ContactInfo `json:"contact,omitempty"`
}

// CreateWorkerRequest is the request to create a worker
type CreateWorkerRequest struct {
	AgencyID   types.ID `json:"agency_id" validate:"required"`
	EmployeeID string   `json:"employee_id" validate:"required,min=1,max=100"`
	FirstName  string   `json:"first_name" validate:"required,min=1,max=100"`
	LastName   string   `json:"last_name" validate:"required,min=1,max=100"`
	Email      string   `json:"email" validate:"required,email"`
	Position   string   `json:"position"`
	Department string   `json:"department"`
	Roles      []string `json:"roles"`
}

// UpdateWorkerRequest is the request to update a worker
type UpdateWorkerRequest struct {
	FirstName  *string       `json:"first_name,omitempty"`
	LastName   *string       `json:"last_name,omitempty"`
	Email      *string       `json:"email,omitempty"`
	Position   *string       `json:"position,omitempty"`
	Department *string       `json:"department,omitempty"`
	Status     *WorkerStatus `json:"status,omitempty"`
}

// ListAgenciesFilter defines filters for listing agencies
type ListAgenciesFilter struct {
	Type     *AgencyType   `json:"type,omitempty"`
	Status   *AgencyStatus `json:"status,omitempty"`
	ParentID *types.ID     `json:"parent_id,omitempty"`
	Search   string        `json:"search,omitempty"`
	Limit    int           `json:"limit,omitempty"`
	Offset   int           `json:"offset,omitempty"`
}

// ListWorkersFilter defines filters for listing workers
type ListWorkersFilter struct {
	AgencyID *types.ID     `json:"agency_id,omitempty"`
	Status   *WorkerStatus `json:"status,omitempty"`
	Role     *string       `json:"role,omitempty"`
	Search   string        `json:"search,omitempty"`
	Limit    int           `json:"limit,omitempty"`
	Offset   int           `json:"offset,omitempty"`
}
