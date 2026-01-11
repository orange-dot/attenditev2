package kurrentdb

import (
	"github.com/google/uuid"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// parseEventID converts a UUID to types.ID string.
func parseEventID(id uuid.UUID) types.ID {
	return types.ID(id.String())
}

// toUUID converts a types.ID to uuid.UUID.
func toUUID(id types.ID) uuid.UUID {
	parsed, err := uuid.Parse(string(id))
	if err != nil {
		// Generate a new UUID if parsing fails
		return uuid.New()
	}
	return parsed
}
