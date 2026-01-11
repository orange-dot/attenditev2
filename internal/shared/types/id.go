package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// ID is a UUID wrapper for type safety
type ID string

// NewID generates a new random ID
func NewID() ID {
	return ID(uuid.New().String())
}

// NewDeterministicID generates a deterministic ID based on namespace and name
// This creates the same UUID for the same namespace+name combination
func NewDeterministicID(namespace, name string) ID {
	// Use UUID v5 (SHA-1 based) for deterministic IDs
	ns := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace as base
	return ID(uuid.NewSHA1(ns, []byte(namespace+":"+name)).String())
}

// ParseID parses a string into an ID
func ParseID(s string) (ID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return "", fmt.Errorf("invalid ID: %w", err)
	}
	return ID(s), nil
}

// MustParseID parses a string into an ID, panics on error
func MustParseID(s string) ID {
	id, err := ParseID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation
func (id ID) String() string {
	return string(id)
}

// IsZero checks if the ID is empty
func (id ID) IsZero() bool {
	return id == ""
}

// Value implements driver.Valuer for database serialization
func (id ID) Value() (driver.Value, error) {
	if id.IsZero() {
		return nil, nil
	}
	return string(id), nil
}

// Scan implements sql.Scanner for database deserialization
func (id *ID) Scan(value interface{}) error {
	if value == nil {
		*id = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*id = ID(v)
	case []byte:
		*id = ID(string(v))
	default:
		return fmt.Errorf("cannot scan %T into ID", value)
	}
	return nil
}
