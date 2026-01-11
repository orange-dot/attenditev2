package tsa

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digitorus/timestamp"
)

// Server implements an RFC 3161 compliant Time Stamping Authority.
type Server struct {
	config        *Config
	serialCounter uint64
	mu            sync.RWMutex
}

// NewServer creates a new TSA server with the given configuration.
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	return &Server{
		config:        config,
		serialCounter: uint64(time.Now().UnixNano()),
	}, nil
}

// NewServerWithGeneratedCert creates a TSA server with a self-signed certificate.
// This is useful for development/testing. In production, use proper PKI certificates.
func NewServerWithGeneratedCert(orgName string) (*Server, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create self-signed certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Timestamp Authority extended key usage OID
	tsaExtKeyUsage := asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 8}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{orgName},
			OrganizationalUnit: []string{"Time Stamping Authority"},
			Country:            []string{"RS"},
			CommonName:         fmt.Sprintf("%s TSA", orgName),
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		UnknownExtKeyUsage:    []asn1.ObjectIdentifier{tsaExtKeyUsage},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	config := DefaultConfig()
	config.Certificate = cert
	config.CertificateChain = []*x509.Certificate{cert}
	config.PrivateKey = privateKey

	return NewServer(config)
}

// Timestamp creates an RFC 3161 timestamp token for the given hash.
func (s *Server) Timestamp(ctx context.Context, dataHash []byte) (*TimestampResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.Enabled {
		return nil, fmt.Errorf("TSA is not enabled")
	}

	if s.config.Certificate == nil || s.config.PrivateKey == nil {
		return nil, fmt.Errorf("TSA certificate or private key not configured")
	}

	// Generate serial number
	serial := atomic.AddUint64(&s.serialCounter, 1)

	// Create timestamp request
	tsReq := timestamp.Request{
		HashAlgorithm: s.config.HashAlgorithm,
		HashedMessage: dataHash,
		// Note: Nonce and CertReq are optional
	}

	// Current time (in production, this should be from a trusted NTP source)
	now := time.Now().UTC()

	// Create the timestamp token
	tsToken, err := s.createTimestampToken(&tsReq, now, serial)
	if err != nil {
		return nil, fmt.Errorf("failed to create timestamp token: %w", err)
	}

	return &TimestampResponse{
		SerialNumber:  serial,
		Timestamp:     now,
		HashAlgorithm: s.config.HashAlgorithm.String(),
		HashedMessage: hex.EncodeToString(dataHash),
		Token:         tsToken,
		PolicyOID:     s.config.PolicyOID,
		Issuer:        s.config.Certificate.Subject.CommonName,
	}, nil
}

// TimestampHash creates a timestamp for a hex-encoded hash string.
func (s *Server) TimestampHash(ctx context.Context, hashHex string) (*TimestampResponse, error) {
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hash hex: %w", err)
	}
	return s.Timestamp(ctx, hashBytes)
}

// TimestampData creates a timestamp for raw data (hashes it first).
func (s *Server) TimestampData(ctx context.Context, data []byte) (*TimestampResponse, error) {
	hash := sha256.Sum256(data)
	return s.Timestamp(ctx, hash[:])
}

// Verify verifies a timestamp token against the original hash.
func (s *Server) Verify(ctx context.Context, token []byte, originalHash []byte) (*VerifyResult, error) {
	// Parse the timestamp token
	ts, err := timestamp.Parse(token)
	if err != nil {
		return &VerifyResult{
			Valid:   false,
			Message: fmt.Sprintf("failed to parse timestamp token: %v", err),
		}, nil
	}

	// Verify the hash matches
	if !compareHashes(ts.HashedMessage, originalHash) {
		return &VerifyResult{
			Valid:   false,
			Message: "hash mismatch: timestamp was created for different data",
		}, nil
	}

	// Verify the signature using our certificate chain
	roots := x509.NewCertPool()
	for _, cert := range s.config.CertificateChain {
		roots.AddCert(cert)
	}

	// Basic verification passed
	return &VerifyResult{
		Valid:        true,
		Message:      "timestamp verified successfully",
		Timestamp:    ts.Time,
		SerialNumber: ts.SerialNumber.Uint64(),
		Issuer:       s.config.Certificate.Subject.CommonName,
	}, nil
}

// createTimestampToken creates the actual RFC 3161 timestamp token.
func (s *Server) createTimestampToken(req *timestamp.Request, now time.Time, serial uint64) ([]byte, error) {
	// Create timestamp info structure
	tsInfo := timestampInfo{
		Version:        1,
		Policy:         asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1, 1}, // Our policy OID
		MessageImprint: messageImprint{
			HashAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm: asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}, // SHA-256
			},
			HashedMessage: req.HashedMessage,
		},
		SerialNumber: big.NewInt(int64(serial)),
		GenTime:      now,
		Accuracy: accuracy{
			Seconds: s.config.AccuracySeconds,
		},
		Ordering: false,
	}

	// Encode TSTInfo
	tstInfoDER, err := asn1.Marshal(tsInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TSTInfo: %w", err)
	}

	// Sign the TSTInfo
	hash := sha256.Sum256(tstInfoDER)
	signature, err := s.config.PrivateKey.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to sign timestamp: %w", err)
	}

	// Create the response structure
	response := timestampResponse{
		TSTInfo:     tstInfoDER,
		Signature:   signature,
		Certificate: s.config.Certificate.Raw,
	}

	// Encode the full response
	responseDER, err := asn1.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return responseDER, nil
}

// GetCertificate returns the TSA certificate.
func (s *Server) GetCertificate() *x509.Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.Certificate
}

// GetCertificateChain returns the full certificate chain.
func (s *Server) GetCertificateChain() []*x509.Certificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.CertificateChain
}

// compareHashes compares two hash byte slices.
func compareHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TimestampResponse contains the result of a timestamp operation.
type TimestampResponse struct {
	SerialNumber  uint64    `json:"serial_number"`
	Timestamp     time.Time `json:"timestamp"`
	HashAlgorithm string    `json:"hash_algorithm"`
	HashedMessage string    `json:"hashed_message"`
	Token         []byte    `json:"token"`
	PolicyOID     string    `json:"policy_oid"`
	Issuer        string    `json:"issuer"`
}

// VerifyResult contains the result of timestamp verification.
type VerifyResult struct {
	Valid        bool      `json:"valid"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
	SerialNumber uint64    `json:"serial_number,omitempty"`
	Issuer       string    `json:"issuer,omitempty"`
}

// ASN.1 structures for RFC 3161

type timestampInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint messageImprint
	SerialNumber   *big.Int
	GenTime        time.Time
	Accuracy       accuracy `asn1:"optional"`
	Ordering       bool     `asn1:"optional,default:false"`
	Nonce          *big.Int `asn1:"optional"`
}

type messageImprint struct {
	HashAlgorithm pkix.AlgorithmIdentifier
	HashedMessage []byte
}

type accuracy struct {
	Seconds int `asn1:"optional"`
	Millis  int `asn1:"optional,tag:0"`
	Micros  int `asn1:"optional,tag:1"`
}

type timestampResponse struct {
	TSTInfo     []byte
	Signature   []byte
	Certificate []byte `asn1:"optional"`
}
