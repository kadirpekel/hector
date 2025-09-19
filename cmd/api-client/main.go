package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIRequest represents the request structure
type APIRequest struct {
	Query         string `json:"query"`
	VerboseFormat string `json:"verbose_format,omitempty"`
}

// APIResponse represents the response structure
type APIResponse struct {
	Answer        string          `json:"answer"`
	VerboseOutput []VerboseOutput `json:"verbose_output,omitempty"`
	TokensUsed    int             `json:"tokens_used"`
	Success       bool            `json:"success"`
	Error         string          `json:"error,omitempty"`
}

// VerboseOutput represents verbose output structure
type VerboseOutput struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func main() {
	baseURL := "http://localhost:8080"

	// Test different verbose formats
	formats := []string{"terminal", "plain", "json", "html"}

	for _, format := range formats {
		fmt.Printf("\n=== Testing Verbose Format: %s ===\n", format)

		// Create request
		req := APIRequest{
			Query:         "What is artificial intelligence?",
			VerboseFormat: format,
		}

		// Marshal request
		jsonData, err := json.Marshal(req)
		if err != nil {
			fmt.Printf("Error marshaling request: %v\n", err)
			continue
		}

		// Make HTTP request
		resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}

		// Parse response
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			fmt.Printf("Raw response: %s\n", string(body))
			continue
		}

		// Display results
		if apiResp.Success {
			fmt.Printf("Answer: %s\n", apiResp.Answer)
			fmt.Printf("Tokens Used: %d\n", apiResp.TokensUsed)

			if len(apiResp.VerboseOutput) > 0 {
				fmt.Printf("Verbose Output (%d messages):\n", len(apiResp.VerboseOutput))
				for i, verbose := range apiResp.VerboseOutput {
					fmt.Printf("  %d. [%s] %s\n", i+1, verbose.Level, verbose.Message)
				}
			}
		} else {
			fmt.Printf("Error: %s\n", apiResp.Error)
		}

		time.Sleep(1 * time.Second) // Rate limiting
	}

	fmt.Println("\n=== Testing Streaming Endpoint ===")
	testStreamingEndpoint(baseURL)
}

func testStreamingEndpoint(baseURL string) {
	// Create request for streaming
	req := APIRequest{
		Query:         "Explain machine learning in simple terms",
		VerboseFormat: "json",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}

	// Make streaming request
	resp, err := http.Post(baseURL+"/query/stream", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error making streaming request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read streaming response
	buffer := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading stream: %v\n", err)
			break
		}

		chunk := string(buffer[:n])
		fmt.Printf("Stream chunk: %s", chunk)
	}
}
