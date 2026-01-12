package internal

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/agency"
	casedomain "github.com/serbia-gov/platform/internal/case/domain"
	"github.com/serbia-gov/platform/internal/document"
	"github.com/serbia-gov/platform/internal/privacy"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// TestFullCaseWorkflow tests the complete case lifecycle
func TestFullCaseWorkflow(t *testing.T) {
	// Setup
	agencyID := types.NewID()
	workerID := types.NewID()

	// 1. Create a new case
	c, err := casedomain.NewCase(
		casedomain.CaseTypeChildWelfare,
		casedomain.PriorityHigh,
		"Child welfare investigation",
		"Investigation into reported child welfare concerns in household",
		agencyID,
		workerID,
	)
	if err != nil {
		t.Fatalf("Failed to create case: %v", err)
	}

	if c.Status != casedomain.CaseStatusDraft {
		t.Errorf("New case should be in draft status, got %s", c.Status)
	}

	// 2. Open the case
	err = c.Open(workerID, agencyID)
	if err != nil {
		t.Fatalf("Failed to open case: %v", err)
	}

	if c.Status != casedomain.CaseStatusOpen {
		t.Errorf("Opened case should be in open status, got %s", c.Status)
	}

	// 3. Start progress
	err = c.StartProgress(workerID, agencyID)
	if err != nil {
		t.Fatalf("Failed to start progress: %v", err)
	}

	if c.Status != casedomain.CaseStatusInProgress {
		t.Errorf("Case should be in progress, got %s", c.Status)
	}

	// 4. Add a participant
	participant := casedomain.Participant{
		Name: "Witness Person",
		Role: "witness",
	}
	c.AddParticipant(participant, workerID, agencyID)

	if len(c.Participants) == 0 {
		t.Error("Participant should be added")
	}

	// 5. Assign a worker
	assigneeID := types.NewID()
	c.Assign(assigneeID, agencyID, casedomain.AssignmentRoleSupport, workerID, agencyID)

	if len(c.Assignments) == 0 {
		t.Error("Assignment should be added")
	}

	// 6. Close the case - complete assignment first
	c.Assignments[0].Complete()
	err = c.Close(workerID, agencyID, "Investigation completed successfully")
	if err != nil {
		t.Fatalf("Failed to close case: %v", err)
	}

	if c.Status != casedomain.CaseStatusClosed {
		t.Errorf("Case should be closed, got %s", c.Status)
	}

	// 7. Verify domain events were generated
	events := c.GetDomainEvents()
	if len(events) == 0 {
		t.Error("Domain events should have been generated")
	}
}

// TestDocumentSigningWorkflow tests the complete document signing lifecycle
func TestDocumentSigningWorkflow(t *testing.T) {
	// Setup
	agencyID := types.NewID()
	creatorID := types.NewID()
	signer1ID := types.NewID()
	signer2ID := types.NewID()

	// 1. Create document
	doc, err := document.NewDocument(
		document.DocumentTypeDecision,
		"Case Closure Decision",
		"Official decision to close the case",
		agencyID,
		creatorID,
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	if doc.Status != document.DocumentStatusDraft {
		t.Errorf("New document should be in draft status, got %s", doc.Status)
	}

	// 2. Add a version
	content := []byte("This is the official decision document content...")
	_, err = doc.AddVersion(
		"/documents/2024/decision-001.pdf",
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
		creatorID,
		"Initial version",
	)
	if err != nil {
		t.Fatalf("Failed to add version: %v", err)
	}

	if doc.CurrentVersion != 1 {
		t.Errorf("Document should be at version 1, got %d", doc.CurrentVersion)
	}

	// 3. Request signatures from two signers
	_, err = doc.RequestSignature(signer1ID, agencyID, creatorID, document.SignatureTypeQualified, nil, "Approval required", "Belgrade")
	if err != nil {
		t.Fatalf("Failed to request signature 1: %v", err)
	}

	_, err = doc.RequestSignature(signer2ID, agencyID, creatorID, document.SignatureTypeQualified, nil, "Approval required", "Belgrade")
	if err != nil {
		t.Fatalf("Failed to request signature 2: %v", err)
	}

	if doc.Status != document.DocumentStatusPendingSignature {
		t.Errorf("Document should be pending signature, got %s", doc.Status)
	}

	// 4. First signer signs
	err = doc.Sign(signer1ID, []byte("sig-data-1"), []byte("cert-1"), []byte("timestamp-1"))
	if err != nil {
		t.Fatalf("Failed to sign as signer 1: %v", err)
	}

	if doc.Status != document.DocumentStatusPartiallySigned {
		t.Errorf("Document should be partially signed, got %s", doc.Status)
	}

	// 5. Second signer signs
	err = doc.Sign(signer2ID, []byte("sig-data-2"), []byte("cert-2"), []byte("timestamp-2"))
	if err != nil {
		t.Fatalf("Failed to sign as signer 2: %v", err)
	}

	if doc.Status != document.DocumentStatusSigned {
		t.Errorf("Document should be fully signed, got %s", doc.Status)
	}

	// 6. Archive the signed document
	err = doc.Archive()
	if err != nil {
		t.Fatalf("Failed to archive document: %v", err)
	}

	if doc.Status != document.DocumentStatusArchived {
		t.Errorf("Document should be archived, got %s", doc.Status)
	}
}

// TestPseudonymizationWorkflow tests the privacy pseudonymization flow
func TestPseudonymizationWorkflow(t *testing.T) {
	ctx := context.Background()

	// Setup mock repository
	repo := newMockPseudonymRepo()

	// Create pseudonymization service
	svc := privacy.NewPseudonymizationService(
		[]byte("test-hmac-key-32-bytes-long!!!!"),
		"CSW-BG-01",
		repo,
		nil,
	)

	// 1. Pseudonymize multiple JMBGs
	jmbgs := []string{
		"0101990123456",
		"0202985654321",
		"0303980111222",
	}

	pseudonyms, err := svc.PseudonymizeMany(ctx, jmbgs)
	if err != nil {
		t.Fatalf("Failed to pseudonymize: %v", err)
	}

	if len(pseudonyms) != 3 {
		t.Errorf("Expected 3 pseudonyms, got %d", len(pseudonyms))
	}

	// 2. Verify determinism - same JMBG produces same pseudonym
	for _, jmbg := range jmbgs {
		p1, _ := svc.Pseudonymize(ctx, jmbg)
		p2, _ := svc.Pseudonymize(ctx, jmbg)

		if p1 != p2 {
			t.Errorf("Pseudonymization should be deterministic for JMBG %s", jmbg)
		}
	}

	// 3. Verify all pseudonyms exist
	for _, pseudonym := range pseudonyms {
		exists, err := svc.Exists(ctx, pseudonym)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if !exists {
			t.Errorf("Pseudonym %s should exist", pseudonym)
		}
	}

	// 4. Test GDPR deletion
	firstPseudonym := pseudonyms["0101990123456"]
	err = svc.Delete(ctx, firstPseudonym)
	if err != nil {
		t.Fatalf("Failed to delete pseudonym: %v", err)
	}

	// 5. Verify deleted pseudonym no longer exists
	exists, _ := svc.Exists(ctx, firstPseudonym)
	if exists {
		t.Error("Deleted pseudonym should not exist")
	}
}

// TestAgencyWorkerRelationship tests agency and worker domain relationship
func TestAgencyWorkerRelationship(t *testing.T) {
	// Create an agency
	agencyID := types.NewID()
	parentID := types.NewID()

	agencyEntity := agency.Agency{
		ID:       agencyID,
		Code:     "CSW-BG-01",
		Name:     "Center for Social Work Belgrade",
		Type:     agency.AgencyTypeSocialServices,
		ParentID: &parentID,
		Status:   agency.AgencyStatusActive,
		Address: types.Address{
			Street:     "Ruzveltova 61",
			City:       "Belgrade",
			PostalCode: "11000",
			Country:    "RS",
		},
		Contact: types.ContactInfo{
			Phone: "+381 11 765 4321",
			Email: "info@csw-bg.gov.rs",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create workers for the agency
	workers := []agency.Worker{
		{
			ID:         types.NewID(),
			AgencyID:   agencyID,
			EmployeeID: "CSW-001",
			FirstName:  "Marija",
			LastName:   "Jovanovic",
			Email:      "marija.jovanovic@csw-bg.gov.rs",
			Position:   "Social Worker",
			Department: "Child Protection",
			Status:     agency.WorkerStatusActive,
			Roles: []agency.WorkerRole{
				{
					ID:        types.NewID(),
					Role:      "case_worker",
					Scope:     "agency",
					GrantedAt: time.Now(),
					GrantedBy: types.NewID(),
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:         types.NewID(),
			AgencyID:   agencyID,
			EmployeeID: "CSW-002",
			FirstName:  "Petar",
			LastName:   "Petrovic",
			Email:      "petar.petrovic@csw-bg.gov.rs",
			Position:   "Senior Social Worker",
			Department: "Child Protection",
			Status:     agency.WorkerStatusActive,
			Roles: []agency.WorkerRole{
				{
					ID:        types.NewID(),
					Role:      "supervisor",
					Scope:     "department",
					GrantedAt: time.Now(),
					GrantedBy: types.NewID(),
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Verify agency status
	if agencyEntity.Status != agency.AgencyStatusActive {
		t.Errorf("Agency should be active, got %s", agencyEntity.Status)
	}

	// Verify workers belong to agency
	for _, worker := range workers {
		if worker.AgencyID != agencyID {
			t.Error("Worker should belong to the agency")
		}
	}

	// Verify worker full names
	if workers[0].FullName() != "Marija Jovanovic" {
		t.Errorf("Expected 'Marija Jovanovic', got '%s'", workers[0].FullName())
	}

	// Verify workers have roles
	for _, worker := range workers {
		if len(worker.Roles) == 0 {
			t.Errorf("Worker %s should have roles", worker.FullName())
		}
	}

	// Verify different workers have different employee IDs
	if workers[0].EmployeeID == workers[1].EmployeeID {
		t.Error("Workers should have different employee IDs")
	}
}

// --- Mock Repository for Integration Tests ---

type mockPseudonymRepo struct {
	mappings map[string]*privacy.PseudonymMapping
}

func newMockPseudonymRepo() *mockPseudonymRepo {
	return &mockPseudonymRepo{
		mappings: make(map[string]*privacy.PseudonymMapping),
	}
}

func (r *mockPseudonymRepo) Store(ctx context.Context, mapping *privacy.PseudonymMapping) error {
	r.mappings[mapping.JMBGHash] = mapping
	return nil
}

func (r *mockPseudonymRepo) GetByJMBGHash(ctx context.Context, jmbgHash, facilityCode string) (*privacy.PseudonymMapping, error) {
	if m, ok := r.mappings[jmbgHash]; ok && m.FacilityCode == facilityCode {
		return m, nil
	}
	return nil, nil
}

func (r *mockPseudonymRepo) GetByPseudonymID(ctx context.Context, pseudonymID privacy.PseudonymID) (*privacy.PseudonymMapping, error) {
	for _, m := range r.mappings {
		if m.PseudonymID == pseudonymID {
			return m, nil
		}
	}
	return nil, nil
}

func (r *mockPseudonymRepo) Delete(ctx context.Context, pseudonymID privacy.PseudonymID) error {
	for k, m := range r.mappings {
		if m.PseudonymID == pseudonymID {
			delete(r.mappings, k)
			return nil
		}
	}
	return nil
}
