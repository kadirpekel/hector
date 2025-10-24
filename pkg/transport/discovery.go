package transport

import (
	"encoding/json"
	"net/http"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/auth"
)

type AgentDiscovery struct {
	service    DiscoverableService
	authConfig *AuthConfig
}

type DiscoverableService interface {
	ListAgents() []string
	GetAgent(agentName string) (pb.A2AServiceServer, bool)
	GetAgentCardAndVisibility(agentName string) (*pb.AgentCard, string, error)
}

type AuthConfig struct {
	Enabled   bool
	Validator *auth.JWTValidator
}

func NewAgentDiscovery(service DiscoverableService, authConfig *AuthConfig) *AgentDiscovery {
	return &AgentDiscovery{
		service:    service,
		authConfig: authConfig,
	}
}

func (d *AgentDiscovery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var claims *auth.Claims
	if d.authConfig != nil && d.authConfig.Enabled && d.authConfig.Validator != nil {

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

	isAuthenticated := claims != nil

	agentNames := d.service.ListAgents()

	agents := []*pb.AgentCard{}

	for _, agentName := range agentNames {
		card, visibility, err := d.service.GetAgentCardAndVisibility(agentName)
		if err != nil {
			continue
		}

		if visibility == "" {
			visibility = "public"
		}

		switch visibility {
		case "public":

		case "internal":

			if !isAuthenticated {
				continue
			}
		case "private":

			if !isAuthenticated {
				continue
			}
		default:

		}

		agents = append(agents, card)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"agents": agents,
		"total":  len(agents),
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (d *AgentDiscovery) AgentCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Use /v1/card endpoint for agent card",
		})
	}
}
