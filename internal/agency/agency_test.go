package agency

import (
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// --- Agency Tests ---

func TestAgencyTypes(t *testing.T) {
	tests := []struct {
		agencyType AgencyType
		expected   string
	}{
		{AgencyTypePolice, "POLICE"},
		{AgencyTypeHealthcare, "HEALTHCARE"},
		{AgencyTypeSocialServices, "SOCIAL_SERVICES"},
		{AgencyTypeJudiciary, "JUDICIARY"},
		{AgencyTypeTax, "TAX"},
		{AgencyTypeLocalGov, "LOCAL_GOVERNMENT"},
		{AgencyTypeEducation, "EDUCATION"},
		{AgencyTypeEmergency, "EMERGENCY"},
		{AgencyTypeOther, "OTHER"},
	}

	for _, tt := range tests {
		t.Run(string(tt.agencyType), func(t *testing.T) {
			if string(tt.agencyType) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.agencyType)
			}
		})
	}
}

func TestAgencyStatus(t *testing.T) {
	tests := []struct {
		status   AgencyStatus
		expected string
	}{
		{AgencyStatusActive, "active"},
		{AgencyStatusInactive, "inactive"},
		{AgencyStatusPending, "pending"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.status)
			}
		})
	}
}

func TestAgencyCreation(t *testing.T) {
	parentID := types.NewID()

	agency := Agency{
		ID:       types.NewID(),
		Code:     "MUP-BG",
		Name:     "Ministry of Interior - Belgrade",
		Type:     AgencyTypePolice,
		ParentID: &parentID,
		Status:   AgencyStatusActive,
		Address: types.Address{
			Street:     "Kneza Milosa 101",
			City:       "Belgrade",
			PostalCode: "11000",
			Country:    "RS",
		},
		Contact: types.ContactInfo{
			Phone: "+381 11 123 4567",
			Email: "info@mup.gov.rs",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if agency.ID.IsZero() {
		t.Error("Agency ID should not be zero")
	}

	if agency.Code != "MUP-BG" {
		t.Errorf("Expected code 'MUP-BG', got '%s'", agency.Code)
	}

	if agency.Type != AgencyTypePolice {
		t.Errorf("Expected type POLICE, got '%s'", agency.Type)
	}

	if agency.Status != AgencyStatusActive {
		t.Errorf("Expected status active, got '%s'", agency.Status)
	}

	if agency.Address.City != "Belgrade" {
		t.Errorf("Expected city 'Belgrade', got '%s'", agency.Address.City)
	}

	if agency.Contact.Email != "info@mup.gov.rs" {
		t.Errorf("Expected email 'info@mup.gov.rs', got '%s'", agency.Contact.Email)
	}

	if agency.ParentID == nil || *agency.ParentID != parentID {
		t.Error("Parent ID should be set correctly")
	}
}

func TestAgencyWithoutParent(t *testing.T) {
	agency := Agency{
		ID:     types.NewID(),
		Code:   "ROOT",
		Name:   "Root Agency",
		Type:   AgencyTypeOther,
		Status: AgencyStatusActive,
	}

	if agency.ParentID != nil {
		t.Error("Agency without parent should have nil ParentID")
	}
}

// --- Worker Tests ---

func TestWorkerStatus(t *testing.T) {
	tests := []struct {
		status   WorkerStatus
		expected string
	}{
		{WorkerStatusActive, "active"},
		{WorkerStatusOnLeave, "on_leave"},
		{WorkerStatusSuspended, "suspended"},
		{WorkerStatusTerminated, "terminated"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, tt.status)
			}
		})
	}
}

func TestWorkerCreation(t *testing.T) {
	agencyID := types.NewID()
	citizenID := types.NewID()

	worker := Worker{
		ID:         types.NewID(),
		AgencyID:   agencyID,
		CitizenID:  &citizenID,
		EmployeeID: "EMP-001",
		FirstName:  "Marko",
		LastName:   "Petrovic",
		Email:      "marko.petrovic@mup.gov.rs",
		Position:   "Senior Investigator",
		Department: "Criminal Investigation",
		Status:     WorkerStatusActive,
		Roles: []WorkerRole{
			{
				ID:        types.NewID(),
				WorkerID:  types.NewID(),
				Role:      "case_manager",
				Scope:     "agency",
				GrantedAt: time.Now(),
				GrantedBy: types.NewID(),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if worker.ID.IsZero() {
		t.Error("Worker ID should not be zero")
	}

	if worker.AgencyID != agencyID {
		t.Error("Agency ID mismatch")
	}

	if worker.EmployeeID != "EMP-001" {
		t.Errorf("Expected employee ID 'EMP-001', got '%s'", worker.EmployeeID)
	}

	if worker.FirstName != "Marko" {
		t.Errorf("Expected first name 'Marko', got '%s'", worker.FirstName)
	}

	if worker.LastName != "Petrovic" {
		t.Errorf("Expected last name 'Petrovic', got '%s'", worker.LastName)
	}

	if worker.Status != WorkerStatusActive {
		t.Errorf("Expected status active, got '%s'", worker.Status)
	}

	if len(worker.Roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(worker.Roles))
	}
}

func TestWorkerFullName(t *testing.T) {
	tests := []struct {
		firstName string
		lastName  string
		expected  string
	}{
		{"Marko", "Petrovic", "Marko Petrovic"},
		{"Ana", "Jovanovic", "Ana Jovanovic"},
		{"", "Smith", " Smith"},
		{"John", "", "John "},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			worker := Worker{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
			}

			if worker.FullName() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, worker.FullName())
			}
		})
	}
}

func TestWorkerWithoutCitizenID(t *testing.T) {
	worker := Worker{
		ID:         types.NewID(),
		AgencyID:   types.NewID(),
		CitizenID:  nil, // External contractor, no citizen ID
		EmployeeID: "EXT-001",
		FirstName:  "External",
		LastName:   "Contractor",
		Status:     WorkerStatusActive,
	}

	if worker.CitizenID != nil {
		t.Error("Worker without citizen ID should have nil CitizenID")
	}
}

func TestWorkerRole(t *testing.T) {
	grantedBy := types.NewID()
	workerID := types.NewID()

	role := WorkerRole{
		ID:        types.NewID(),
		WorkerID:  workerID,
		Role:      "admin",
		Scope:     "agency",
		GrantedAt: time.Now(),
		GrantedBy: grantedBy,
	}

	if role.ID.IsZero() {
		t.Error("Role ID should not be zero")
	}

	if role.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role.Role)
	}

	if role.Scope != "agency" {
		t.Errorf("Expected scope 'agency', got '%s'", role.Scope)
	}

	if role.WorkerID != workerID {
		t.Error("Worker ID mismatch")
	}

	if role.GrantedBy != grantedBy {
		t.Error("GrantedBy mismatch")
	}
}

// --- Request Validation Tests ---

func TestCreateAgencyRequest(t *testing.T) {
	parentID := types.NewID()

	req := CreateAgencyRequest{
		Code:     "CSW-BG",
		Name:     "Center for Social Work Belgrade",
		Type:     AgencyTypeSocialServices,
		ParentID: &parentID,
		Address: types.Address{
			Street: "Ruzveltova 61",
			City:   "Belgrade",
		},
		Contact: types.ContactInfo{
			Phone: "+381 11 765 4321",
			Email: "info@csw-bg.gov.rs",
		},
	}

	if req.Code == "" {
		t.Error("Code should not be empty")
	}

	if req.Name == "" {
		t.Error("Name should not be empty")
	}

	if req.Type != AgencyTypeSocialServices {
		t.Errorf("Expected type SOCIAL_SERVICES, got '%s'", req.Type)
	}
}

func TestCreateWorkerRequest(t *testing.T) {
	agencyID := types.NewID()

	req := CreateWorkerRequest{
		AgencyID:   agencyID,
		EmployeeID: "EMP-002",
		FirstName:  "Jovana",
		LastName:   "Nikolic",
		Email:      "jovana.nikolic@csw.gov.rs",
		Position:   "Social Worker",
		Department: "Child Protection",
		Roles:      []string{"case_worker", "report_viewer"},
	}

	if req.AgencyID.IsZero() {
		t.Error("Agency ID should not be zero")
	}

	if req.EmployeeID == "" {
		t.Error("Employee ID should not be empty")
	}

	if req.FirstName == "" {
		t.Error("First name should not be empty")
	}

	if req.LastName == "" {
		t.Error("Last name should not be empty")
	}

	if req.Email == "" {
		t.Error("Email should not be empty")
	}

	if len(req.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(req.Roles))
	}
}

func TestUpdateAgencyRequest(t *testing.T) {
	newName := "Updated Agency Name"
	newStatus := AgencyStatusInactive

	req := UpdateAgencyRequest{
		Name:   &newName,
		Status: &newStatus,
	}

	if req.Name == nil || *req.Name != newName {
		t.Error("Name should be set correctly")
	}

	if req.Status == nil || *req.Status != newStatus {
		t.Error("Status should be set correctly")
	}
}

func TestUpdateWorkerRequest(t *testing.T) {
	newFirstName := "Updated"
	newStatus := WorkerStatusOnLeave

	req := UpdateWorkerRequest{
		FirstName: &newFirstName,
		Status:    &newStatus,
	}

	if req.FirstName == nil || *req.FirstName != newFirstName {
		t.Error("First name should be set correctly")
	}

	if req.Status == nil || *req.Status != newStatus {
		t.Error("Status should be set correctly")
	}
}

// --- Filter Tests ---

func TestListAgenciesFilter(t *testing.T) {
	agencyType := AgencyTypePolice
	status := AgencyStatusActive
	parentID := types.NewID()

	filter := ListAgenciesFilter{
		Type:     &agencyType,
		Status:   &status,
		ParentID: &parentID,
		Search:   "Ministry",
		Limit:    10,
		Offset:   0,
	}

	if filter.Type == nil || *filter.Type != AgencyTypePolice {
		t.Error("Type filter should be set correctly")
	}

	if filter.Status == nil || *filter.Status != AgencyStatusActive {
		t.Error("Status filter should be set correctly")
	}

	if filter.Search != "Ministry" {
		t.Errorf("Expected search 'Ministry', got '%s'", filter.Search)
	}

	if filter.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", filter.Limit)
	}
}

func TestListWorkersFilter(t *testing.T) {
	agencyID := types.NewID()
	status := WorkerStatusActive
	role := "admin"

	filter := ListWorkersFilter{
		AgencyID: &agencyID,
		Status:   &status,
		Role:     &role,
		Search:   "John",
		Limit:    20,
		Offset:   5,
	}

	if filter.AgencyID == nil || *filter.AgencyID != agencyID {
		t.Error("Agency ID filter should be set correctly")
	}

	if filter.Status == nil || *filter.Status != WorkerStatusActive {
		t.Error("Status filter should be set correctly")
	}

	if filter.Role == nil || *filter.Role != "admin" {
		t.Error("Role filter should be set correctly")
	}

	if filter.Offset != 5 {
		t.Errorf("Expected offset 5, got %d", filter.Offset)
	}
}
