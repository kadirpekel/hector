import type { Agent, AgentCard } from '../types';
import { getBaseUrl } from '../lib/api-utils';

export const api = {
    async fetchAgents(): Promise<{ agents: Agent[] }> {
        const baseUrl = getBaseUrl();
        const response = await fetch(`${baseUrl}/v1/agents`);
        if (!response.ok) {
            throw new Error(`Failed to fetch agents: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },

    async fetchAgentCard(agentUrl: string): Promise<AgentCard> {
        const baseUrl = getBaseUrl();
        // Extract agent ID from URL (format: /v1/agents/{agentId})
        const urlMatch = agentUrl.match(/\/v1\/agents\/([^\/]+)/);
        if (!urlMatch) {
            throw new Error('Could not extract agent ID from URL');
        }
        const agentId = urlMatch[1];
        const response = await fetch(`${baseUrl}/v1/agents/${agentId}/.well-known/agent-card.json`);
        if (!response.ok) {
            throw new Error(`Failed to fetch agent card: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },
};
