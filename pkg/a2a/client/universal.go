package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// UniversalA2AClient is a transport-agnostic A2A client that:
// 1. Discovers agent capabilities via agent card
// 2. Chooses appropriate transport (gRPC, REST, JSON-RPC)
// 3. Works with any A2A-compliant service
type UniversalA2AClient struct {
	baseURL    string
	agentID    string
	token      string
	agentCard  *pb.AgentCard
	httpClient *http.Client
	grpcClient pb.A2AServiceClient
	grpcConn   *grpc.ClientConn
	transport  string // "grpc", "rest", or "jsonrpc"
}

// NewUniversalA2AClient creates a client that auto-discovers the agent and chooses transport
// url can be:
// - Agent card URL: https://service.com/v1/agents/assistant/.well-known/agent-card.json
// - Agent-specific base URL: https://service.com/v1/agents/assistant
// - Service base URL: https://service.com (will discover agents)
func NewUniversalA2AClient(url, agentID, token string) (*UniversalA2AClient, error) {
	client := &UniversalA2AClient{
		baseURL: strings.TrimSuffix(url, "/"),
		agentID: agentID,
		token:   token,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}

	// Discover agent card
	if err := client.discoverAgent(); err != nil {
		return nil, fmt.Errorf("failed to discover agent: %w", err)
	}

	// Initialize transport based on agent card preferences
	if err := client.initializeTransport(); err != nil {
		return nil, fmt.Errorf("failed to initialize transport: %w", err)
	}

	return client, nil
}

// discoverAgent fetches the agent card to learn about the agent's capabilities
func (c *UniversalA2AClient) discoverAgent() error {
	// Try agent card URL patterns per A2A spec (Section 5.3)
	// Per A2A spec: agent cards MUST be at /.well-known/agent-card.json
	urls := []string{}

	// If agentID is provided, try agent-specific endpoint first
	if c.agentID != "" {
		urls = append(urls, fmt.Sprintf("%s/v1/agents/%s/.well-known/agent-card.json", c.baseURL, c.agentID))
	}

	// Try direct URL (might already be the card URL)
	urls = append(urls, c.baseURL)

	var lastErr error
	for _, url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		if c.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// Parse as JSON (A2A agent cards are JSON)
		var cardJSON map[string]interface{}
		if err := json.Unmarshal(body, &cardJSON); err != nil {
			lastErr = err
			continue
		}

		// Convert to AgentCard proto
		c.agentCard = jsonToAgentCard(cardJSON)

		// Extract agent ID from the URL we fetched
		// With agent-scoped endpoints, the agent ID is always in the URL path
		// e.g., /v1/agents/weather_assistant/.well-known/agent-card.json -> weather_assistant
		if c.agentID == "" && strings.Contains(url, "/v1/agents/") {
			parts := strings.Split(url, "/v1/agents/")
			if len(parts) > 1 {
				agentPath := strings.Split(parts[1], "/")[0]
				if agentPath != "" {
					c.agentID = agentPath
				}
			}
		}

		// Extract base service URL from the agent card URL
		// The agent card URL is always /v1/agents/{agent}
		// e.g., http://host/v1/agents/foo -> http://host
		if c.agentCard.Url != "" {
			cardURL := c.agentCard.Url
			if strings.Contains(cardURL, "/v1/agents/") {
				c.baseURL = strings.Split(cardURL, "/v1/agents/")[0]
			}
		}

		return nil
	}

	// If single-agent service, auto-route
	if c.agentID == "" {
		c.agentCard = &pb.AgentCard{
			Name:               "default",
			PreferredTransport: "grpc",
		}
		c.agentID = "default"
		return nil
	}

	return fmt.Errorf("failed to discover agent card: %w", lastErr)
}

// initializeTransport sets up the appropriate transport client
func (c *UniversalA2AClient) initializeTransport() error {
	// Determine transport from agent card
	preferredTransport := c.agentCard.GetPreferredTransport()
	if preferredTransport == "" {
		preferredTransport = "grpc" // Default to gRPC
	}

	// Parse URL from agent card if provided
	serviceURL := c.agentCard.GetUrl()
	if serviceURL == "" {
		serviceURL = c.baseURL
	}

	c.transport = preferredTransport

	switch preferredTransport {
	case "grpc":
		return c.initGRPC(serviceURL)
	case "rest", "http", "https":
		// Already have HTTP client
		return nil
	case "jsonrpc", "json-rpc":
		// Use HTTP client with JSON-RPC
		return nil
	default:
		// Fallback to gRPC
		return c.initGRPC(serviceURL)
	}
}

// initGRPC initializes gRPC transport
func (c *UniversalA2AClient) initGRPC(serviceURL string) error {
	// Extract host:port from URL
	grpcAddr := extractGRPCAddress(serviceURL)

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(grpcAddr, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect via gRPC: %w", err)
	}

	c.grpcConn = conn
	c.grpcClient = pb.NewA2AServiceClient(conn)
	return nil
}

// SendMessage sends a message using the appropriate transport
func (c *UniversalA2AClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	switch c.transport {
	case "grpc":
		return c.sendMessageGRPC(ctx, agentID, message)
	case "rest", "http", "https":
		return c.sendMessageREST(ctx, agentID, message)
	case "jsonrpc", "json-rpc":
		return c.sendMessageJSONRPC(ctx, agentID, message)
	default:
		return c.sendMessageGRPC(ctx, agentID, message)
	}
}

// sendMessageGRPC sends via gRPC
func (c *UniversalA2AClient) sendMessageGRPC(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// Add agent-name metadata if agentID specified
	if agentID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "agent-name", agentID)
	}

	req := &pb.SendMessageRequest{
		Request: message,
	}

	return c.grpcClient.SendMessage(ctx, req)
}

// sendMessageREST sends via REST
func (c *UniversalA2AClient) sendMessageREST(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// Use HTTP client from http.go
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.SendMessage(ctx, agentID, message)
}

// sendMessageJSONRPC sends via JSON-RPC
func (c *UniversalA2AClient) sendMessageJSONRPC(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// TODO: Implement JSON-RPC client
	// For now, fallback to REST
	return c.sendMessageREST(ctx, agentID, message)
}

// StreamMessage streams messages using the appropriate transport
func (c *UniversalA2AClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	// Use HTTP client for streaming (SSE)
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.StreamMessage(ctx, agentID, message)
}

// ListAgents lists available agents
func (c *UniversalA2AClient) ListAgents(ctx context.Context) ([]*pb.AgentCard, error) {
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.ListAgents(ctx)
}

// GetAgentCard gets agent card
func (c *UniversalA2AClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	if c.agentCard != nil && (agentID == "" || agentID == c.agentID) {
		return c.agentCard, nil
	}
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.GetAgentCard(ctx, agentID)
}

// GetTask gets task status
func (c *UniversalA2AClient) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.GetTask(ctx, agentID, taskID)
}

// CancelTask cancels a task
func (c *UniversalA2AClient) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	httpClient := NewHTTPClient(c.baseURL, c.token)
	return httpClient.CancelTask(ctx, agentID, taskID)
}

// GetAgentID returns the agent ID discovered/used by this client
func (c *UniversalA2AClient) GetAgentID() string {
	return c.agentID
}

// Close closes connections
func (c *UniversalA2AClient) Close() error {
	if c.grpcConn != nil {
		return c.grpcConn.Close()
	}
	return nil
}

// Helper functions

func extractGRPCAddress(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "grpc://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Extract host:port
	parts := strings.Split(url, "/")
	return parts[0]
}

func jsonToAgentCard(data map[string]interface{}) *pb.AgentCard {
	card := &pb.AgentCard{}

	if name, ok := data["name"].(string); ok {
		card.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		card.Description = desc
	}
	if version, ok := data["version"].(string); ok {
		card.Version = version
	}
	if url, ok := data["url"].(string); ok {
		card.Url = url
	}
	if transport, ok := data["preferred_transport"].(string); ok {
		card.PreferredTransport = transport
	}
	if transport, ok := data["preferredTransport"].(string); ok {
		card.PreferredTransport = transport
	}

	// Parse capabilities
	if caps, ok := data["capabilities"].(map[string]interface{}); ok {
		card.Capabilities = &pb.AgentCapabilities{}
		if streaming, ok := caps["streaming"].(bool); ok {
			card.Capabilities.Streaming = streaming
		}
	}

	return card
}
