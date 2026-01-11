package gateway

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/serbia-gov/platform/internal/federation/trust"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Gateway handles secure cross-agency communication
type Gateway struct {
	agencyID    types.ID
	agencyCode  string
	privateKey  ed25519.PrivateKey
	publicKey   ed25519.PublicKey
	authority   *trust.Authority
	httpClient  *http.Client
}

// Config holds gateway configuration
type Config struct {
	AgencyID   types.ID
	AgencyCode string
	PrivateKey ed25519.PrivateKey
}

// SignedRequest represents a signed cross-agency request
type SignedRequest struct {
	ID            string            `json:"id"`
	Timestamp     time.Time         `json:"timestamp"`
	SourceAgency  string            `json:"source_agency"`
	TargetAgency  string            `json:"target_agency"`
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          []byte            `json:"body,omitempty"`
	Signature     string            `json:"signature"`
	CorrelationID string            `json:"correlation_id,omitempty"`
}

// SignedResponse represents a signed response from a cross-agency request
type SignedResponse struct {
	RequestID    string            `json:"request_id"`
	Timestamp    time.Time         `json:"timestamp"`
	SourceAgency string            `json:"source_agency"`
	StatusCode   int               `json:"status_code"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         []byte            `json:"body,omitempty"`
	Signature    string            `json:"signature"`
}

// NewGateway creates a new agency gateway
func NewGateway(cfg Config, authority *trust.Authority) (*Gateway, error) {
	if cfg.PrivateKey == nil {
		return nil, fmt.Errorf("private key is required")
	}

	publicKey := cfg.PrivateKey.Public().(ed25519.PublicKey)

	return &Gateway{
		agencyID:   cfg.AgencyID,
		agencyCode: cfg.AgencyCode,
		privateKey: cfg.PrivateKey,
		publicKey:  publicKey,
		authority:  authority,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendRequest sends a signed request to another agency
func (g *Gateway) SendRequest(ctx context.Context, targetAgencyCode, method, path string, body []byte) (*SignedResponse, error) {
	// Get target agency info
	targetAgency, err := g.authority.GetAgencyByCode(ctx, targetAgencyCode)
	if err != nil {
		return nil, fmt.Errorf("target agency not found: %w", err)
	}

	if targetAgency.Status != "active" {
		return nil, fmt.Errorf("target agency is not active: %s", targetAgency.Status)
	}

	// Create signed request
	request := &SignedRequest{
		ID:           types.NewID().String(),
		Timestamp:    time.Now().UTC(),
		SourceAgency: g.agencyCode,
		TargetAgency: targetAgencyCode,
		Method:       method,
		Path:         path,
		Body:         body,
	}

	// Sign the request
	if err := g.signRequest(request); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Serialize request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send to target gateway
	url := fmt.Sprintf("%s/federation/gateway/receive", targetAgency.GatewayURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Source-Agency", g.agencyCode)
	httpReq.Header.Set("X-Request-ID", request.ID)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var signedResp SignedResponse
	if err := json.Unmarshal(respBody, &signedResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Verify response signature
	if err := g.verifyResponse(&signedResp, targetAgency.PublicKey); err != nil {
		return nil, fmt.Errorf("failed to verify response signature: %w", err)
	}

	return &signedResp, nil
}

// signRequest signs a request with the agency's private key
func (g *Gateway) signRequest(req *SignedRequest) error {
	// Create canonical representation for signing
	toSign := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		req.ID,
		req.Timestamp.Format(time.RFC3339Nano),
		req.SourceAgency,
		req.TargetAgency,
		req.Method,
		req.Path,
	)

	// Add body hash if present
	if len(req.Body) > 0 {
		bodyHash := sha256.Sum256(req.Body)
		toSign += "|" + base64.StdEncoding.EncodeToString(bodyHash[:])
	}

	// Sign
	signature := ed25519.Sign(g.privateKey, []byte(toSign))
	req.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// VerifyRequest verifies an incoming request signature
func (g *Gateway) VerifyRequest(ctx context.Context, req *SignedRequest) error {
	// Get source agency
	sourceAgency, err := g.authority.GetAgencyByCode(ctx, req.SourceAgency)
	if err != nil {
		return fmt.Errorf("source agency not found: %w", err)
	}

	if sourceAgency.Status != "active" {
		return fmt.Errorf("source agency is not active: %s", sourceAgency.Status)
	}

	// Check timestamp (prevent replay attacks)
	if time.Since(req.Timestamp) > 5*time.Minute {
		return fmt.Errorf("request timestamp too old")
	}

	// Create canonical representation
	toSign := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		req.ID,
		req.Timestamp.Format(time.RFC3339Nano),
		req.SourceAgency,
		req.TargetAgency,
		req.Method,
		req.Path,
	)

	if len(req.Body) > 0 {
		bodyHash := sha256.Sum256(req.Body)
		toSign += "|" + base64.StdEncoding.EncodeToString(bodyHash[:])
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Verify
	if !ed25519.Verify(sourceAgency.PublicKey, []byte(toSign), signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// CreateResponse creates a signed response
func (g *Gateway) CreateResponse(requestID string, statusCode int, body []byte) (*SignedResponse, error) {
	resp := &SignedResponse{
		RequestID:    requestID,
		Timestamp:    time.Now().UTC(),
		SourceAgency: g.agencyCode,
		StatusCode:   statusCode,
		Body:         body,
	}

	// Sign response
	if err := g.signResponse(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// signResponse signs a response with the agency's private key
func (g *Gateway) signResponse(resp *SignedResponse) error {
	toSign := fmt.Sprintf("%s|%s|%s|%d",
		resp.RequestID,
		resp.Timestamp.Format(time.RFC3339Nano),
		resp.SourceAgency,
		resp.StatusCode,
	)

	if len(resp.Body) > 0 {
		bodyHash := sha256.Sum256(resp.Body)
		toSign += "|" + base64.StdEncoding.EncodeToString(bodyHash[:])
	}

	signature := ed25519.Sign(g.privateKey, []byte(toSign))
	resp.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// verifyResponse verifies a response signature
func (g *Gateway) verifyResponse(resp *SignedResponse, publicKey []byte) error {
	toSign := fmt.Sprintf("%s|%s|%s|%d",
		resp.RequestID,
		resp.Timestamp.Format(time.RFC3339Nano),
		resp.SourceAgency,
		resp.StatusCode,
	)

	if len(resp.Body) > 0 {
		bodyHash := sha256.Sum256(resp.Body)
		toSign += "|" + base64.StdEncoding.EncodeToString(bodyHash[:])
	}

	signature, err := base64.StdEncoding.DecodeString(resp.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !ed25519.Verify(publicKey, []byte(toSign), signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}
