package tsa

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// MultiAgencyWitness implements distributed trust through multiple agency signatures.
// This provides Byzantine fault tolerance - even if some agencies are compromised,
// the timestamp remains valid as long as a minimum threshold of signatures is present.
type MultiAgencyWitness struct {
	config       *MultiAgencyConfig
	localAgency  *LocalAgency
	httpClient   *http.Client
	mu           sync.RWMutex
}

// LocalAgency represents this server's agency identity for signing.
type LocalAgency struct {
	AgencyCode  string
	AgencyName  string
	PrivateKey  crypto.Signer
	Certificate *x509.Certificate
}

// WitnessRequest is sent to other agencies for co-signing.
type WitnessRequest struct {
	CheckpointHash string    `json:"checkpoint_hash"`
	LastSequence   int64     `json:"last_sequence"`
	EntryCount     int       `json:"entry_count"`
	Timestamp      time.Time `json:"timestamp"`
	RequestingAgency string  `json:"requesting_agency"`
}

// AgencySignature represents a single agency's signature.
type AgencySignature struct {
	AgencyCode  string    `json:"agency_code"`
	AgencyName  string    `json:"agency_name"`
	Signature   string    `json:"signature"`      // Base64-encoded signature
	Certificate string    `json:"certificate"`    // Base64-encoded DER certificate
	SignedAt    time.Time `json:"signed_at"`
}

// MultiAgencyProof contains all agency signatures for a checkpoint.
type MultiAgencyProof struct {
	ID             types.ID          `json:"id"`
	CheckpointHash string            `json:"checkpoint_hash"`
	Signatures     []AgencySignature `json:"signatures"`
	MinRequired    int               `json:"min_required"`
	CreatedAt      time.Time         `json:"created_at"`
	Status         string            `json:"status"` // pending, confirmed, failed
}

// NewMultiAgencyWitness creates a new multi-agency witness system.
func NewMultiAgencyWitness(config *MultiAgencyConfig, localAgency *LocalAgency) (*MultiAgencyWitness, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.MinSignatures < 1 {
		config.MinSignatures = 1
	}
	if config.MinSignatures > len(config.Agencies)+1 { // +1 for local agency
		return nil, fmt.Errorf("min_signatures (%d) exceeds available agencies (%d)",
			config.MinSignatures, len(config.Agencies)+1)
	}

	return &MultiAgencyWitness{
		config:      config,
		localAgency: localAgency,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// CreateProof creates a multi-agency proof by collecting signatures from participating agencies.
func (w *MultiAgencyWitness) CreateProof(ctx context.Context, checkpointHash string, lastSequence int64, entryCount int) (*MultiAgencyProof, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.config.Enabled {
		return nil, fmt.Errorf("multi-agency witness is not enabled")
	}

	now := time.Now().UTC()
	proof := &MultiAgencyProof{
		ID:             types.NewID(),
		CheckpointHash: checkpointHash,
		Signatures:     make([]AgencySignature, 0),
		MinRequired:    w.config.MinSignatures,
		CreatedAt:      now,
		Status:         "pending",
	}

	// Create the witness request
	request := &WitnessRequest{
		CheckpointHash:   checkpointHash,
		LastSequence:     lastSequence,
		EntryCount:       entryCount,
		Timestamp:        now,
		RequestingAgency: w.localAgency.AgencyCode,
	}

	// Sign locally first
	localSig, err := w.signLocally(request)
	if err != nil {
		return nil, fmt.Errorf("failed to sign locally: %w", err)
	}
	proof.Signatures = append(proof.Signatures, *localSig)

	// Collect signatures from other agencies concurrently
	var wg sync.WaitGroup
	sigChan := make(chan *AgencySignature, len(w.config.Agencies))
	errChan := make(chan error, len(w.config.Agencies))

	for _, agency := range w.config.Agencies {
		wg.Add(1)
		go func(ag AgencyWitnessConfig) {
			defer wg.Done()
			sig, err := w.requestSignature(ctx, &ag, request)
			if err != nil {
				errChan <- fmt.Errorf("agency %s: %w", ag.AgencyCode, err)
				return
			}
			sigChan <- sig
		}(agency)
	}

	// Wait for all requests to complete
	wg.Wait()
	close(sigChan)
	close(errChan)

	// Collect successful signatures
	for sig := range sigChan {
		proof.Signatures = append(proof.Signatures, *sig)
	}

	// Check if we have enough signatures
	if len(proof.Signatures) >= w.config.MinSignatures {
		proof.Status = "confirmed"
	} else {
		proof.Status = "failed"
		// Collect errors for debugging
		var errs []error
		for err := range errChan {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return proof, fmt.Errorf("insufficient signatures: got %d, need %d. Errors: %v",
				len(proof.Signatures), w.config.MinSignatures, errs)
		}
	}

	return proof, nil
}

// signLocally signs the request with this agency's private key.
func (w *MultiAgencyWitness) signLocally(request *WitnessRequest) (*AgencySignature, error) {
	if w.localAgency == nil || w.localAgency.PrivateKey == nil {
		return nil, fmt.Errorf("local agency not configured")
	}

	// Create canonical data to sign
	dataToSign := w.createSignatureData(request)
	hash := sha256.Sum256(dataToSign)

	// Sign the hash
	signature, err := w.localAgency.PrivateKey.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Encode certificate if present
	var certB64 string
	if w.localAgency.Certificate != nil {
		certB64 = base64.StdEncoding.EncodeToString(w.localAgency.Certificate.Raw)
	}

	return &AgencySignature{
		AgencyCode:  w.localAgency.AgencyCode,
		AgencyName:  w.localAgency.AgencyName,
		Signature:   base64.StdEncoding.EncodeToString(signature),
		Certificate: certB64,
		SignedAt:    time.Now().UTC(),
	}, nil
}

// requestSignature requests a signature from a remote agency.
func (w *MultiAgencyWitness) requestSignature(ctx context.Context, agency *AgencyWitnessConfig, request *WitnessRequest) (*AgencySignature, error) {
	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/witness/sign", agency.EndpointURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agency returned status %d", resp.StatusCode)
	}

	// Parse response
	var sig AgencySignature
	if err := json.NewDecoder(resp.Body).Decode(&sig); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Verify the signature
	if err := w.verifySignature(&sig, request, agency); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &sig, nil
}

// VerifyProof verifies that a multi-agency proof is valid.
func (w *MultiAgencyWitness) VerifyProof(ctx context.Context, proof *MultiAgencyProof) (*VerifyProofResult, error) {
	result := &VerifyProofResult{
		Valid:             true,
		TotalSignatures:   len(proof.Signatures),
		ValidSignatures:   0,
		InvalidSignatures: 0,
		Details:           make([]SignatureVerifyDetail, 0),
	}

	// Create witness request from proof for verification
	request := &WitnessRequest{
		CheckpointHash: proof.CheckpointHash,
		Timestamp:      proof.CreatedAt,
	}

	// Verify each signature
	for _, sig := range proof.Signatures {
		detail := SignatureVerifyDetail{
			AgencyCode: sig.AgencyCode,
			AgencyName: sig.AgencyName,
			SignedAt:   sig.SignedAt,
		}

		// Find agency config
		var agencyConfig *AgencyWitnessConfig
		for _, ag := range w.config.Agencies {
			if ag.AgencyCode == sig.AgencyCode {
				agencyConfig = &ag
				break
			}
		}

		// Check if it's our local agency
		if agencyConfig == nil && w.localAgency != nil && w.localAgency.AgencyCode == sig.AgencyCode {
			agencyConfig = &AgencyWitnessConfig{
				AgencyCode:  w.localAgency.AgencyCode,
				AgencyName:  w.localAgency.AgencyName,
				Certificate: w.localAgency.Certificate,
			}
		}

		if agencyConfig == nil {
			detail.Valid = false
			detail.Error = "unknown agency"
			result.InvalidSignatures++
		} else if err := w.verifySignature(&sig, request, agencyConfig); err != nil {
			detail.Valid = false
			detail.Error = err.Error()
			result.InvalidSignatures++
		} else {
			detail.Valid = true
			result.ValidSignatures++
		}

		result.Details = append(result.Details, detail)
	}

	// Check if we have enough valid signatures
	result.Valid = result.ValidSignatures >= proof.MinRequired
	if !result.Valid {
		result.Message = fmt.Sprintf("insufficient valid signatures: got %d, need %d",
			result.ValidSignatures, proof.MinRequired)
	} else {
		result.Message = fmt.Sprintf("proof verified with %d/%d valid signatures",
			result.ValidSignatures, result.TotalSignatures)
	}

	return result, nil
}

// verifySignature verifies a single agency signature.
func (w *MultiAgencyWitness) verifySignature(sig *AgencySignature, request *WitnessRequest, agency *AgencyWitnessConfig) error {
	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Get certificate for verification
	var cert *x509.Certificate
	if sig.Certificate != "" {
		certDER, err := base64.StdEncoding.DecodeString(sig.Certificate)
		if err != nil {
			return fmt.Errorf("failed to decode certificate: %w", err)
		}
		cert, err = x509.ParseCertificate(certDER)
		if err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}
	} else if agency.Certificate != nil {
		cert = agency.Certificate
	} else {
		return fmt.Errorf("no certificate available for verification")
	}

	// Create the data that was signed
	dataToSign := w.createSignatureData(request)
	hash := sha256.Sum256(dataToSign)

	// Verify based on key type
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hash[:], sigBytes)
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, hash[:], sigBytes) {
			err = fmt.Errorf("ECDSA signature verification failed")
		}
	case ed25519.PublicKey:
		if !ed25519.Verify(pub, dataToSign, sigBytes) {
			err = fmt.Errorf("Ed25519 signature verification failed")
		}
	default:
		err = fmt.Errorf("unsupported key type: %T", pub)
	}

	return err
}

// createSignatureData creates the canonical data to be signed.
func (w *MultiAgencyWitness) createSignatureData(request *WitnessRequest) []byte {
	// Create deterministic JSON
	data := map[string]interface{}{
		"checkpoint_hash": request.CheckpointHash,
		"last_sequence":   request.LastSequence,
		"entry_count":     request.EntryCount,
		"timestamp":       request.Timestamp.UTC().Format(time.RFC3339Nano),
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build canonical string
	var buf bytes.Buffer
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('|')
		}
		buf.WriteString(k)
		buf.WriteByte(':')
		buf.WriteString(fmt.Sprintf("%v", data[k]))
	}

	return buf.Bytes()
}

// HandleSignRequest handles incoming witness sign requests from other agencies.
// This should be exposed as an HTTP endpoint.
func (w *MultiAgencyWitness) HandleSignRequest(ctx context.Context, request *WitnessRequest) (*AgencySignature, error) {
	// Validate request
	if request.CheckpointHash == "" {
		return nil, fmt.Errorf("checkpoint_hash is required")
	}
	if request.RequestingAgency == "" {
		return nil, fmt.Errorf("requesting_agency is required")
	}

	// Verify the requesting agency is known
	known := false
	for _, ag := range w.config.Agencies {
		if ag.AgencyCode == request.RequestingAgency {
			known = true
			break
		}
	}
	if !known {
		return nil, fmt.Errorf("unknown requesting agency: %s", request.RequestingAgency)
	}

	// Sign the request
	return w.signLocally(request)
}

// Serialize serializes the proof for storage.
func (p *MultiAgencyProof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// DeserializeProof deserializes a proof from storage.
func DeserializeProof(data []byte) (*MultiAgencyProof, error) {
	var proof MultiAgencyProof
	if err := json.Unmarshal(data, &proof); err != nil {
		return nil, err
	}
	return &proof, nil
}

// Hash returns the SHA-256 hash of the proof for verification.
func (p *MultiAgencyProof) Hash() string {
	data, _ := json.Marshal(p)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// VerifyProofResult contains the result of proof verification.
type VerifyProofResult struct {
	Valid             bool                    `json:"valid"`
	Message           string                  `json:"message"`
	TotalSignatures   int                     `json:"total_signatures"`
	ValidSignatures   int                     `json:"valid_signatures"`
	InvalidSignatures int                     `json:"invalid_signatures"`
	Details           []SignatureVerifyDetail `json:"details"`
}

// SignatureVerifyDetail contains verification details for a single signature.
type SignatureVerifyDetail struct {
	AgencyCode string    `json:"agency_code"`
	AgencyName string    `json:"agency_name"`
	SignedAt   time.Time `json:"signed_at"`
	Valid      bool      `json:"valid"`
	Error      string    `json:"error,omitempty"`
}
