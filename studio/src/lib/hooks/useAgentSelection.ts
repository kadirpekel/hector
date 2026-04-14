import { useCallback } from 'react';
import { useStore } from '../../store/useStore';
import { useServersStore } from '../../store/serversStore';
import { api } from '../../services/api';
import { handleError } from '../error-handler';
import { DEFAULT_SUPPORTED_FILE_TYPES } from '../constants';
import type { Agent } from '../../types';

/**
 * Shared hook for agent selection logic
 * Eliminates duplicate code between ChatWidget and Sidebar
 */
export function useAgentSelection() {
  const availableAgents = useStore((state) => state.availableAgents);
  const setSelectedAgent = useStore((state) => state.setSelectedAgent);
  const setAgentCard = useStore((state) => state.setAgentCard);
  const setSupportedFileTypes = useStore((state) => state.setSupportedFileTypes);

  /**
   * Centralized agent card fetching with consistent error handling
   */
  const fetchAgentCardSafe = useCallback(
    async (agent: Agent) => {
      try {
        const card = await api.fetchAgentCard(agent.url);
        setAgentCard(card);
      } catch (error) {
        handleError(error, 'Failed to fetch agent card');
        setAgentCard(null);
        setSupportedFileTypes([...DEFAULT_SUPPORTED_FILE_TYPES]);
      }
    },
    [setAgentCard, setSupportedFileTypes]
  );

  /**
   * Handle agent selection change from dropdown
   * Manages session lifecycle: resume existing session for agent or create new one
   */
  const handleAgentChange = useCallback(
    async (input: React.ChangeEvent<HTMLSelectElement> | string) => {
      const name = typeof input === 'string' ? input : input.target.value;
      const agent = availableAgents.find((a) => a.name === name);

      if (agent) {
        setSelectedAgent(agent);
        
        // Resume or create session for this agent
        const sessions = useStore.getState().sessions;
        const activeServerId = useServersStore.getState().activeServerId;
        const agentSessions = Object.values(sessions)
          .filter(s => s.serverId === activeServerId && s.agentName === agent.name)
          .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime());
        
        if (agentSessions.length > 0) {
          // Resume most recent session for this agent
          useStore.getState().selectSession(agentSessions[0].id);
        } else {
          // No sessions for this agent - create one
          useStore.getState().createSession();
        }
        
        await fetchAgentCardSafe(agent);
      }
    },
    [availableAgents, setSelectedAgent, fetchAgentCardSafe]
  );

  return {
    handleAgentChange,
    fetchAgentCardSafe,
  };
}

