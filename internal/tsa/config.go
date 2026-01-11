// Package tsa provides an internal RFC 3161 Time Stamping Authority.
// This allows the platform to issue legally valid timestamps without
// relying on external services like OpenTimestamps or Bitcoin.
package tsa

import (
	"crypto"
	"crypto/x509"
)

// Config holds TSA server configuration.
type Config struct {
	// Enabled controls whether the TSA is active
	Enabled bool

	// PolicyOID is the timestamp policy OID (e.g., "1.2.3.4.1")
	// This identifies the policy under which timestamps are issued
	PolicyOID string

	// Certificate is the TSA signing certificate
	Certificate *x509.Certificate

	// CertificateChain is the full certificate chain for verification
	CertificateChain []*x509.Certificate

	// PrivateKey is the TSA private key for signing
	// In production, this should come from an HSM
	PrivateKey crypto.Signer

	// HashAlgorithm for timestamp tokens (default: SHA-256)
	HashAlgorithm crypto.Hash

	// AccuracySeconds defines the claimed accuracy of timestamps
	AccuracySeconds int

	// IncludeCertificate includes signing cert in response
	IncludeCertificate bool
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		PolicyOID:          "1.3.6.1.4.1.99999.1.1", // Custom OID for Serbia Gov
		HashAlgorithm:      crypto.SHA256,
		AccuracySeconds:    1,
		IncludeCertificate: true,
	}
}

// MultiAgencyConfig holds configuration for multi-agency witnessing.
type MultiAgencyConfig struct {
	// Enabled controls whether multi-agency witnessing is active
	Enabled bool

	// MinSignatures is the minimum number of agency signatures required
	MinSignatures int

	// Agencies is the list of participating agencies
	Agencies []AgencyWitnessConfig
}

// AgencyWitnessConfig holds configuration for a single agency witness.
type AgencyWitnessConfig struct {
	// AgencyCode is the unique agency identifier
	AgencyCode string

	// AgencyName is the human-readable agency name
	AgencyName string

	// EndpointURL is the agency's witness endpoint
	EndpointURL string

	// PublicKey is the agency's public key for signature verification
	PublicKey crypto.PublicKey

	// Certificate is the agency's signing certificate
	Certificate *x509.Certificate
}
