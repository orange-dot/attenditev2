package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/serbia-gov/platform/internal/shared/config"
	"github.com/serbia-gov/platform/internal/shared/types"
)

// Decision represents an authorization decision from OPA
type Decision struct {
	Allow   bool              `json:"allow"`
	Reasons []string          `json:"reasons,omitempty"`
	Fields  map[string]bool   `json:"fields,omitempty"` // Field-level access
}

// Input represents the input to an OPA policy evaluation
type Input struct {
	// Actor
	ActorID       types.ID `json:"actor_id"`
	ActorType     string   `json:"actor_type"` // citizen, worker, system
	ActorAgencyID types.ID `json:"actor_agency_id,omitempty"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`

	// Action
	Action       string `json:"action"`       // read, write, delete, share, etc.
	ResourceType string `json:"resource_type"` // case, document, agency, etc.
	ResourceID   types.ID `json:"resource_id,omitempty"`

	// Resource context
	Resource map[string]any `json:"resource,omitempty"`

	// Request context
	RequestIP     string            `json:"request_ip,omitempty"`
	RequestMethod string            `json:"request_method,omitempty"`
	RequestPath   string            `json:"request_path,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// Client provides OPA policy evaluation
type Client struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewClient creates a new OPA client
func NewClient(cfg config.OPAConfig) *Client {
	return &Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		enabled: cfg.Enabled,
	}
}

// Evaluate evaluates a policy and returns a decision
func (c *Client) Evaluate(ctx context.Context, policy string, input Input) (*Decision, error) {
	if !c.enabled {
		// OPA disabled - allow all
		return &Decision{Allow: true}, nil
	}

	// Prepare request body
	reqBody := map[string]any{
		"input": input,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, policy)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// OPA unavailable - fail closed (deny)
		return &Decision{
			Allow:   false,
			Reasons: []string{"policy engine unavailable"},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &Decision{
			Allow:   false,
			Reasons: []string{fmt.Sprintf("policy evaluation failed: %d", resp.StatusCode)},
		}, nil
	}

	// Parse response
	var result struct {
		Result Decision `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Result, nil
}

// CheckAccess is a convenience method for common access checks
func (c *Client) CheckAccess(ctx context.Context, input Input) (bool, error) {
	// Determine policy based on resource type
	policy := fmt.Sprintf("platform/%s/allow", input.ResourceType)

	decision, err := c.Evaluate(ctx, policy, input)
	if err != nil {
		return false, err
	}

	return decision.Allow, nil
}

// CheckCaseAccess checks if actor can access a case
func (c *Client) CheckCaseAccess(ctx context.Context, actorID, actorAgencyID, caseID types.ID, action string, roles []string, caseData map[string]any) (bool, error) {
	input := Input{
		ActorID:       actorID,
		ActorType:     "worker",
		ActorAgencyID: actorAgencyID,
		Roles:         roles,
		Action:        action,
		ResourceType:  "case",
		ResourceID:    caseID,
		Resource:      caseData,
	}

	return c.CheckAccess(ctx, input)
}

// CheckDocumentAccess checks if actor can access a document
func (c *Client) CheckDocumentAccess(ctx context.Context, actorID, actorAgencyID, documentID types.ID, action string, roles []string, docData map[string]any) (bool, error) {
	input := Input{
		ActorID:       actorID,
		ActorType:     "worker",
		ActorAgencyID: actorAgencyID,
		Roles:         roles,
		Action:        action,
		ResourceType:  "document",
		ResourceID:    documentID,
		Resource:      docData,
	}

	return c.CheckAccess(ctx, input)
}

// Health checks OPA connection
func (c *Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OPA health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
