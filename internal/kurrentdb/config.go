package kurrentdb

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds KurrentDB connection configuration.
type Config struct {
	// Host is the KurrentDB server hostname
	Host string
	// Port is the KurrentDB gRPC/HTTP port (default 2113)
	Port int
	// Insecure disables TLS (for development)
	Insecure bool
	// Username for authentication (optional for insecure mode)
	Username string
	// Password for authentication (optional for insecure mode)
	Password string
}

// ConnectionString returns the esdb:// connection string for EventStore client.
func (c *Config) ConnectionString() string {
	var auth string
	if c.Username != "" && c.Password != "" {
		auth = fmt.Sprintf("%s:%s@", c.Username, c.Password)
	}

	var tls string
	if c.Insecure {
		tls = "?tls=false"
	}

	return fmt.Sprintf("esdb://%s%s:%d%s", auth, c.Host, c.Port, tls)
}

// LoadConfig loads KurrentDB configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Host:     getEnv("KURRENTDB_HOST", "localhost"),
		Port:     getEnvInt("KURRENTDB_PORT", 2113),
		Insecure: getEnvBool("KURRENTDB_INSECURE", true),
		Username: getEnv("KURRENTDB_USERNAME", ""),
		Password: getEnv("KURRENTDB_PASSWORD", ""),
	}
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
