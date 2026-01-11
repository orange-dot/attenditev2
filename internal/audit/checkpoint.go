package audit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/serbia-gov/platform/internal/shared/errors"
	"github.com/serbia-gov/platform/internal/shared/types"
	"github.com/serbia-gov/platform/internal/tsa"
)

// WitnessType defines the type of external witness
type WitnessType string

const (
	WitnessTypeLocal       WitnessType = "local"        // Local storage only (dev)
	WitnessTypeRFC3161TSA  WitnessType = "rfc3161_tsa"  // Internal RFC 3161 TSA
	WitnessTypeMultiAgency WitnessType = "multi_agency" // Multi-agency distributed witness
)

// WitnessStatus defines the status of witness confirmation
type WitnessStatus string

const (
	WitnessStatusPending   WitnessStatus = "pending"
	WitnessStatusConfirmed WitnessStatus = "confirmed"
	WitnessStatusFailed    WitnessStatus = "failed"
)

// Checkpoint represents an audit chain checkpoint with external witness
type Checkpoint struct {
	ID             types.ID      `json:"id"`
	CheckpointHash string        `json:"checkpoint_hash"`
	LastSequence   int64         `json:"last_sequence"`
	LastEntryID    types.ID      `json:"last_entry_id"`
	EntryCount     int           `json:"entry_count"`
	WitnessType    WitnessType   `json:"witness_type"`
	WitnessProof   []byte        `json:"witness_proof,omitempty"`
	WitnessURL     string        `json:"witness_url,omitempty"`
	WitnessStatus  WitnessStatus `json:"witness_status"`
	CreatedAt      time.Time     `json:"created_at"`
	ConfirmedAt    *time.Time    `json:"confirmed_at,omitempty"`
}

// Witness interface for external timestamping services
type Witness interface {
	// Type returns the witness type
	Type() WitnessType

	// Timestamp submits a hash to the witness service and returns proof
	Timestamp(ctx context.Context, hash string, lastSequence int64, entryCount int) (proof []byte, url string, err error)

	// Verify checks if a hash matches the stored proof
	Verify(ctx context.Context, hash string, proof []byte) (bool, error)

	// GetStatus returns the confirmation status of a proof
	GetStatus(ctx context.Context, proof []byte) (WitnessStatus, error)
}

// LocalWitness provides local checkpoint storage (for development)
type LocalWitness struct{}

func NewLocalWitness() *LocalWitness {
	return &LocalWitness{}
}

func (w *LocalWitness) Type() WitnessType {
	return WitnessTypeLocal
}

func (w *LocalWitness) Timestamp(ctx context.Context, hash string, lastSequence int64, entryCount int) ([]byte, string, error) {
	// Local witness just stores the hash with timestamp
	proof := fmt.Sprintf("LOCAL_WITNESS:%s:%d:%d:%d", hash, lastSequence, entryCount, time.Now().UnixNano())
	proofHash := sha256.Sum256([]byte(proof))
	return proofHash[:], "", nil
}

func (w *LocalWitness) Verify(ctx context.Context, hash string, proof []byte) (bool, error) {
	// Local witness can't truly verify, always returns true if proof exists
	return len(proof) > 0, nil
}

func (w *LocalWitness) GetStatus(ctx context.Context, proof []byte) (WitnessStatus, error) {
	// Local witness is immediately confirmed
	return WitnessStatusConfirmed, nil
}

// RFC3161Witness uses internal RFC 3161 TSA for legally valid timestamps.
// This runs entirely on government servers without external dependencies.
type RFC3161Witness struct {
	tsaServer *tsa.Server
}

func NewRFC3161Witness(tsaServer *tsa.Server) *RFC3161Witness {
	return &RFC3161Witness{tsaServer: tsaServer}
}

func (w *RFC3161Witness) Type() WitnessType {
	return WitnessTypeRFC3161TSA
}

func (w *RFC3161Witness) Timestamp(ctx context.Context, hash string, lastSequence int64, entryCount int) ([]byte, string, error) {
	if w.tsaServer == nil {
		return nil, "", fmt.Errorf("TSA server not configured")
	}

	// Create timestamp using internal TSA
	resp, err := w.tsaServer.TimestampHash(ctx, hash)
	if err != nil {
		return nil, "", fmt.Errorf("TSA timestamp failed: %w", err)
	}

	// The token is the proof
	return resp.Token, "", nil
}

func (w *RFC3161Witness) Verify(ctx context.Context, hash string, proof []byte) (bool, error) {
	if w.tsaServer == nil {
		return false, fmt.Errorf("TSA server not configured")
	}

	// Decode hash
	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		return false, fmt.Errorf("invalid hash: %w", err)
	}

	// Verify using TSA
	result, err := w.tsaServer.Verify(ctx, proof, hashBytes)
	if err != nil {
		return false, err
	}

	return result.Valid, nil
}

func (w *RFC3161Witness) GetStatus(ctx context.Context, proof []byte) (WitnessStatus, error) {
	// RFC 3161 timestamps are immediately confirmed
	if len(proof) > 0 {
		return WitnessStatusConfirmed, nil
	}
	return WitnessStatusFailed, nil
}

// MultiAgencyWitness uses distributed signatures from multiple government agencies.
// This provides Byzantine fault tolerance and prevents any single agency from
// being able to forge timestamps.
type MultiAgencyWitness struct {
	witness *tsa.MultiAgencyWitness
}

func NewMultiAgencyWitness(witness *tsa.MultiAgencyWitness) *MultiAgencyWitness {
	return &MultiAgencyWitness{witness: witness}
}

func (w *MultiAgencyWitness) Type() WitnessType {
	return WitnessTypeMultiAgency
}

func (w *MultiAgencyWitness) Timestamp(ctx context.Context, hash string, lastSequence int64, entryCount int) ([]byte, string, error) {
	if w.witness == nil {
		return nil, "", fmt.Errorf("multi-agency witness not configured")
	}

	// Create multi-agency proof
	proof, err := w.witness.CreateProof(ctx, hash, lastSequence, entryCount)
	if err != nil {
		return nil, "", fmt.Errorf("multi-agency proof creation failed: %w", err)
	}

	// Serialize proof
	proofBytes, err := proof.Serialize()
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize proof: %w", err)
	}

	return proofBytes, "", nil
}

func (w *MultiAgencyWitness) Verify(ctx context.Context, hash string, proof []byte) (bool, error) {
	if w.witness == nil {
		return false, fmt.Errorf("multi-agency witness not configured")
	}

	// Deserialize proof
	p, err := tsa.DeserializeProof(proof)
	if err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// Verify hash matches
	if p.CheckpointHash != hash {
		return false, nil
	}

	// Verify all signatures
	result, err := w.witness.VerifyProof(ctx, p)
	if err != nil {
		return false, err
	}

	return result.Valid, nil
}

func (w *MultiAgencyWitness) GetStatus(ctx context.Context, proof []byte) (WitnessStatus, error) {
	// Deserialize and check status
	p, err := tsa.DeserializeProof(proof)
	if err != nil {
		return WitnessStatusFailed, err
	}

	switch p.Status {
	case "confirmed":
		return WitnessStatusConfirmed, nil
	case "pending":
		return WitnessStatusPending, nil
	default:
		return WitnessStatusFailed, nil
	}
}

// CompositeWitness combines multiple witness types for maximum security.
// It collects proofs from all configured witnesses.
type CompositeWitness struct {
	witnesses []Witness
}

func NewCompositeWitness(witnesses ...Witness) *CompositeWitness {
	return &CompositeWitness{witnesses: witnesses}
}

func (w *CompositeWitness) Type() WitnessType {
	return "composite"
}

func (w *CompositeWitness) Timestamp(ctx context.Context, hash string, lastSequence int64, entryCount int) ([]byte, string, error) {
	type proofEntry struct {
		Type  WitnessType `json:"type"`
		Proof string      `json:"proof"` // base64
	}

	var proofs []proofEntry
	for _, witness := range w.witnesses {
		proof, _, err := witness.Timestamp(ctx, hash, lastSequence, entryCount)
		if err != nil {
			// Log but continue with other witnesses
			continue
		}
		proofs = append(proofs, proofEntry{
			Type:  witness.Type(),
			Proof: base64.StdEncoding.EncodeToString(proof),
		})
	}

	if len(proofs) == 0 {
		return nil, "", fmt.Errorf("no witnesses succeeded")
	}

	// Serialize all proofs
	composite := struct {
		Proofs []proofEntry `json:"proofs"`
	}{Proofs: proofs}

	data, err := canonicalJSON(composite)
	if err != nil {
		return nil, "", err
	}

	return data, "", nil
}

func (w *CompositeWitness) Verify(ctx context.Context, hash string, proof []byte) (bool, error) {
	// At least one witness must verify
	for _, witness := range w.witnesses {
		valid, err := witness.Verify(ctx, hash, proof)
		if err == nil && valid {
			return true, nil
		}
	}
	return false, nil
}

func (w *CompositeWitness) GetStatus(ctx context.Context, proof []byte) (WitnessStatus, error) {
	// Return confirmed if any witness is confirmed
	for _, witness := range w.witnesses {
		status, err := witness.GetStatus(ctx, proof)
		if err == nil && status == WitnessStatusConfirmed {
			return WitnessStatusConfirmed, nil
		}
	}
	return WitnessStatusPending, nil
}

// CheckpointService manages audit checkpoints
type CheckpointService struct {
	repo    AuditRepository
	witness Witness
}

func NewCheckpointService(repo AuditRepository, witness Witness) *CheckpointService {
	if witness == nil {
		witness = NewLocalWitness()
	}
	return &CheckpointService{repo: repo, witness: witness}
}

// CreateCheckpoint creates a new checkpoint of the current audit chain state
func (s *CheckpointService) CreateCheckpoint(ctx context.Context) (*Checkpoint, error) {
	// Get last hash and sequence from repository
	lastHash := s.repo.GetLastHash()
	lastSequence := s.repo.GetSequence()

	if lastHash == "" {
		return nil, errors.BadRequest("no audit entries to checkpoint")
	}

	// Get total count
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to count audit entries")
	}

	// Calculate checkpoint hash (hash of: last_hash + sequence + count + timestamp)
	now := time.Now().UTC()
	checkpointData := fmt.Sprintf("%s:%d:%d:%d",
		lastHash, lastSequence, count, now.UnixNano())
	checkpointHashBytes := sha256.Sum256([]byte(checkpointData))
	checkpointHash := hex.EncodeToString(checkpointHashBytes[:])

	// Get witness proof
	proof, url, err := s.witness.Timestamp(ctx, checkpointHash, lastSequence, count)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get witness timestamp")
	}

	// Get initial status
	status, _ := s.witness.GetStatus(ctx, proof)

	// Create checkpoint
	checkpoint := &Checkpoint{
		ID:             types.NewID(),
		CheckpointHash: checkpointHash,
		LastSequence:   lastSequence,
		LastEntryID:    "", // We don't have easy access to last entry ID without another query
		EntryCount:     count,
		WitnessType:    s.witness.Type(),
		WitnessProof:   proof,
		WitnessURL:     url,
		WitnessStatus:  status,
		CreatedAt:      now,
	}

	if status == WitnessStatusConfirmed {
		checkpoint.ConfirmedAt = &now
	}

	// Save checkpoint
	if err := s.repo.SaveCheckpoint(ctx, checkpoint); err != nil {
		return nil, errors.Wrap(err, "failed to save checkpoint")
	}

	return checkpoint, nil
}

// GetLatestCheckpoint returns the most recent checkpoint
func (s *CheckpointService) GetLatestCheckpoint(ctx context.Context) (*Checkpoint, error) {
	return s.repo.GetLatestCheckpoint(ctx)
}

// VerifyCheckpoint verifies that the checkpoint matches the current chain state
func (s *CheckpointService) VerifyCheckpoint(ctx context.Context, checkpointID types.ID) (*CheckpointVerifyResult, error) {
	// Get checkpoint
	cp, err := s.repo.GetCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, err
	}

	result := &CheckpointVerifyResult{
		Checkpoint:    cp,
		ChainValid:    true,
		WitnessValid:  true,
		EntriesIntact: true,
	}

	// Verify entry count matches (KurrentDB is append-only, so deletions are impossible)
	count, err := s.repo.Count(ctx)
	if err != nil {
		result.ChainValid = false
		result.Violations = append(result.Violations, "Failed to count entries: "+err.Error())
	} else if count < cp.EntryCount {
		// In KurrentDB this should never happen as events can't be deleted
		result.EntriesIntact = false
		result.Violations = append(result.Violations,
			fmt.Sprintf("Entry count mismatch: expected %d, found %d (impossible in KurrentDB)",
				cp.EntryCount, count))
	}

	// Verify witness proof
	if s.witness.Type() == cp.WitnessType {
		valid, err := s.witness.Verify(ctx, cp.CheckpointHash, cp.WitnessProof)
		if err != nil || !valid {
			result.WitnessValid = false
			result.Violations = append(result.Violations, "Witness proof verification failed")
		}
	}

	result.Valid = result.ChainValid && result.WitnessValid && result.EntriesIntact

	return result, nil
}

// ListCheckpoints returns all checkpoints
func (s *CheckpointService) ListCheckpoints(ctx context.Context, limit int) ([]Checkpoint, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListCheckpoints(ctx, limit)
}

// CheckpointVerifyResult contains checkpoint verification results
type CheckpointVerifyResult struct {
	Checkpoint    *Checkpoint `json:"checkpoint"`
	Valid         bool        `json:"valid"`
	ChainValid    bool        `json:"chain_valid"`
	WitnessValid  bool        `json:"witness_valid"`
	EntriesIntact bool        `json:"entries_intact"`
	Violations    []string    `json:"violations,omitempty"`
}
