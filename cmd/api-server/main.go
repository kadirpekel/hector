package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kadirpekel/hector"
)

// APIResponse represents the HTTP API response structure
type APIResponse struct {
	Answer        string                 `json:"answer"`
	VerboseOutput []hector.VerboseOutput `json:"verbose_output,omitempty"`
	TokensUsed    int                    `json:"tokens_used"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
}

// HTTPHandler handles HTTP requests for the Hector API
type HTTPHandler struct {
	agent *hector.Agent
}

// NewHTTPHandler creates a new HTTP handler with an agent
func NewHTTPHandler(agent *hector.Agent) *HTTPHandler {
	return &HTTPHandler{agent: agent}
}

// HandleQuery handles POST requests to /query endpoint
func (h *HTTPHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Query         string `json:"query"`
		VerboseFormat string `json:"verbose_format,omitempty"` // "terminal", "plain", "json", "html"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set verbose format if specified
	if request.VerboseFormat != "" {
		h.agent.ReasoningConfig.VerboseFormat = request.VerboseFormat
		h.agent.ReasoningConfig.Verbose = true
	}

	// Execute query
	response, err := h.agent.ExecuteQueryWithReasoning(request.Query)
	if err != nil {
		apiResp := APIResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(apiResp)
		return
	}

	// Get verbose outputs (this would be populated during execution)
	verboseOutputs := h.agent.GetVerboseOutputs()

	// Create API response
	apiResp := APIResponse{
		Answer:        response.Answer,
		VerboseOutput: verboseOutputs,
		TokensUsed:    response.TokensUsed,
		Success:       true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResp)
}

// HandleStreamingQuery handles POST requests to /query/stream endpoint
func (h *HTTPHandler) HandleStreamingQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Query         string `json:"query"`
		VerboseFormat string `json:"verbose_format,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set verbose format if specified
	if request.VerboseFormat != "" {
		h.agent.ReasoningConfig.VerboseFormat = request.VerboseFormat
		h.agent.ReasoningConfig.Verbose = true
	}

	// Set up Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Execute streaming query
	streamChan, err := h.agent.ExecuteQueryWithReasoningStreaming(request.Query)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\":\"%s\"}\n\n", err.Error())
		return
	}

	// Stream responses
	for chunk := range streamChan {
		// Format chunk as SSE
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func main() {
	// Load agent from config
	configPath := "local-configs/openai-minimal.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	agent, err := hector.LoadAgentFromFile(configPath)
	if err != nil {
		log.Fatalf("Failed to load agent: %v", err)
	}

	handler := NewHTTPHandler(agent)

	// Set up routes
	http.HandleFunc("/query", handler.HandleQuery)
	http.HandleFunc("/query/stream", handler.HandleStreamingQuery)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	port := ":8080"
	fmt.Printf("Starting Hector HTTP API server on port %s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  POST /query - Execute query with JSON response\n")
	fmt.Printf("  POST /query/stream - Execute query with streaming response\n")
	fmt.Printf("  GET /health - Health check\n")
	fmt.Printf("\nExample usage:\n")
	fmt.Printf("  curl -X POST http://localhost:8080/query \\\n")
	fmt.Printf("    -H 'Content-Type: application/json' \\\n")
	fmt.Printf("    -d '{\"query\": \"Hello world\", \"verbose_format\": \"json\"}'\n")

	log.Fatal(http.ListenAndServe(port, nil))
}
