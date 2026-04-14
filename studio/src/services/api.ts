import * as yaml from 'js-yaml';
import { useAppsStore } from '../store/appsStore';
import { useServersStore } from '../store/serversStore';
import type { Agent, AgentCard } from '../types';
import { apiFetch } from '../lib/api-utils';

export const api = {
    fetchAgents: async (): Promise<{ agents: Agent[] }> => {
        const response = await apiFetch('/agents');
        if (!response.ok) {
            throw new Error(`Failed to fetch agents: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },

    fetchSchema: async () => {
        const response = await apiFetch('/schema');
        if (!response.ok) {
            throw new Error(`Failed to fetch schema: ${response.statusText}`);
        }
        return response.json();
    },

    fetchSessionEvents: async (
        sessionId: string,
        options: { limit?: number; afterEventId?: string; appId?: string } = {}
    ): Promise<{
        events: import('../types').SessionEvent[];
        eventCount: number;
        lastUpdate: string;
    }> => {
        const { limit = 500, afterEventId, appId } = options;

        const params = new URLSearchParams();
        params.set('limit', String(limit));
        if (appId) {
            params.set('app_id', appId);
            params.set('user_id', appId);
        }
        if (afterEventId) params.set('after_event_id', afterEventId);

        const response = await apiFetch(`/admin/sessions/${sessionId}?${params.toString()}`);
        if (!response.ok) {
            throw new Error(`Failed to fetch events: ${response.status} ${response.statusText}`);
        }
        const data = await response.json();
        return {
            events: data.events || [],
            eventCount: data.event_count || 0,
            lastUpdate: data.last_update || '',
        };
    },

    fetchSessions: async (appId = 'default', userID = 'default', pageSize = 50): Promise<{
        sessions: Array<{
            id: string;
            app_name: string;
            user_id: string;
            last_update: string;
            event_count: number;
        }>;
        next_page_token: string;
    }> => {
        const response = await apiFetch(`/admin/sessions?app_id=${appId}&user_id=${userID}&page_size=${pageSize}`);
        if (!response.ok) {
            throw new Error(`Failed to fetch sessions: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },

    deleteSessionOnBackend: async (sessionId: string, appId = 'default'): Promise<void> => {
        const response = await apiFetch(`/admin/sessions/${sessionId}?app_id=${appId}&user_id=${appId}`, {
            method: 'DELETE',
        });
        if (!response.ok && response.status !== 404) {
            throw new Error(`Failed to delete session: ${response.status} ${response.statusText}`);
        }
    },

    async fetchAgentCard(agentUrl: string): Promise<AgentCard> {
        const cardUrl = agentUrl.endsWith('/')
            ? `${agentUrl}.well-known/agent-card.json`
            : `${agentUrl}/.well-known/agent-card.json`;

        const response = await fetch(cardUrl);
        if (!response.ok) {
            throw new Error(`Failed to fetch agent card: ${response.status} ${response.statusText}`);
        }
        return response.json();
    },

    fetchConfig: async (): Promise<string> => {
        const activeServer = useServersStore.getState().getActiveServer();
        if (!activeServer) throw new Error('No active server');

        const serverId = activeServer.config.id;
        const serverData = useAppsStore.getState().appsByServer[serverId];
        const activeAppId = serverData?.activeAppId;

        if (activeAppId) {
            try {
                const res = await apiFetch(`/admin/apps/${activeAppId}`);
                if (!res.ok) throw new Error(`Failed to fetch app: ${res.status}`);
                const app = await res.json();
                if (app.config_json) {
                    try {
                        const jsonObj = JSON.parse(app.config_json);
                        return yaml.dump(jsonObj);
                    } catch {
                        try {
                            const yamlObj = yaml.load(app.config_json);
                            return yaml.dump(yamlObj);
                        } catch {
                            return app.config_json;
                        }
                    }
                }
                return "";
            } catch (error) {
                console.error("Failed to fetch app config:", error);
                throw error;
            }
        }

        return "";
    },

    saveConfig: async (content: string): Promise<{ message: string }> => {
        const activeServer = useServersStore.getState().getActiveServer();
        if (!activeServer) throw new Error('No active server');

        const serverId = activeServer.config.id;
        const serverData = useAppsStore.getState().appsByServer[serverId];
        const activeAppId = serverData?.activeAppId;

        if (activeAppId) {
            const jsonObj = yaml.load(content);
            const jsonString = JSON.stringify(jsonObj, null, 2);

            const res = await apiFetch(`/admin/apps/${activeAppId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ config_json: jsonString }),
            });
            if (!res.ok) throw new Error(`Failed to save config: ${res.status}`);

            return { message: "App configuration saved successfully." };
        }

        throw new Error('No active application selected.');
    }
};
