// Package auth provides session management types.
package auth

import (
	"time"
)

// SessionConfig defines session parameters from security-model.md.
type SessionConfig struct {
	AccessTokenTTL     time.Duration // 15 minutes
	RefreshTokenTTL    time.Duration // 8 hours
	IdleTimeout        time.Duration // 30 minutes
	AbsoluteTimeout    time.Duration // 12 hours
	MaxConcurrentSessions int        // 3 per user
}

// DefaultSessionConfig returns the default session configuration.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    8 * time.Hour,
		IdleTimeout:        30 * time.Minute,
		AbsoluteTimeout:    12 * time.Hour,
		MaxConcurrentSessions: 3,
	}
}

// Session represents an active user session.
type Session struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	UserType      string    `json:"user_type"` // citizen, worker, admin
	AgencyID      string    `json:"agency_id,omitempty"`
	Roles         []Role    `json:"roles"`
	Permissions   []Permission `json:"permissions"`

	// Authentication details
	EIDVerified   bool      `json:"eid_verified"`
	EIDAssurance  string    `json:"eid_assurance"` // high, highest
	MFAVerified   bool      `json:"mfa_verified"`

	// Timestamps
	CreatedAt     time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	ExpiresAt     time.Time `json:"expires_at"`

	// Client info
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsIdle checks if the session has been idle too long.
func (s *Session) IsIdle(timeout time.Duration) bool {
	return time.Since(s.LastActivityAt) > timeout
}

// JWTClaims represents the JWT token structure from security-model.md.
type JWTClaims struct {
	Subject     string   `json:"sub"`           // user-uuid
	Issuer      string   `json:"iss"`           // keycloak
	Audience    string   `json:"aud"`           // gov-platform
	ExpiresAt   int64    `json:"exp"`
	IssuedAt    int64    `json:"iat"`

	UserType    string   `json:"user_type"`     // worker, citizen, admin
	AgencyID    string   `json:"agency_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`

	EIDVerified  bool   `json:"eid_verified"`
	EIDAssurance string `json:"eid_assurance"`
	SessionID    string `json:"session_id"`
	MFAVerified  bool   `json:"mfa_verified"`
}

// MFAMethod represents supported MFA methods.
type MFAMethod string

const (
	MFATOTP       MFAMethod = "totp"        // Authenticator app
	MFAConsentID  MFAMethod = "consent_id"  // ConsentID push
	MFASMS        MFAMethod = "sms"         // SMS fallback
)

// AuthenticationMethod represents identity sources.
type AuthenticationMethod string

const (
	AuthSerbiaEID   AuthenticationMethod = "serbia_eid"
	AuthAgencyLDAP  AuthenticationMethod = "agency_ldap"
	AuthLocalAccount AuthenticationMethod = "local_account"
	AuthMTLS        AuthenticationMethod = "mtls"
)

// AssuranceLevel represents identity assurance levels.
type AssuranceLevel string

const (
	AssuranceHigh    AssuranceLevel = "high"    // Serbia eID (ConsentID), LDAP+MFA
	AssuranceHighest AssuranceLevel = "highest" // Serbia eID (QES), mTLS
)
