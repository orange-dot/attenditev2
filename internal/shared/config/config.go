package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	KurrentDB  KurrentDBConfig
	Auth       AuthConfig
	OPA        OPAConfig
	AI         AIConfig
	Privacy    PrivacyConfig
	TSA        TSAConfig
}

// TSAConfig holds configuration for the Time Stamping Authority.
type TSAConfig struct {
	// Enabled controls whether TSA is active
	Enabled bool
	// WitnessType: "local", "rfc3161_tsa", "multi_agency", "composite"
	WitnessType string
	// OrgName for self-signed TSA certificate (development)
	OrgName string
	// CertPath for production TSA certificate
	CertPath string
	// KeyPath for production TSA private key
	KeyPath string
	// MultiAgencyEnabled enables multi-agency witness
	MultiAgencyEnabled bool
	// MultiAgencyMinSignatures is the minimum signatures required
	MultiAgencyMinSignatures int
}

type AIConfig struct {
	URL     string
	Enabled bool
}

type OPAConfig struct {
	URL     string
	Enabled bool
}

// PrivacyConfig holds configuration for the privacy module.
type PrivacyConfig struct {
	// FacilityType: "local" for local facilities, "central" for central system
	FacilityType string
	// FacilityCode: Unique identifier for the facility (e.g., "CSR-KG-001")
	FacilityCode string
	// HMACKeyPath: Path to the HMAC key file (or HSM config in production)
	HMACKeyPath string
	// HMACKey: Direct key value (for testing only, use HMACKeyPath in production)
	HMACKey string

	// Privacy Guard settings
	EnablePrivacyGuard bool
	BlockOnViolation   bool
	LogViolations      bool
	ExemptPaths        []string
	ExemptPrefixes     []string

	// De-pseudonymization settings
	TokenTTLHours        int
	MaxTokenUses         int
	ApprovalTimeoutHours int

	// AI Access settings
	DefaultAIAccessLevel int // 0=aggregated, 1=pseudonymized, 2=linkable
	AIAccessTTLHours     int
	MaxRecordsLevel1     int
	MaxRecordsLevel2     int
}

// KurrentDBConfig holds configuration for KurrentDB (EventStoreDB).
type KurrentDBConfig struct {
	// Host is the KurrentDB server hostname
	Host string
	// Port is the gRPC/HTTP port (default 2113)
	Port int
	// Insecure disables TLS (for development)
	Insecure bool
	// Username for authentication (optional)
	Username string
	// Password for authentication (optional)
	Password string
}

type ServerConfig struct {
	Port int
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode,
	)
}


type AuthConfig struct {
	KeycloakURL   string
	Realm         string
	ClientID      string
	ClientSecret  string
	JWTSecret     string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port: getEnvInt("SERVER_PORT", 8080),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "platform"),
			Password: getEnv("DB_PASSWORD", "platform"),
			Database: getEnv("DB_NAME", "platform"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		KurrentDB: KurrentDBConfig{
			Host:     getEnv("KURRENTDB_HOST", "localhost"),
			Port:     getEnvInt("KURRENTDB_PORT", 2113),
			Insecure: getEnvBool("KURRENTDB_INSECURE", true),
			Username: getEnv("KURRENTDB_USERNAME", ""),
			Password: getEnv("KURRENTDB_PASSWORD", ""),
		},
		Auth: AuthConfig{
			KeycloakURL:  getEnv("KEYCLOAK_URL", "http://localhost:8180"),
			Realm:        getEnv("KEYCLOAK_REALM", "serbia-gov"),
			ClientID:     getEnv("KEYCLOAK_CLIENT_ID", "platform"),
			ClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
			JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-in-prod"),
		},
		OPA: OPAConfig{
			URL:     getEnv("OPA_URL", "http://localhost:8181"),
			Enabled: getEnvBool("OPA_ENABLED", false),
		},
		AI: AIConfig{
			URL:     getEnv("AI_SERVICE_URL", "http://localhost:5000"),
			Enabled: getEnvBool("AI_ENABLED", true),
		},
		Privacy: PrivacyConfig{
			FacilityType:         getEnv("PRIVACY_FACILITY_TYPE", "local"),
			FacilityCode:         getEnv("PRIVACY_FACILITY_CODE", "LOCAL-001"),
			HMACKeyPath:          getEnv("PRIVACY_HMAC_KEY_PATH", ""),
			HMACKey:              getEnv("PRIVACY_HMAC_KEY", "dev-hmac-key-change-in-production"),
			EnablePrivacyGuard:   getEnvBool("PRIVACY_GUARD_ENABLED", true),
			BlockOnViolation:     getEnvBool("PRIVACY_BLOCK_VIOLATIONS", true),
			LogViolations:        getEnvBool("PRIVACY_LOG_VIOLATIONS", true),
			ExemptPaths:          getEnvSlice("PRIVACY_EXEMPT_PATHS", []string{"/health", "/ready", "/metrics"}),
			ExemptPrefixes:       getEnvSlice("PRIVACY_EXEMPT_PREFIXES", []string{"/internal/", "/local/"}),
			TokenTTLHours:        getEnvInt("PRIVACY_TOKEN_TTL_HOURS", 1),
			MaxTokenUses:         getEnvInt("PRIVACY_MAX_TOKEN_USES", 3),
			ApprovalTimeoutHours: getEnvInt("PRIVACY_APPROVAL_TIMEOUT_HOURS", 24),
			DefaultAIAccessLevel: getEnvInt("PRIVACY_DEFAULT_AI_ACCESS_LEVEL", 0),
			AIAccessTTLHours:     getEnvInt("PRIVACY_AI_ACCESS_TTL_HOURS", 24),
			MaxRecordsLevel1:     getEnvInt("PRIVACY_MAX_RECORDS_LEVEL1", 1000),
			MaxRecordsLevel2:     getEnvInt("PRIVACY_MAX_RECORDS_LEVEL2", 100),
		},
		TSA: TSAConfig{
			Enabled:                  getEnvBool("TSA_ENABLED", true),
			WitnessType:              getEnv("TSA_WITNESS_TYPE", "local"), // local, rfc3161_tsa, multi_agency, composite
			OrgName:                  getEnv("TSA_ORG_NAME", "Serbia Government Platform"),
			CertPath:                 getEnv("TSA_CERT_PATH", ""),
			KeyPath:                  getEnv("TSA_KEY_PATH", ""),
			MultiAgencyEnabled:       getEnvBool("TSA_MULTI_AGENCY_ENABLED", false),
			MultiAgencyMinSignatures: getEnvInt("TSA_MULTI_AGENCY_MIN_SIGNATURES", 2),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Parse comma-separated values
		var result []string
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
