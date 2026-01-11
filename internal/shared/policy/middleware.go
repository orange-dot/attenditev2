package policy

import (
	"encoding/json"
	"net/http"

	"github.com/serbia-gov/platform/internal/shared/auth"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// contextKey is used for context values
type contextKey string

const (
	// PolicyInputKey is the context key for policy input
	PolicyInputKey contextKey = "policy_input"
)

// Middleware creates an authorization middleware using OPA
func Middleware(client *Client, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.GetUser(r.Context())
			if user == nil {
				// No user context - let auth middleware handle this
				next.ServeHTTP(w, r)
				return
			}

			// Build policy input
			input := Input{
				ActorID:       user.ID,
				ActorType:     user.UserType,
				ActorAgencyID: user.AgencyID,
				Roles:         user.Roles,
				Permissions:   user.Permissions,
				Action:        methodToAction(r.Method),
				ResourceType:  resourceType,
				RequestIP:     r.RemoteAddr,
				RequestMethod: r.Method,
				RequestPath:   r.URL.Path,
			}

			// Check access
			allowed, err := client.CheckAccess(r.Context(), input)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "authorization error")
				return
			}

			if !allowed {
				writeError(w, http.StatusForbidden, "access denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ResourceMiddleware creates middleware that includes resource data in policy check
func ResourceMiddleware(client *Client, resourceType string, getResource func(r *http.Request) (map[string]any, types.ID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.GetUser(r.Context())
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Get resource data
			resource, resourceID, err := getResource(r)
			if err != nil {
				// Resource not found - let handler deal with it
				next.ServeHTTP(w, r)
				return
			}

			// Build policy input
			input := Input{
				ActorID:       user.ID,
				ActorType:     user.UserType,
				ActorAgencyID: user.AgencyID,
				Roles:         user.Roles,
				Permissions:   user.Permissions,
				Action:        methodToAction(r.Method),
				ResourceType:  resourceType,
				ResourceID:    resourceID,
				Resource:      resource,
				RequestIP:     r.RemoteAddr,
				RequestMethod: r.Method,
				RequestPath:   r.URL.Path,
			}

			// Check access
			allowed, err := client.CheckAccess(r.Context(), input)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "authorization error")
				return
			}

			if !allowed {
				writeError(w, http.StatusForbidden, "access denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// methodToAction converts HTTP method to action
func methodToAction(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "unknown"
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
