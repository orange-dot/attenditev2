package domain

import (
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// TestNewCase tests creating a new case
func TestNewCase(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	c, err := NewCase(
		CaseTypeChildWelfare,
		PriorityHigh,
		"Test Case",
		"Test description",
		agencyID,
		workerID,
	)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if c.ID.IsZero() {
		t.Error("Expected non-zero ID")
	}

	if c.Status != CaseStatusDraft {
		t.Errorf("Expected status %s, got %s", CaseStatusDraft, c.Status)
	}

	if c.Type != CaseTypeChildWelfare {
		t.Errorf("Expected type %s, got %s", CaseTypeChildWelfare, c.Type)
	}

	if c.Priority != PriorityHigh {
		t.Errorf("Expected priority %s, got %s", PriorityHigh, c.Priority)
	}

	if c.SLADeadline == nil {
		t.Error("Expected SLA deadline to be set")
	}

	// Should have creation event
	if len(c.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(c.Events))
	}

	if c.Events[0].Type != CaseEventTypeCreated {
		t.Errorf("Expected event type %s, got %s", CaseEventTypeCreated, c.Events[0].Type)
	}
}

// TestNewCaseValidation tests validation when creating a case
func TestNewCaseValidation(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	tests := []struct {
		name        string
		title       string
		agencyID    types.ID
		workerID    types.ID
		expectError bool
	}{
		{"Empty title", "", agencyID, workerID, true},
		{"Zero agency ID", "Test", types.ID(""), workerID, true},
		{"Zero worker ID", "Test", agencyID, types.ID(""), true},
		{"Valid case", "Test", agencyID, workerID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCase(
				CaseTypeChildWelfare,
				PriorityMedium,
				tt.title,
				"Description",
				tt.agencyID,
				tt.workerID,
			)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCaseStateTransitions tests the state machine transitions
func TestCaseStateTransitions(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	c, _ := NewCase(
		CaseTypeAdministrative,
		PriorityMedium,
		"State Test Case",
		"Testing state transitions",
		agencyID,
		workerID,
	)

	// Draft -> Open
	if err := c.Open(workerID, agencyID); err != nil {
		t.Fatalf("Failed to open case: %v", err)
	}
	if c.Status != CaseStatusOpen {
		t.Errorf("Expected status %s, got %s", CaseStatusOpen, c.Status)
	}

	// Open -> InProgress
	if err := c.StartProgress(workerID, agencyID); err != nil {
		t.Fatalf("Failed to start progress: %v", err)
	}
	if c.Status != CaseStatusInProgress {
		t.Errorf("Expected status %s, got %s", CaseStatusInProgress, c.Status)
	}

	// InProgress -> Closed
	if err := c.Close(workerID, agencyID, "Case resolved"); err != nil {
		t.Fatalf("Failed to close case: %v", err)
	}
	if c.Status != CaseStatusClosed {
		t.Errorf("Expected status %s, got %s", CaseStatusClosed, c.Status)
	}

	if c.ClosedAt == nil {
		t.Error("Expected ClosedAt to be set")
	}
}

// TestInvalidStateTransitions tests that invalid transitions are rejected
func TestInvalidStateTransitions(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	t.Run("Cannot open non-draft case", func(t *testing.T) {
		c, _ := NewCase(CaseTypeAdministrative, PriorityMedium, "Test", "Desc", agencyID, workerID)
		c.Open(workerID, agencyID)

		// Try to open again
		err := c.Open(workerID, agencyID)
		if err == nil {
			t.Error("Expected error when opening non-draft case")
		}
	})

	t.Run("Cannot start progress on non-open case", func(t *testing.T) {
		c, _ := NewCase(CaseTypeAdministrative, PriorityMedium, "Test", "Desc", agencyID, workerID)

		// Try to start progress without opening first
		err := c.StartProgress(workerID, agencyID)
		if err == nil {
			t.Error("Expected error when starting progress on draft case")
		}
	})

	t.Run("Cannot close already closed case", func(t *testing.T) {
		c, _ := NewCase(CaseTypeAdministrative, PriorityMedium, "Test", "Desc", agencyID, workerID)
		c.Open(workerID, agencyID)
		c.StartProgress(workerID, agencyID)
		c.Close(workerID, agencyID, "Resolved")

		// Try to close again
		err := c.Close(workerID, agencyID, "Trying to close again")
		if err == nil {
			t.Error("Expected error when closing already closed case")
		}
	})
}

// TestCaseEscalation tests escalating a case
func TestCaseEscalation(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	supervisorID := types.NewID()

	c, _ := NewCase(CaseTypeChildWelfare, PriorityUrgent, "Urgent Case", "Needs escalation", agencyID, workerID)
	c.Open(workerID, agencyID)

	err := c.Escalate(1, "Needs supervisor attention", supervisorID, workerID, agencyID)
	if err != nil {
		t.Fatalf("Failed to escalate case: %v", err)
	}

	if c.Status != CaseStatusEscalated {
		t.Errorf("Expected status %s, got %s", CaseStatusEscalated, c.Status)
	}

	// Check event was recorded
	foundEscalationEvent := false
	for _, e := range c.Events {
		if e.Type == CaseEventTypeEscalated {
			foundEscalationEvent = true
			if e.Data["level"].(int) != 1 {
				t.Error("Escalation level not recorded correctly")
			}
			break
		}
	}

	if !foundEscalationEvent {
		t.Error("Escalation event not found")
	}
}

// TestCaseSharing tests sharing a case with another agency
func TestCaseSharing(t *testing.T) {
	ownerAgencyID := types.NewID()
	sharedAgencyID := types.NewID()
	workerID := types.NewID()

	c, _ := NewCase(CaseTypeSocialAssistance, PriorityMedium, "Shared Case", "Multi-agency case", ownerAgencyID, workerID)

	// Share with read access
	err := c.Share(sharedAgencyID, AccessLevelRead, workerID, ownerAgencyID)
	if err != nil {
		t.Fatalf("Failed to share case: %v", err)
	}

	if len(c.SharedWith) != 1 {
		t.Errorf("Expected 1 shared agency, got %d", len(c.SharedWith))
	}

	if c.AccessLevels[sharedAgencyID.String()] != AccessLevelRead {
		t.Error("Access level not set correctly")
	}

	// Cannot share with owner
	err = c.Share(ownerAgencyID, AccessLevelRead, workerID, ownerAgencyID)
	if err == nil {
		t.Error("Expected error when sharing with owner agency")
	}

	// Remove sharing by setting level to None
	err = c.Share(sharedAgencyID, AccessLevelNone, workerID, ownerAgencyID)
	if err != nil {
		t.Fatalf("Failed to remove sharing: %v", err)
	}

	if len(c.SharedWith) != 0 {
		t.Errorf("Expected 0 shared agencies after removal, got %d", len(c.SharedWith))
	}
}

// TestCaseTransfer tests transferring case ownership
func TestCaseTransfer(t *testing.T) {
	ownerAgencyID := types.NewID()
	newAgencyID := types.NewID()
	workerID := types.NewID()
	newWorkerID := types.NewID()

	c, _ := NewCase(CaseTypeHealthcare, PriorityHigh, "Transfer Case", "Being transferred", ownerAgencyID, workerID)
	c.Open(workerID, ownerAgencyID)

	err := c.Transfer(newAgencyID, newWorkerID, workerID, ownerAgencyID, "Patient moved to new district")
	if err != nil {
		t.Fatalf("Failed to transfer case: %v", err)
	}

	if c.OwningAgencyID != newAgencyID {
		t.Error("Ownership not transferred")
	}

	if c.LeadWorkerID != newWorkerID {
		t.Error("Lead worker not updated")
	}

	// Previous owner should have read access
	if c.AccessLevels[ownerAgencyID.String()] != AccessLevelRead {
		t.Error("Previous owner should have read access")
	}

	// Status should be reset to open
	if c.Status != CaseStatusOpen {
		t.Errorf("Expected status %s after transfer, got %s", CaseStatusOpen, c.Status)
	}

	// Cannot transfer to same agency
	err = c.Transfer(newAgencyID, newWorkerID, newWorkerID, newAgencyID, "Invalid transfer")
	if err == nil {
		t.Error("Expected error when transferring to same agency")
	}
}

// TestCaseAccessControl tests access control checks
func TestCaseAccessControl(t *testing.T) {
	ownerAgencyID := types.NewID()
	sharedAgencyID := types.NewID()
	unrelatedAgencyID := types.NewID()
	workerID := types.NewID()

	c, _ := NewCase(CaseTypeCivil, PriorityLow, "Access Test", "Testing access control", ownerAgencyID, workerID)
	c.Share(sharedAgencyID, AccessLevelContribute, workerID, ownerAgencyID)

	// Owner has full access
	if !c.CanAccess(ownerAgencyID, AccessLevelFull) {
		t.Error("Owner should have full access")
	}

	// Shared agency has up to contribute access
	if !c.CanAccess(sharedAgencyID, AccessLevelRead) {
		t.Error("Shared agency should have read access")
	}
	if !c.CanAccess(sharedAgencyID, AccessLevelContribute) {
		t.Error("Shared agency should have contribute access")
	}
	if c.CanAccess(sharedAgencyID, AccessLevelFull) {
		t.Error("Shared agency should not have full access")
	}

	// Unrelated agency has no access
	if c.CanAccess(unrelatedAgencyID, AccessLevelRead) {
		t.Error("Unrelated agency should not have access")
	}
}

// TestAddParticipant tests adding participants to a case
func TestAddParticipant(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	c, _ := NewCase(CaseTypeChildWelfare, PriorityHigh, "Participant Test", "Testing participants", agencyID, workerID)

	participant := Participant{
		Role:         ParticipantRoleSubject,
		Name:         "Jane Doe",
		ContactEmail: "jane@example.com",
		Notes:        "Child subject",
	}

	err := c.AddParticipant(participant, workerID, agencyID)
	if err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	if len(c.Participants) != 1 {
		t.Errorf("Expected 1 participant, got %d", len(c.Participants))
	}

	added := c.Participants[0]
	if added.Name != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%s'", added.Name)
	}

	if added.ID.IsZero() {
		t.Error("Participant should have an ID assigned")
	}

	if added.AddedBy != workerID {
		t.Error("AddedBy should be set to actor")
	}

	// Test validation - empty name
	emptyParticipant := Participant{Role: ParticipantRoleWitness}
	err = c.AddParticipant(emptyParticipant, workerID, agencyID)
	if err == nil {
		t.Error("Expected error when adding participant with empty name")
	}
}

// TestAssignWorker tests assigning workers to a case
func TestAssignWorker(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	supportWorkerID := types.NewID()

	c, _ := NewCase(CaseTypeCriminal, PriorityUrgent, "Assignment Test", "Testing assignments", agencyID, workerID)

	err := c.Assign(supportWorkerID, agencyID, AssignmentRoleSupport, workerID, agencyID)
	if err != nil {
		t.Fatalf("Failed to assign worker: %v", err)
	}

	if len(c.Assignments) != 1 {
		t.Errorf("Expected 1 assignment, got %d", len(c.Assignments))
	}

	assignment := c.Assignments[0]
	if assignment.WorkerID != supportWorkerID {
		t.Error("Worker ID not set correctly")
	}
	if assignment.Role != AssignmentRoleSupport {
		t.Error("Assignment role not set correctly")
	}
	if assignment.Status != AssignmentStatusActive {
		t.Error("Assignment should be active")
	}

	// Cannot assign same worker twice
	err = c.Assign(supportWorkerID, agencyID, AssignmentRoleLead, workerID, agencyID)
	if err == nil {
		t.Error("Expected error when assigning same worker twice")
	}
}

// TestCloseWithActiveAssignments tests that case cannot be closed with active assignments
func TestCloseWithActiveAssignments(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	supportWorkerID := types.NewID()

	c, _ := NewCase(CaseTypeAdministrative, PriorityMedium, "Close Test", "Testing close validation", agencyID, workerID)
	c.Open(workerID, agencyID)
	c.StartProgress(workerID, agencyID)
	c.Assign(supportWorkerID, agencyID, AssignmentRoleSupport, workerID, agencyID)

	// Should fail because there's an active assignment
	err := c.Close(workerID, agencyID, "Trying to close")
	if err == nil {
		t.Error("Expected error when closing case with active assignments")
	}

	// Complete the assignment
	c.Assignments[0].Complete()

	// Now should succeed
	err = c.Close(workerID, agencyID, "Case resolved")
	if err != nil {
		t.Fatalf("Failed to close case after completing assignments: %v", err)
	}
}

// TestSLACalculation tests that SLA deadlines are calculated correctly
func TestSLACalculation(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	tests := []struct {
		caseType         CaseType
		priority         Priority
		expectedMinHours float64
		expectedMaxHours float64
	}{
		{CaseTypeChildWelfare, PriorityEmergency, 5, 7},     // 24 * 0.25 = 6
		{CaseTypeChildWelfare, PriorityUrgent, 11, 13},     // 24 * 0.5 = 12
		{CaseTypeChildWelfare, PriorityHigh, 17, 19},       // 24 * 0.75 = 18
		{CaseTypeTax, PriorityLow, 350, 370},               // 240 * 1.5 = 360
	}

	for _, tt := range tests {
		t.Run(string(tt.caseType)+"-"+string(tt.priority), func(t *testing.T) {
			c, _ := NewCase(tt.caseType, tt.priority, "SLA Test", "Testing SLA", agencyID, workerID)

			if c.SLADeadline == nil {
				t.Fatal("SLA deadline should be set")
			}

			hoursUntilDeadline := c.SLADeadline.Sub(c.CreatedAt).Hours()

			if hoursUntilDeadline < tt.expectedMinHours || hoursUntilDeadline > tt.expectedMaxHours {
				t.Errorf("Expected SLA between %.0f-%.0f hours, got %.0f hours",
					tt.expectedMinHours, tt.expectedMaxHours, hoursUntilDeadline)
			}
		})
	}
}

// TestDomainEvents tests that domain events are generated
func TestDomainEvents(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	c, _ := NewCase(CaseTypeHealthcare, PriorityMedium, "Event Test", "Testing events", agencyID, workerID)

	// Get domain events (should include creation event)
	events := c.GetDomainEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 domain event, got %d", len(events))
	}

	// Events should be cleared after getting
	events = c.GetDomainEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 domain events after clear, got %d", len(events))
	}

	// Open case and check new event
	c.Open(workerID, agencyID)
	events = c.GetDomainEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 domain event after open, got %d", len(events))
	}

	if events[0].Type != string(CaseEventTypeStatusChanged) {
		t.Errorf("Expected event type %s, got %s", CaseEventTypeStatusChanged, events[0].Type)
	}
}

// TestCaseNumberGeneration tests that case numbers are generated correctly
func TestCaseNumberGeneration(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	currentYear := time.Now().Year()

	cases := []struct {
		caseType       CaseType
		expectedPrefix string
	}{
		{CaseTypeChildWelfare, "CW"},
		{CaseTypeCriminal, "CR"},
		{CaseTypeAdministrative, "AD"},
		{CaseTypeHealthcare, "HC"},
		{CaseTypeSocialAssistance, "SA"},
		{CaseTypeTax, "TX"},
		{CaseTypeCivil, "CV"},
	}

	for _, tc := range cases {
		t.Run(string(tc.caseType), func(t *testing.T) {
			c, _ := NewCase(tc.caseType, PriorityMedium, "Number Test", "Testing case number", agencyID, workerID)

			expectedPrefix := tc.expectedPrefix + "-" + string(rune(currentYear/1000+'0')) + string(rune((currentYear%1000)/100+'0'))

			if c.CaseNumber[:5] != expectedPrefix[:5] {
				t.Errorf("Expected case number to start with %s, got %s", expectedPrefix, c.CaseNumber[:5])
			}
		})
	}
}
