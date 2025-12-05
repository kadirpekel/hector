import type { Agent, AgentCard } from '../types';
import { getBaseUrl } from '../lib/api-utils';

export const api = {
    // Fetch all agents from discovery endpoint (Hector extension)
    async fetchAgents(): Promise<{ agents: Agent[] }> {
        const baseUrl = getBaseUrl();
        const response = await fetch(`${baseUrl}/agents`);
        if (!response.ok) {
            throw new Error(`Failed to fetch agents: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },

    // Fetch agent card (A2A spec: /.well-known/agent-card.json)
    async fetchAgentCard(agentUrl: string): Promise<AgentCard> {
        // agentUrl is the full URL from the agent card, e.g., http://localhost:8080/agents/assistant
        // Per A2A spec, card is at {agentUrl}/.well-known/agent-card.json
        const cardUrl = agentUrl.endsWith('/') 
            ? `${agentUrl}.well-known/agent-card.json`
            : `${agentUrl}/.well-known/agent-card.json`;
        
        const response = await fetch(cardUrl);
        if (!response.ok) {
            throw new Error(`Failed to fetch agent card: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },
};
