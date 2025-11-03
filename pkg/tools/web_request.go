package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

type WebRequestTool struct {
	config     *WebRequestConfig
	httpClient *httpclient.Client
}

type WebRequestConfig struct {
	Timeout         time.Duration
	MaxRetries      int
	MaxRequestSize  int64
	MaxResponseSize int64
	AllowedDomains  []string
	DeniedDomains   []string
	AllowedMethods  []string
	AllowRedirects  bool
	MaxRedirects    int
	UserAgent       string
}

func NewWebRequestTool(cfg *WebRequestConfig) *WebRequestTool {
	httpClientConfig := &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !cfg.AllowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", cfg.MaxRedirects)
			}
			return nil
		},
	}

	hc := httpclient.New(
		httpclient.WithHTTPClient(httpClientConfig),
		httpclient.WithMaxRetries(cfg.MaxRetries),
	)

	return &WebRequestTool{
		config:     cfg,
		httpClient: hc,
	}
}

func NewWebRequestToolWithConfig(toolName string, toolConfig *config.ToolConfig) (Tool, error) {
	timeout, err := time.ParseDuration(toolConfig.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	allowRedirects := true
	if toolConfig.AllowRedirects != nil {
		allowRedirects = *toolConfig.AllowRedirects
	}

	cfg := &WebRequestConfig{
		Timeout:         timeout,
		MaxRetries:      toolConfig.MaxRetries,
		MaxRequestSize:  toolConfig.MaxRequestSize,
		MaxResponseSize: toolConfig.MaxResponseSize,
		AllowedDomains:  toolConfig.AllowedDomains,
		DeniedDomains:   toolConfig.DeniedDomains,
		AllowedMethods:  toolConfig.AllowedMethods,
		AllowRedirects:  allowRedirects,
		MaxRedirects:    toolConfig.MaxRedirects,
		UserAgent:       toolConfig.UserAgent,
	}

	return NewWebRequestTool(cfg), nil
}

func (t *WebRequestTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Extract URL
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return t.errorResult("url parameter is required", start), fmt.Errorf("url parameter is required")
	}

	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return t.errorResult(fmt.Sprintf("invalid URL: %v", err), start), err
	}

	if err := t.validateDomain(parsedURL.Host); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	// Extract HTTP method (default: GET)
	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	if err := t.validateMethod(method); err != nil {
		return t.errorResult(err.Error(), start), err
	}

	// Extract headers
	headers := make(map[string]string)
	if h, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range h {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	// Extract body
	var body io.Reader
	if bodyData, ok := args["body"]; ok {
		var bodyBytes []byte
		switch v := bodyData.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			return t.errorResult("body must be string or bytes", start), fmt.Errorf("invalid body type")
		}

		if int64(len(bodyBytes)) > t.config.MaxRequestSize {
			return t.errorResult(
				fmt.Sprintf("request body too large: %d bytes (max: %d)",
					len(bodyBytes), t.config.MaxRequestSize),
				start), fmt.Errorf("request body exceeds max size")
		}
		body = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to create request: %v", err), start), err
	}

	// Set headers
	req.Header.Set("User-Agent", t.config.UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return t.errorResult(fmt.Sprintf("request failed: %v", err), start), err
	}
	defer resp.Body.Close()

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, t.config.MaxResponseSize+1)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return t.errorResult(fmt.Sprintf("failed to read response: %v", err), start), err
	}

	if int64(len(responseBody)) > t.config.MaxResponseSize {
		return t.errorResult(
			fmt.Sprintf("response too large: exceeds %d bytes", t.config.MaxResponseSize),
			start), fmt.Errorf("response exceeds max size")
	}

	// Build response headers map
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	executionTime := time.Since(start)

	return ToolResult{
		Success:       resp.StatusCode >= 200 && resp.StatusCode < 300,
		Content:       string(responseBody),
		ToolName:      "web_request",
		ExecutionTime: executionTime,
		Metadata: map[string]interface{}{
			"url":          urlStr,
			"method":       method,
			"status_code":  resp.StatusCode,
			"status":       resp.Status,
			"headers":      respHeaders,
			"content_type": resp.Header.Get("Content-Type"),
			"size":         len(responseBody),
		},
	}, nil
}

func (t *WebRequestTool) validateDomain(host string) error {
	// If no domain restrictions, allow all
	if len(t.config.AllowedDomains) == 0 && len(t.config.DeniedDomains) == 0 {
		return nil
	}

	// Check denied list first (takes precedence)
	for _, denied := range t.config.DeniedDomains {
		if matchesDomain(host, denied) {
			return fmt.Errorf("domain not allowed: %s (matches deny rule: %s)", host, denied)
		}
	}

	// If allowed list is specified, check it
	if len(t.config.AllowedDomains) > 0 {
		for _, allowed := range t.config.AllowedDomains {
			if matchesDomain(host, allowed) {
				return nil
			}
		}
		return fmt.Errorf("domain not allowed: %s (not in allowed list)", host)
	}

	return nil
}

func (t *WebRequestTool) validateMethod(method string) error {
	// If no method restrictions, allow all
	if len(t.config.AllowedMethods) == 0 {
		return nil
	}

	for _, allowed := range t.config.AllowedMethods {
		if strings.EqualFold(method, allowed) {
			return nil
		}
	}

	return fmt.Errorf("HTTP method not allowed: %s (allowed: %v)", method, t.config.AllowedMethods)
}

func matchesDomain(host, pattern string) bool {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Exact match
	if host == pattern {
		return true
	}

	// Wildcard match (e.g., "*.example.com")
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove '*'
		return strings.HasSuffix(host, suffix)
	}

	return false
}

func (t *WebRequestTool) errorResult(message string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         message,
		ToolName:      "web_request",
		ExecutionTime: time.Since(start),
	}
}

func (t *WebRequestTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "web_request",
		Description: "Make HTTP requests to external APIs and web services",
		Parameters: []ToolParameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to request",
				Required:    true,
			},
			{
				Name:        "method",
				Type:        "string",
				Description: "HTTP method (GET, POST, PUT, DELETE, etc.). Default: GET",
				Required:    false,
				Enum:        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
			},
			{
				Name:        "headers",
				Type:        "object",
				Description: "HTTP headers as key-value pairs",
				Required:    false,
			},
			{
				Name:        "body",
				Type:        "string",
				Description: "Request body (for POST, PUT, PATCH)",
				Required:    false,
			},
		},
		ServerURL: "local",
	}
}

func (t *WebRequestTool) GetName() string {
	return "web_request"
}

func (t *WebRequestTool) GetDescription() string {
	return "Make HTTP requests to external APIs and web services. Supports all HTTP methods, custom headers, and request bodies."
}
