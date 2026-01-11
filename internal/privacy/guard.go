package privacy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/serbia-gov/platform/internal/shared/types"
)

// ViolationHandler handles detected PII violations.
type ViolationHandler interface {
	HandleViolation(ctx context.Context, violation *PIIViolation) error
}

// ViolationLogger is a simple violation handler that logs to audit.
type ViolationLogger struct {
	audit AuditLogger
}

// NewViolationLogger creates a new violation logger.
func NewViolationLogger(audit AuditLogger) *ViolationLogger {
	return &ViolationLogger{audit: audit}
}

// HandleViolation logs a PII violation to the audit log.
func (l *ViolationLogger) HandleViolation(ctx context.Context, violation *PIIViolation) error {
	action := AuditActionPIIViolationDetected
	if violation.Blocked {
		action = AuditActionPIIViolationBlocked
	}

	return l.audit.Log(ctx, action, "pii_violation", violation.ID, map[string]any{
		"field":          violation.Field,
		"location":       violation.Location,
		"blocked":        violation.Blocked,
		"masked_value":   violation.MaskedValue,
		"request_path":   violation.RequestPath,
		"request_method": violation.RequestMethod,
	})
}

// PrivacyGuard is middleware that blocks PII from leaving the central system.
type PrivacyGuard struct {
	// Compiled regex patterns for PII detection
	jmbgPattern  *regexp.Regexp
	phonePattern *regexp.Regexp
	emailPattern *regexp.Regexp
	lboPattern   *regexp.Regexp

	// Violation handler
	violationHandler ViolationHandler

	// Exemption paths (e.g., local facility endpoints)
	exemptPaths    []string
	exemptPrefixes []string

	// Configuration
	blockOnViolation bool
	logViolations    bool
}

// PrivacyGuardConfig holds configuration for the privacy guard.
type PrivacyGuardConfig struct {
	ExemptPaths      []string
	ExemptPrefixes   []string
	BlockOnViolation bool
	LogViolations    bool
}

// DefaultPrivacyGuardConfig returns default configuration.
func DefaultPrivacyGuardConfig() PrivacyGuardConfig {
	return PrivacyGuardConfig{
		ExemptPaths:      []string{"/health", "/ready", "/metrics"},
		ExemptPrefixes:   []string{"/internal/", "/local/"},
		BlockOnViolation: true,
		LogViolations:    true,
	}
}

// NewPrivacyGuard creates a new privacy guard middleware.
func NewPrivacyGuard(handler ViolationHandler, cfg PrivacyGuardConfig) *PrivacyGuard {
	return &PrivacyGuard{
		// JMBG: exactly 13 digits (Serbian personal ID)
		jmbgPattern: regexp.MustCompile(`\b\d{13}\b`),

		// Serbian phone patterns: +381... or 0...
		phonePattern: regexp.MustCompile(`\b(?:\+381|0)[\s\-]?\d{2}[\s\-]?\d{3}[\s\-]?\d{3,4}\b`),

		// Email pattern
		emailPattern: regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),

		// LBO (health insurance number): 11 digits
		lboPattern: regexp.MustCompile(`\b\d{11}\b`),

		violationHandler: handler,
		exemptPaths:      cfg.ExemptPaths,
		exemptPrefixes:   cfg.ExemptPrefixes,
		blockOnViolation: cfg.BlockOnViolation,
		logViolations:    cfg.LogViolations,
	}
}

// Middleware returns the HTTP middleware function.
func (g *PrivacyGuard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is exempt
		if g.isExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check request body for PII
		if r.Body != nil && r.ContentLength > 0 {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				// Restore body for handler
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				location := "request_body:" + r.URL.Path
				if violations := g.detectPII(string(bodyBytes), location, r); len(violations) > 0 {
					g.handleViolations(r.Context(), violations)
					if g.blockOnViolation {
						http.Error(w, `{"error":"request contains prohibited personal data","code":"PII_DETECTED"}`, http.StatusBadRequest)
						return
					}
				}
			}
		}

		// Wrap response writer to inspect response
		wrapper := &responseWrapper{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
			guard:          g,
			request:        r,
		}

		next.ServeHTTP(wrapper, r)

		// Check response for PII before sending
		responseBody := wrapper.body.String()
		location := "response_body:" + r.URL.Path

		if violations := g.detectPII(responseBody, location, r); len(violations) > 0 {
			g.handleViolations(r.Context(), violations)

			if g.blockOnViolation {
				// Redact PII from response
				redactedBody := g.redactPII(responseBody)
				w.Header().Set("X-PII-Redacted", "true")
				w.WriteHeader(wrapper.statusCode)
				w.Write([]byte(redactedBody))
				return
			}
		}

		// Write original response
		w.WriteHeader(wrapper.statusCode)
		w.Write(wrapper.body.Bytes())
	})
}

// detectPII scans content for PII patterns.
func (g *PrivacyGuard) detectPII(content, location string, r *http.Request) []PIIViolation {
	var violations []PIIViolation

	// Check for JMBG
	if matches := g.jmbgPattern.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			violations = append(violations, PIIViolation{
				ID:            types.NewID(),
				Timestamp:     time.Now().UTC(),
				Field:         PIIFieldJMBG,
				Location:      location,
				Blocked:       g.blockOnViolation,
				RawValue:      match,
				MaskedValue:   MaskJMBG(match),
				RequestPath:   r.URL.Path,
				RequestMethod: r.Method,
				RequestIP:     getClientIP(r),
			})
		}
	}

	// Check for phone numbers (only if looks like Serbian format)
	if matches := g.phonePattern.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			violations = append(violations, PIIViolation{
				ID:            types.NewID(),
				Timestamp:     time.Now().UTC(),
				Field:         PIIFieldPhone,
				Location:      location,
				Blocked:       g.blockOnViolation,
				RawValue:      match,
				MaskedValue:   MaskPhone(match),
				RequestPath:   r.URL.Path,
				RequestMethod: r.Method,
				RequestIP:     getClientIP(r),
			})
		}
	}

	// Check for emails
	if matches := g.emailPattern.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			violations = append(violations, PIIViolation{
				ID:            types.NewID(),
				Timestamp:     time.Now().UTC(),
				Field:         PIIFieldEmail,
				Location:      location,
				Blocked:       g.blockOnViolation,
				RawValue:      match,
				MaskedValue:   MaskEmail(match),
				RequestPath:   r.URL.Path,
				RequestMethod: r.Method,
				RequestIP:     getClientIP(r),
			})
		}
	}

	// Check for LBO (only if exactly 11 digits, to reduce false positives)
	if matches := g.lboPattern.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			// Skip if it could be a JMBG (13 digits would already be caught)
			violations = append(violations, PIIViolation{
				ID:            types.NewID(),
				Timestamp:     time.Now().UTC(),
				Field:         PIIFieldLBO,
				Location:      location,
				Blocked:       g.blockOnViolation,
				RawValue:      match,
				MaskedValue:   match[:4] + "*******",
				RequestPath:   r.URL.Path,
				RequestMethod: r.Method,
				RequestIP:     getClientIP(r),
			})
		}
	}

	return violations
}

// redactPII replaces PII with redaction markers.
func (g *PrivacyGuard) redactPII(content string) string {
	// Replace JMBG with placeholder
	content = g.jmbgPattern.ReplaceAllString(content, "[REDACTED-JMBG]")

	// Replace phone numbers
	content = g.phonePattern.ReplaceAllString(content, "[REDACTED-PHONE]")

	// Replace emails
	content = g.emailPattern.ReplaceAllString(content, "[REDACTED-EMAIL]")

	// Replace LBO
	content = g.lboPattern.ReplaceAllString(content, "[REDACTED-LBO]")

	return content
}

// isExempt checks if a path is exempt from PII checking.
func (g *PrivacyGuard) isExempt(path string) bool {
	// Check exact paths
	for _, exempt := range g.exemptPaths {
		if path == exempt {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range g.exemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// handleViolations processes detected violations.
func (g *PrivacyGuard) handleViolations(ctx context.Context, violations []PIIViolation) {
	if !g.logViolations || g.violationHandler == nil {
		return
	}

	for i := range violations {
		g.violationHandler.HandleViolation(ctx, &violations[i])
	}
}

// AddExemptPath adds a path to the exemption list.
func (g *PrivacyGuard) AddExemptPath(path string) {
	g.exemptPaths = append(g.exemptPaths, path)
}

// AddExemptPrefix adds a prefix to the exemption list.
func (g *PrivacyGuard) AddExemptPrefix(prefix string) {
	g.exemptPrefixes = append(g.exemptPrefixes, prefix)
}

// responseWrapper intercepts and inspects response body.
type responseWrapper struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	guard      *PrivacyGuard
	request    *http.Request
}

func (w *responseWrapper) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if r.RemoteAddr != "" {
		// Remove port if present
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}

	return ""
}

// ContainsPII checks if a string contains any PII.
func (g *PrivacyGuard) ContainsPII(content string) bool {
	return g.jmbgPattern.MatchString(content) ||
		g.phonePattern.MatchString(content) ||
		g.emailPattern.MatchString(content) ||
		g.lboPattern.MatchString(content)
}

// ScanForPII returns all detected PII types in the content.
func (g *PrivacyGuard) ScanForPII(content string) []PIIField {
	var fields []PIIField

	if g.jmbgPattern.MatchString(content) {
		fields = append(fields, PIIFieldJMBG)
	}
	if g.phonePattern.MatchString(content) {
		fields = append(fields, PIIFieldPhone)
	}
	if g.emailPattern.MatchString(content) {
		fields = append(fields, PIIFieldEmail)
	}
	if g.lboPattern.MatchString(content) {
		fields = append(fields, PIIFieldLBO)
	}

	return fields
}
