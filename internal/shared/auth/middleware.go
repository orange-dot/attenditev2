package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/serbia-gov/platform/internal/shared/config"
	"github.com/serbia-gov/platform/internal/shared/types"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// User represents the authenticated user from JWT claims
type User struct {
	ID            types.ID `json:"sub"`
	UserType      string   `json:"user_type"`      // worker, citizen, admin
	AgencyID      types.ID `json:"agency_id"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	EIDVerified   bool     `json:"eid_verified"`
	EIDAssurance  string   `json:"eid_assurance"`
	SessionID     string   `json:"session_id"`
	MFAVerified   bool     `json:"mfa_verified"`
}

// Claims extends JWT claims with platform-specific data
type Claims struct {
	jwt.RegisteredClaims
	UserType     string   `json:"user_type"`
	AgencyID     string   `json:"agency_id,omitempty"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	EIDVerified  bool     `json:"eid_verified"`
	EIDAssurance string   `json:"eid_assurance,omitempty"`
	SessionID    string   `json:"session_id"`
	MFAVerified  bool     `json:"mfa_verified"`
}

// Middleware creates JWT authentication middleware
func Middleware(cfg config.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				writeError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// For development, use symmetric key
				// In production, use Keycloak's public key
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(*Claims)
			if !ok || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// Build user from claims
			user := &User{
				ID:           types.ID(claims.Subject),
				UserType:     claims.UserType,
				AgencyID:     types.ID(claims.AgencyID),
				Roles:        claims.Roles,
				Permissions:  claims.Permissions,
				EIDVerified:  claims.EIDVerified,
				EIDAssurance: claims.EIDAssurance,
				SessionID:    claims.SessionID,
				MFAVerified:  claims.MFAVerified,
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser extracts the user from request context
func GetUser(ctx context.Context) *User {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// RequireRoles creates middleware that requires specific roles
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			if !hasAnyRole(user.Roles, roles) {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermissions creates middleware that requires specific permissions
func RequirePermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			for _, required := range permissions {
				if !hasPermission(user.Permissions, required) {
					writeError(w, http.StatusForbidden, "insufficient permissions")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role string) bool {
	return hasAnyRole(u.Roles, []string{role})
}

// HasPermission checks if user has a specific permission
func (u *User) HasPermission(permission string) bool {
	return hasPermission(u.Permissions, permission)
}

// IsAdmin checks if user is an admin
func (u *User) IsAdmin() bool {
	return u.UserType == "admin" || u.HasRole("admin") || u.HasRole("platform_admin")
}

func hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, required := range requiredRoles {
		for _, role := range userRoles {
			if role == required {
				return true
			}
		}
	}
	return false
}

func hasPermission(userPermissions []string, required string) bool {
	for _, perm := range userPermissions {
		if perm == required {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
