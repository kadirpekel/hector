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
	GetAgent(agentName string) (pb.A2AServiceServer, bool)
	GetAgentCardAndVisibility(agentName string) (*pb.AgentCard, string, error)
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

	// Get all agent names
	agentNames := d.service.ListAgents()

	// Build response with pure A2A-compliant agent cards
	// The card.Name field IS the agent identifier - following A2A protocol!
	agents := []*pb.AgentCard{}

	for _, agentName := range agentNames {
		card, visibility, err := d.service.GetAgentCardAndVisibility(agentName)
		if err != nil {
			continue // Skip if card unavailable
		}

		// Apply visibility filtering (server-side only)
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

		// Return pure A2A AgentCard - card.Name is the identifier!
		// card.Url already contains the endpoint
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
		// Extract agent name from path (e.g., /v1/agents/{name})
		// This is handled by the grpc-gateway for /v1/card
		// This is just a fallback handler

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Use /v1/card endpoint for agent card",
		})
	}
}
