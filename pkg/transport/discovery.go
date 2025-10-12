package transport

import (
	"encoding/json"
	"net/http"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/auth"
)

// AgentDiscovery handles agent discovery endpoint
type AgentDiscovery struct {
	service    DiscoverableService
	authConfig *AuthConfig
}

// DiscoverableService interface for services that support agent discovery
type DiscoverableService interface {
	ListAgents() []string
	GetAgent(agentID string) (pb.A2AServiceServer, bool)
	GetAgentMetadata(agentID string) (*AgentMetadata, error)
}

// AgentMetadata holds agent metadata for discovery
type AgentMetadata struct {
	ID              string
	Name            string
	Description     string
	Version         string
	Visibility      string // "public", "internal", "private"
	Capabilities    *pb.AgentCapabilities
	SecuritySchemes map[string]*pb.SecurityScheme
	Security        []*pb.Security
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled   bool
	Validator *auth.JWTValidator
}

// NewAgentDiscovery creates a new agent discovery handler
func NewAgentDiscovery(service DiscoverableService, authConfig *AuthConfig) *AgentDiscovery {
	return &AgentDiscovery{
		service:    service,
		authConfig: authConfig,
	}
}

// ServeHTTP handles the /v1/agents discovery endpoint
// Per A2A spec section 5.2:
// - Returns list of agent cards
// - Filters by visibility (public by default, internal/private require auth)
// - Returns 200 OK with JSON array of agent cards
func (d *AgentDiscovery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if enabled
	var claims *auth.Claims
	if d.authConfig != nil && d.authConfig.Enabled && d.authConfig.Validator != nil {
		// Try to get claims (optional for public agents)
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			tokenString := authHeader[len("Bearer "):]
			claimsInterface, err := d.authConfig.Validator.ValidateToken(r.Context(), tokenString)
			if err == nil {
				if c, ok := claimsInterface.(*auth.Claims); ok {
					claims = c
				}
			}
		}
	}

	// Determine which agents to show based on auth status
	isAuthenticated := claims != nil

	// Get all agent IDs
	agentIDs := d.service.ListAgents()

	// Build response with agent cards
	type AgentCard struct {
		ID              string                        `json:"id"`
		Name            string                        `json:"name"`
		Description     string                        `json:"description"`
		Version         string                        `json:"version"`
		Visibility      string                        `json:"visibility,omitempty"`
		Capabilities    *pb.AgentCapabilities         `json:"capabilities,omitempty"`
		Endpoint        string                        `json:"endpoint,omitempty"`
		AgentCardURL    string                        `json:"agent_card_url,omitempty"`
		SecuritySchemes map[string]*pb.SecurityScheme `json:"security_schemes,omitempty"`
		Security        []*pb.Security                `json:"security,omitempty"`
	}

	agents := []AgentCard{}

	for _, agentID := range agentIDs {
		metadata, err := d.service.GetAgentMetadata(agentID)
		if err != nil {
			continue // Skip if metadata unavailable
		}

		// Apply visibility filtering per A2A spec section 5.2
		visibility := metadata.Visibility
		if visibility == "" {
			visibility = "public" // Default visibility
		}

		// Filter based on visibility and auth status
		switch visibility {
		case "public":
			// Always include public agents
		case "internal":
			// Only include if authenticated
			if !isAuthenticated {
				continue
			}
		case "private":
			// Only include if authenticated (could add tenant check here)
			if !isAuthenticated {
				continue
			}
		default:
			// Unknown visibility, treat as public
		}

		// Build agent card
		card := AgentCard{
			ID:              agentID,
			Name:            metadata.Name,
			Description:     metadata.Description,
			Version:         metadata.Version,
			Visibility:      visibility,
			Capabilities:    metadata.Capabilities,
			Endpoint:        "/v1/agents/" + agentID,
			AgentCardURL:    "/v1/agents/" + agentID + "/.well-known/agent-card.json",
			SecuritySchemes: metadata.SecuritySchemes,
			Security:        metadata.Security,
		}

		agents = append(agents, card)
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	_ = json.NewEncoder(w).Encode(response)
}

// AgentCardHandler returns the card for a specific agent
func (d *AgentDiscovery) AgentCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract agent ID from path (e.g., /v1/agents/{agentID})
		// This is handled by the grpc-gateway for /v1/card
		// This is just a fallback handler

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Use /v1/card endpoint for agent card",
		})
	}
}
