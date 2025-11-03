package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebRequestTool_GetInfo(t *testing.T) {
	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	info := tool.GetInfo()
	if info.Name != "web_request" {
		t.Errorf("Expected name 'web_request', got '%s'", info.Name)
	}
	if len(info.Parameters) != 4 {
		t.Errorf("Expected 4 parameters, got %d", len(info.Parameters))
	}
}

func TestWebRequestTool_Execute_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.Content != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", result.Content)
	}
	if result.Metadata["status_code"] != http.StatusOK {
		t.Errorf("Expected status code 200, got %v", result.Metadata["status_code"])
	}
}

func TestWebRequestTool_Execute_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("Created"))
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body":   `{"key":"value"}`,
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
	})

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Success {
		t.Error("Expected success=true")
	}
	if result.Content != "Created" {
		t.Errorf("Expected 'Created', got '%s'", result.Content)
	}
}

func TestWebRequestTool_DomainAllowList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		AllowedDomains:  []string{"allowed.com"},
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err == nil {
		t.Error("Expected error for disallowed domain")
	}
	if result.Success {
		t.Error("Expected success=false for disallowed domain")
	}
}

func TestWebRequestTool_DomainDenyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		DeniedDomains:   []string{"127.0.0.1", "localhost"},
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err == nil {
		t.Error("Expected error for denied domain")
	}
	if result.Success {
		t.Error("Expected success=false for denied domain")
	}
}

func TestWebRequestTool_MethodRestriction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 52428800,
		AllowedMethods:  []string{"GET", "POST"},
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	// Allowed method
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
	})
	if err != nil {
		t.Errorf("GET should be allowed: %v", err)
	}
	if !result.Success {
		t.Error("Expected success=true for GET")
	}

	// Disallowed method
	result, err = tool.Execute(context.Background(), map[string]interface{}{
		"url":    server.URL,
		"method": "DELETE",
	})
	if err == nil {
		t.Error("Expected error for disallowed method DELETE")
	}
	if result.Success {
		t.Error("Expected success=false for disallowed method")
	}
}

func TestWebRequestTool_MaxResponseSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send response larger than max size
		w.WriteHeader(http.StatusOK)
		data := make([]byte, 1024*1024) // 1MB
		_, _ = w.Write(data)
	}))
	defer server.Close()

	tool := NewWebRequestTool(&WebRequestConfig{
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		MaxRequestSize:  10485760,
		MaxResponseSize: 1024, // Only 1KB
		AllowRedirects:  true,
		MaxRedirects:    10,
		UserAgent:       "Hector-Agent/1.0",
	})

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": server.URL,
	})

	if err == nil {
		t.Error("Expected error for response exceeding max size")
	}
	if result.Success {
		t.Error("Expected success=false for oversized response")
	}
}

func TestWebRequestTool_WildcardDomain(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		host     string
		expected bool
	}{
		{"Exact match", "example.com", "example.com", true},
		{"Wildcard subdomain match", "*.example.com", "api.example.com", true},
		{"Wildcard subdomain match deep", "*.example.com", "sub.api.example.com", true},
		{"No match", "*.example.com", "other.com", false},
		{"Port stripped", "example.com", "example.com:8080", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesDomain(tt.host, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesDomain(%s, %s) = %v, want %v",
					tt.host, tt.pattern, result, tt.expected)
			}
		})
	}
}
