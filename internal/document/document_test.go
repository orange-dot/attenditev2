package document

import (
	"bytes"
	"testing"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// TestNewDocument tests creating a new document
func TestNewDocument(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	caseID := types.NewID()

	doc, err := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"Description of the report",
		agencyID,
		workerID,
		&caseID,
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.ID.IsZero() {
		t.Error("Expected non-zero ID")
	}

	if doc.Type != DocumentTypeReport {
		t.Errorf("Expected type REPORT, got %s", doc.Type)
	}

	if doc.Status != DocumentStatusDraft {
		t.Errorf("Expected status draft, got %s", doc.Status)
	}

	if doc.Title != "Test Report" {
		t.Errorf("Expected title 'Test Report', got %s", doc.Title)
	}

	if doc.OwnerAgencyID != agencyID {
		t.Error("Owner agency ID mismatch")
	}

	if *doc.CaseID != caseID {
		t.Error("Case ID mismatch")
	}

	if doc.CurrentVersion != 0 {
		t.Errorf("Expected version 0, got %d", doc.CurrentVersion)
	}

	if doc.DocumentNumber == "" {
		t.Error("Document number should be generated")
	}
}

// TestNewDocumentValidation tests document validation
func TestNewDocumentValidation(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	tests := []struct {
		name          string
		title         string
		agencyID      types.ID
		expectError   bool
		errorContains string
	}{
		{
			name:          "Empty title",
			title:         "",
			agencyID:      agencyID,
			expectError:   true,
			errorContains: "title",
		},
		{
			name:          "Zero agency ID",
			title:         "Test",
			agencyID:      types.ID(""),
			expectError:   true,
			errorContains: "owner agency",
		},
		{
			name:        "Valid document",
			title:       "Test",
			agencyID:    agencyID,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDocument(
				DocumentTypeReport,
				tt.title,
				"",
				tt.agencyID,
				workerID,
				nil,
			)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestAddVersion tests adding document versions
func TestAddVersion(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("This is the document content")
	version, err := doc.AddVersion(
		"/path/to/file.pdf",
		"application/pdf",
		int64(len(content)),
		bytes.NewReader(content),
		workerID,
		"Initial version",
	)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.CurrentVersion != 1 {
		t.Errorf("Expected version 1, got %d", doc.CurrentVersion)
	}

	if version.Version != 1 {
		t.Errorf("Expected version 1, got %d", version.Version)
	}

	if version.FileHash == "" {
		t.Error("File hash should be calculated")
	}

	if version.FilePath != "/path/to/file.pdf" {
		t.Errorf("File path mismatch")
	}

	if version.MimeType != "application/pdf" {
		t.Errorf("MIME type mismatch")
	}

	if version.ChangeSummary != "Initial version" {
		t.Errorf("Change summary mismatch")
	}

	if len(doc.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(doc.Versions))
	}
}

// TestAddVersionResetsSignatures tests that adding a version resets signatures
func TestAddVersionResetsSignatures(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	// Add initial version
	content := []byte("content v1")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	// Request signature
	doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeSimple, nil, "", "")

	// Sign the document
	doc.Sign(signerID, []byte("sig"), nil, nil)

	if doc.Status != DocumentStatusSigned {
		t.Errorf("Expected signed status, got %s", doc.Status)
	}

	// Add new version
	content2 := []byte("content v2")
	doc.AddVersion("/path/v2.pdf", "application/pdf", int64(len(content2)), bytes.NewReader(content2), workerID, "v2")

	// Status should reset to draft
	if doc.Status != DocumentStatusDraft {
		t.Errorf("Expected draft status after new version, got %s", doc.Status)
	}

	// Signatures should be cleared
	if len(doc.Signatures) != 0 {
		t.Errorf("Signatures should be cleared, got %d", len(doc.Signatures))
	}

	if doc.CurrentVersion != 2 {
		t.Errorf("Expected version 2, got %d", doc.CurrentVersion)
	}
}

// TestAddVersionToVoidedDocument tests that voided documents cannot have new versions
func TestAddVersionToVoidedDocument(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	// Add a version first
	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	// Void the document
	doc.Void()

	// Try to add another version
	content2 := []byte("content v2")
	_, err := doc.AddVersion("/path/v2.pdf", "application/pdf", int64(len(content2)), bytes.NewReader(content2), workerID, "v2")

	if err == nil {
		t.Error("Expected error when adding version to voided document")
	}
}

// TestRequestSignature tests requesting signatures
func TestRequestSignature(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	// Add a version first
	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	sig, err := doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeQualified, nil, "For approval", "Belgrade")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if sig.SignerID != signerID {
		t.Error("Signer ID mismatch")
	}

	if sig.Type != SignatureTypeQualified {
		t.Errorf("Expected qualified signature type, got %s", sig.Type)
	}

	if sig.Status != SignatureStatusPending {
		t.Errorf("Expected pending status, got %s", sig.Status)
	}

	if sig.Reason != "For approval" {
		t.Errorf("Reason mismatch")
	}

	if sig.Location != "Belgrade" {
		t.Errorf("Location mismatch")
	}

	if doc.Status != DocumentStatusPendingSignature {
		t.Errorf("Expected pending_signature status, got %s", doc.Status)
	}
}

// TestRequestSignatureWithoutVersion tests that signature request fails without version
func TestRequestSignatureWithoutVersion(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	_, err := doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeSimple, nil, "", "")

	if err == nil {
		t.Error("Expected error when requesting signature without version")
	}
}

// TestDuplicateSignatureRequest tests that duplicate signature requests are rejected
func TestDuplicateSignatureRequest(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	// First request
	doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeSimple, nil, "", "")

	// Duplicate request
	_, err := doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeSimple, nil, "", "")

	if err == nil {
		t.Error("Expected error for duplicate signature request")
	}
}

// TestSign tests signing a document
func TestSign(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")
	doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeQualified, nil, "", "")

	signatureData := []byte("signature-data")
	certificate := []byte("certificate-data")
	timestamp := []byte("timestamp-token")

	err := doc.Sign(signerID, signatureData, certificate, timestamp)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.Status != DocumentStatusSigned {
		t.Errorf("Expected signed status, got %s", doc.Status)
	}

	if doc.Signatures[0].Status != SignatureStatusSigned {
		t.Errorf("Expected signature status signed, got %s", doc.Signatures[0].Status)
	}

	if doc.Signatures[0].SignedAt == nil {
		t.Error("SignedAt should be set")
	}

	if !bytes.Equal(doc.Signatures[0].SignatureData, signatureData) {
		t.Error("Signature data mismatch")
	}
}

// TestPartialSignature tests multiple signers with partial completion
func TestPartialSignature(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signer1 := types.NewID()
	signer2 := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeContract,
		"Test Contract",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	// Request from two signers
	doc.RequestSignature(signer1, agencyID, workerID, SignatureTypeQualified, nil, "", "")
	doc.RequestSignature(signer2, agencyID, workerID, SignatureTypeQualified, nil, "", "")

	// Only first signer signs
	doc.Sign(signer1, []byte("sig1"), nil, nil)

	if doc.Status != DocumentStatusPartiallySigned {
		t.Errorf("Expected partially_signed status, got %s", doc.Status)
	}

	// Second signer signs
	doc.Sign(signer2, []byte("sig2"), nil, nil)

	if doc.Status != DocumentStatusSigned {
		t.Errorf("Expected signed status after all signers, got %s", doc.Status)
	}
}

// TestRejectSignature tests rejecting a signature
func TestRejectSignature(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	signerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")
	doc.RequestSignature(signerID, agencyID, workerID, SignatureTypeSimple, nil, "", "")

	err := doc.RejectSignature(signerID, "Document contains errors")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.Status != DocumentStatusRejected {
		t.Errorf("Expected rejected status, got %s", doc.Status)
	}

	if doc.Signatures[0].Status != SignatureStatusRejected {
		t.Errorf("Expected signature status rejected, got %s", doc.Signatures[0].Status)
	}

	if doc.Signatures[0].Reason != "Document contains errors" {
		t.Errorf("Rejection reason mismatch")
	}
}

// TestDocumentArchive tests archiving a document
func TestDocumentArchive(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	err := doc.Archive()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.Status != DocumentStatusArchived {
		t.Errorf("Expected archived status, got %s", doc.Status)
	}
}

// TestArchiveVoidedDocument tests that voided documents cannot be archived
func TestArchiveVoidedDocument(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	doc.Void()
	err := doc.Archive()

	if err == nil {
		t.Error("Expected error when archiving voided document")
	}
}

// TestDocumentVoid tests voiding a document
func TestDocumentVoid(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	err := doc.Void()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if doc.Status != DocumentStatusVoid {
		t.Errorf("Expected void status, got %s", doc.Status)
	}
}

// TestVoidArchivedDocument tests that archived documents cannot be voided
func TestVoidArchivedDocument(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	doc.Archive()
	err := doc.Void()

	if err == nil {
		t.Error("Expected error when voiding archived document")
	}
}

// TestDocumentShare tests sharing a document
func TestDocumentShare(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	otherAgencyID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	err := doc.Share(otherAgencyID)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(doc.SharedWith) != 1 {
		t.Errorf("Expected 1 shared agency, got %d", len(doc.SharedWith))
	}

	if doc.SharedWith[0] != otherAgencyID {
		t.Error("Shared agency ID mismatch")
	}

	// Share again - should be idempotent
	err = doc.Share(otherAgencyID)
	if err != nil {
		t.Fatalf("Expected no error for duplicate share, got: %v", err)
	}

	if len(doc.SharedWith) != 1 {
		t.Errorf("Expected still 1 shared agency, got %d", len(doc.SharedWith))
	}
}

// TestShareWithOwner tests that sharing with owner fails
func TestShareWithOwner(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	err := doc.Share(agencyID)

	if err == nil {
		t.Error("Expected error when sharing with owner")
	}
}

// TestDocumentAccess tests access control
func TestDocumentAccess(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	otherAgencyID := types.NewID()
	thirdAgencyID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	// Owner has access
	if !doc.CanAccess(agencyID) {
		t.Error("Owner should have access")
	}

	// Other agency doesn't have access
	if doc.CanAccess(otherAgencyID) {
		t.Error("Other agency should not have access initially")
	}

	// Share with other agency
	doc.Share(otherAgencyID)

	// Other agency now has access
	if !doc.CanAccess(otherAgencyID) {
		t.Error("Shared agency should have access")
	}

	// Third agency still doesn't have access
	if doc.CanAccess(thirdAgencyID) {
		t.Error("Third agency should not have access")
	}
}

// TestDocumentNumberGeneration tests document number format
func TestDocumentNumberGeneration(t *testing.T) {
	tests := []struct {
		docType        DocumentType
		expectedPrefix string
	}{
		{DocumentTypeReport, "RPT"},
		{DocumentTypeStatement, "STM"},
		{DocumentTypeDecision, "DEC"},
		{DocumentTypeCertificate, "CRT"},
		{DocumentTypeEvidence, "EVD"},
		{DocumentTypeForm, "FRM"},
		{DocumentTypeCorrespondence, "COR"},
		{DocumentTypeContract, "CON"},
		{DocumentTypeOther, "DOC"},
	}

	agencyID := types.NewID()
	workerID := types.NewID()

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			doc, err := NewDocument(
				tt.docType,
				"Test",
				"",
				agencyID,
				workerID,
				nil,
			)

			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			if len(doc.DocumentNumber) < 3 {
				t.Error("Document number too short")
			}

			prefix := doc.DocumentNumber[:3]
			if prefix != tt.expectedPrefix {
				t.Errorf("Expected prefix %s, got %s", tt.expectedPrefix, prefix)
			}
		})
	}
}

// TestSignWithoutPendingRequest tests signing without a pending request
func TestSignWithoutPendingRequest(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	randomID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	err := doc.Sign(randomID, []byte("sig"), nil, nil)

	if err == nil {
		t.Error("Expected error when signing without pending request")
	}
}

// TestRejectWithoutPendingRequest tests rejecting without a pending request
func TestRejectWithoutPendingRequest(t *testing.T) {
	agencyID := types.NewID()
	workerID := types.NewID()
	randomID := types.NewID()

	doc, _ := NewDocument(
		DocumentTypeReport,
		"Test Report",
		"",
		agencyID,
		workerID,
		nil,
	)

	content := []byte("content")
	doc.AddVersion("/path/v1.pdf", "application/pdf", int64(len(content)), bytes.NewReader(content), workerID, "v1")

	err := doc.RejectSignature(randomID, "reason")

	if err == nil {
		t.Error("Expected error when rejecting without pending request")
	}
}
