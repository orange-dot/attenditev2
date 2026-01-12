package audit

import (
	"testing"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// TestNewAuditEntry tests creating a new audit entry
func TestNewAuditEntry(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()
	resourceID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		&agencyID,
		ActionCaseCreated,
		"case",
		&resourceID,
		map[string]any{"title": "Test Case"},
		"",
	)

	if entry.ID.IsZero() {
		t.Error("Expected non-zero ID")
	}

	if entry.ActorType != ActorTypeWorker {
		t.Errorf("Expected ActorTypeWorker, got %s", entry.ActorType)
	}

	if entry.ActorID != actorID {
		t.Errorf("Expected actorID %s, got %s", actorID, entry.ActorID)
	}

	if entry.Action != ActionCaseCreated {
		t.Errorf("Expected action %s, got %s", ActionCaseCreated, entry.Action)
	}

	if entry.Hash == "" {
		t.Error("Expected non-empty hash")
	}

	if entry.PrevHash != "" {
		t.Error("Expected empty prev_hash for first entry")
	}
}

// TestHashChainIntegrity tests that hash chain links are valid
func TestHashChainIntegrity(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()

	// Create a chain of entries
	entries := make([]*AuditEntry, 5)

	prevHash := ""
	for i := 0; i < 5; i++ {
		resourceID := types.NewID()
		entries[i] = NewAuditEntry(
			ActorTypeWorker,
			actorID,
			&agencyID,
			ActionCaseCreated,
			"case",
			&resourceID,
			map[string]any{"index": i},
			prevHash,
		)
		prevHash = entries[i].Hash
	}

	// Verify chain integrity
	for i := 1; i < len(entries); i++ {
		if entries[i].PrevHash != entries[i-1].Hash {
			t.Errorf("Chain broken at entry %d: expected prev_hash %s, got %s",
				i, entries[i-1].Hash, entries[i].PrevHash)
		}
	}
}

// TestHashChainTamperDetection tests that modifying an entry invalidates its hash
func TestHashChainTamperDetection(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()
	resourceID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		&agencyID,
		ActionCaseCreated,
		"case",
		&resourceID,
		map[string]any{"title": "Original"},
		"",
	)

	originalHash := entry.Hash

	// Verify hash is valid initially
	if !entry.VerifyHash() {
		t.Error("Hash should be valid before tampering")
	}

	// Tamper with the entry
	entry.Changes["title"] = "Tampered"

	// Hash should now be invalid
	if entry.VerifyHash() {
		t.Error("Hash should be invalid after tampering")
	}

	// Verify hash changed
	computedHash := entry.ComputeHash()
	if computedHash == originalHash {
		t.Error("Computed hash should differ after tampering")
	}
}

// TestVerifyHash tests hash verification
func TestVerifyHash(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()
	resourceID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		&agencyID,
		ActionDocumentSigned,
		"document",
		&resourceID,
		map[string]any{
			"signer": actorID.String(),
			"type":   "qualified",
		},
		"abc123prevhash",
	)

	if !entry.VerifyHash() {
		t.Error("Hash should be valid for newly created entry")
	}

	// Verify that PrevHash is correctly set
	if entry.PrevHash != "abc123prevhash" {
		t.Errorf("Expected prev_hash 'abc123prevhash', got '%s'", entry.PrevHash)
	}
}

// TestCanonicalJSONDeterminism tests that canonical JSON produces consistent output
func TestCanonicalJSONDeterminism(t *testing.T) {
	// Create entries with same data but potentially different internal key ordering
	actorID := types.NewID()
	agencyID := types.NewID()
	resourceID := types.NewID()

	changes := map[string]any{
		"zebra":    "last",
		"apple":    "first",
		"middle":   "middle",
		"nested": map[string]any{
			"z": 3,
			"a": 1,
			"m": 2,
		},
	}

	entry1 := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		&agencyID,
		ActionCaseUpdated,
		"case",
		&resourceID,
		changes,
		"prevhash",
	)

	// Create another entry with the same data
	entry2 := &AuditEntry{
		ID:            entry1.ID,
		Timestamp:     entry1.Timestamp,
		PrevHash:      entry1.PrevHash,
		ActorType:     entry1.ActorType,
		ActorID:       entry1.ActorID,
		ActorAgencyID: entry1.ActorAgencyID,
		Action:        entry1.Action,
		ResourceType:  entry1.ResourceType,
		ResourceID:    entry1.ResourceID,
		Changes:       changes,
	}
	entry2.Hash = entry2.calculateHash()

	// Hashes should be identical
	if entry1.Hash != entry2.Hash {
		t.Errorf("Hashes should be identical for same data: got %s and %s", entry1.Hash, entry2.Hash)
	}
}

// TestEntryTimestampPrecision tests that timestamps are handled correctly
func TestEntryTimestampPrecision(t *testing.T) {
	actorID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeSystem,
		actorID,
		nil,
		ActionLogin,
		"auth",
		nil,
		nil,
		"",
	)

	// Timestamp should be truncated to microseconds for PostgreSQL compatibility
	if entry.Timestamp.Nanosecond()%1000 != 0 {
		t.Error("Timestamp should be truncated to microseconds")
	}

	// Timestamp should be in UTC
	if entry.Timestamp.Location() != time.UTC {
		t.Error("Timestamp should be in UTC")
	}
}

// TestCorrectionEntry tests creating a correction entry
func TestCorrectionEntry(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()
	originalEntryID := types.NewID()
	approverID := types.NewID()

	correction := CorrectionEntry{
		OriginalEntryID:   originalEntryID,
		OriginalAction:    ActionCaseCreated,
		OriginalTimestamp: time.Now().Add(-24 * time.Hour),
		Reason:            CorrectionReasonDataEntry,
		Justification:     "Typo in case title",
		ApprovedBy:        &approverID,
		OldValue:          map[string]any{"title": "Tset Case"},
		NewValue:          map[string]any{"title": "Test Case"},
	}

	entry := NewCorrectionAuditEntry(
		ActorTypeWorker,
		actorID,
		&agencyID,
		correction,
		"prevhash123",
	)

	if entry.Action != ActionCorrection {
		t.Errorf("Expected action %s, got %s", ActionCorrection, entry.Action)
	}

	if entry.ResourceType != "correction" {
		t.Errorf("Expected resource_type 'correction', got '%s'", entry.ResourceType)
	}

	if *entry.ResourceID != originalEntryID {
		t.Errorf("ResourceID should reference original entry")
	}

	// Verify correction data is in changes
	if entry.Changes == nil {
		t.Error("Correction entry should have changes")
	}

	correctionData, ok := entry.Changes["correction"].(map[string]any)
	if !ok {
		t.Error("Changes should contain 'correction' map")
	}

	if correctionData["reason"] != CorrectionReasonDataEntry {
		t.Errorf("Expected reason %s, got %v", CorrectionReasonDataEntry, correctionData["reason"])
	}
}

// TestWithContext tests adding context to an entry
func TestWithContext(t *testing.T) {
	actorID := types.NewID()
	correlationID := types.NewID()
	sessionID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		nil,
		ActionCaseViewed,
		"case",
		nil,
		nil,
		"",
	)

	entry.WithContext(&correlationID, &sessionID, "Viewing case for review")

	if *entry.CorrelationID != correlationID {
		t.Error("CorrelationID not set correctly")
	}

	if *entry.SessionID != sessionID {
		t.Error("SessionID not set correctly")
	}

	if entry.Justification != "Viewing case for review" {
		t.Error("Justification not set correctly")
	}
}

// TestWithRequest tests adding request info to an entry
func TestWithRequest(t *testing.T) {
	actorID := types.NewID()

	entry := NewAuditEntry(
		ActorTypeWorker,
		actorID,
		nil,
		ActionLogin,
		"auth",
		nil,
		nil,
		"",
	)

	entry.WithRequest("192.168.1.100", "Mozilla/5.0 (Windows NT 10.0)")

	if entry.ActorIP != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", entry.ActorIP)
	}

	if entry.ActorDevice != "Mozilla/5.0 (Windows NT 10.0)" {
		t.Errorf("Expected device info, got '%s'", entry.ActorDevice)
	}
}

// TestChainVerificationWithMultipleEntries tests verifying a longer chain
func TestChainVerificationWithMultipleEntries(t *testing.T) {
	actorID := types.NewID()
	agencyID := types.NewID()

	// Create a chain of 100 entries
	entries := make([]*AuditEntry, 100)
	prevHash := ""

	for i := 0; i < 100; i++ {
		resourceID := types.NewID()
		entries[i] = NewAuditEntry(
			ActorTypeWorker,
			actorID,
			&agencyID,
			ActionCaseCreated,
			"case",
			&resourceID,
			map[string]any{"index": i, "timestamp": time.Now().Unix()},
			prevHash,
		)
		prevHash = entries[i].Hash
	}

	// Verify all hashes
	for i, entry := range entries {
		if !entry.VerifyHash() {
			t.Errorf("Entry %d has invalid hash", i)
		}
	}

	// Verify chain links
	for i := 1; i < len(entries); i++ {
		if entries[i].PrevHash != entries[i-1].Hash {
			t.Errorf("Chain broken at entry %d", i)
		}
	}

	// Tamper with middle entry and verify chain is broken
	middleIndex := 50
	entries[middleIndex].Changes["index"] = 999

	// The tampered entry's hash should now be invalid
	if entries[middleIndex].VerifyHash() {
		t.Error("Tampered entry should have invalid hash")
	}

	// Entries after the tampered one should have correct prev_hash linking
	// but the link is now broken because we changed the content
	expectedPrevHash := entries[middleIndex-1].Hash
	if entries[middleIndex].PrevHash != expectedPrevHash {
		t.Errorf("PrevHash should still reference previous entry's hash")
	}
}

// TestActorTypes tests different actor types
func TestActorTypes(t *testing.T) {
	tests := []struct {
		name      string
		actorType ActorType
	}{
		{"Citizen", ActorTypeCitizen},
		{"Worker", ActorTypeWorker},
		{"System", ActorTypeSystem},
		{"External", ActorTypeExternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actorID := types.NewID()
			entry := NewAuditEntry(
				tt.actorType,
				actorID,
				nil,
				ActionLogin,
				"auth",
				nil,
				nil,
				"",
			)

			if entry.ActorType != tt.actorType {
				t.Errorf("Expected actor type %s, got %s", tt.actorType, entry.ActorType)
			}

			if !entry.VerifyHash() {
				t.Error("Hash should be valid")
			}
		})
	}
}
