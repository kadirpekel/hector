package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LakeraConfig configures the Lakera Guard provider.
type LakeraConfig struct {
	// APIKey is the Lakera API key.
	// Uses LAKERA_API_KEY environment variable if empty.
	APIKey string

	// ProjectID is the Lakera project ID for policy configuration.
	// If empty, uses the default Lakera Guard policy.
	ProjectID string

	// Endpoint is the API endpoint URL.
	// Default: https://api.lakera.ai/v2/guard
	Endpoint string

	// Breakdown enables detailed detector breakdown in response.
	Breakdown bool

	// Timeout is the request timeout.
	// Default: 10s
	Timeout time.Duration
}

// lakeraProvider implements Provider using Lakera Guard v2 API.
type lakeraProvider struct {
	apiKey    string
	projectID string
	endpoint  string
	breakdown bool
	client    *http.Client
}

// NewLakera creates a Lakera Guard moderation provider.
// Lakera specializes in AI security, prompt injection, and jailbreak detection.
// Uses the v2 Guard API endpoint.
func NewLakera(cfg LakeraConfig) Provider {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("LAKERA_API_KEY")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.lakera.ai/v2/guard"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &lakeraProvider{
		apiKey:    apiKey,
		projectID: cfg.ProjectID,
		endpoint:  endpoint,
		breakdown: cfg.Breakdown,
		client:    &http.Client{Timeout: timeout},
	}
}

func (p *lakeraProvider) Name() string {
	return "lakera"
}

func (p *lakeraProvider) Moderate(ctx context.Context, content string) (*Result, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("lakera moderation: API key not configured")
	}

	// Build request per Lakera v2 Guard API spec
	guardReq := lakeraGuardRequest{
		Messages: []lakeraMessage{
			{
				Content: content,
				Role:    "user",
			},
		},
		Breakdown: p.breakdown,
	}

	if p.projectID != "" {
		guardReq.ProjectID = p.projectID
	}

	reqBody, _ := json.Marshal(guardReq)
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("lakera moderation: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lakera moderation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lakera moderation: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result lakeraGuardResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("lakera moderation: parse response: %w", err)
	}

	// Determine category from breakdown if available
	category := ""
	scores := make(map[string]float64)

	if len(result.Breakdown) > 0 {
		for _, item := range result.Breakdown {
			if item.Detected {
				category = item.DetectorType
				scores[item.DetectorType] = 1.0
			} else {
				scores[item.DetectorType] = 0.0
			}
		}
	}

	reason := ""
	if result.Flagged {
		reason = "Lakera Guard: content flagged"
		if category != "" {
			reason = fmt.Sprintf("Lakera Guard: %s detected", category)
		}
	}

	return &Result{
		Safe:       !result.Flagged,
		Category:   category,
		Confidence: boolToFloat(result.Flagged),
		Reason:     reason,
		Scores:     scores,
	}, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// Lakera v2 Guard API request/response types per official documentation

type lakeraMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"` // system, user, assistant, tool, developer
}

type lakeraGuardRequest struct {
	Messages  []lakeraMessage `json:"messages"`
	ProjectID string          `json:"project_id,omitempty"`
	Payload   bool            `json:"payload,omitempty"`
	Breakdown bool            `json:"breakdown,omitempty"`
	DevInfo   bool            `json:"dev_info,omitempty"`
}

type lakeraGuardResponse struct {
	Flagged   bool                    `json:"flagged"`
	Payload   []lakeraPayloadItem     `json:"payload,omitempty"`
	Breakdown []lakeraBreakdownItem   `json:"breakdown,omitempty"`
	DevInfo   *lakeraDevInfo          `json:"dev_info,omitempty"`
	Metadata  *lakeraResponseMetadata `json:"metadata,omitempty"`
}

type lakeraPayloadItem struct {
	Start        int      `json:"start"`
	End          int      `json:"end"`
	Text         string   `json:"text"`
	DetectorType string   `json:"detector_type"`
	Labels       []string `json:"labels"`
	MessageID    int      `json:"message_id"`
}

type lakeraBreakdownItem struct {
	ProjectID    string `json:"project_id"`
	PolicyID     string `json:"policy_id"`
	DetectorID   string `json:"detector_id"`
	DetectorType string `json:"detector_type"`
	Detected     bool   `json:"detected"`
	MessageID    int    `json:"message_id"`
}

type lakeraDevInfo struct {
	GitRevision  string `json:"git_revision"`
	GitTimestamp string `json:"git_timestamp"`
	ModelVersion string `json:"model_version"`
	Version      string `json:"version"`
}

type lakeraResponseMetadata struct {
	RequestUUID string `json:"request_uuid"`
}
