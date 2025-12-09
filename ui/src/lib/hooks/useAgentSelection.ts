import { useCallback } from 'react';
import { useStore } from '../../store/useStore';
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
   * Uses name-based lookup for referential stability
   */
  const handleAgentChange = useCallback(
    async (e: React.ChangeEvent<HTMLSelectElement>) => {
      const name = e.target.value;
      const agent = availableAgents.find((a) => a.name === name);

      if (agent) {
        setSelectedAgent(agent);
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
