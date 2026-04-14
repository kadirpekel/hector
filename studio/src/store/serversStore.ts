/**
 * Servers Store
 *
 * Manages server connections.
 * Servers are added/removed manually by the user and persisted in localStorage.
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { ServerConfig, ServerState, ServerStatus } from '../types';

interface ServersStore {
  servers: Record<string, ServerState>;
  activeServerId: string | null;

  // Accessors
  getActiveServer: () => ServerState | null;

  // Server lifecycle
  addServer: (config: ServerConfig) => void;
  updateServer: (server: ServerState) => void;
  updateServerConfig: (id: string, updates: Partial<ServerConfig>) => void;
  removeServer: (id: string) => void;
  selectServer: (id: string) => void;

  // Status transitions
  setServerStatus: (id: string, status: ServerStatus, error?: string, timestamp?: number) => void;

  // Data updates
  setServerConfig: (id: string, yaml: string) => void;
}

export const useServersStore = create<ServersStore>()(
  persist(
    (set, get) => ({
      servers: {},
      activeServerId: null,

      getActiveServer: () => {
        const { servers, activeServerId } = get();
        if (!activeServerId) return null;
        return servers[activeServerId] || null;
      },

      addServer: (config: ServerConfig) => {
        set((state) => ({
          servers: {
            ...state.servers,
            [config.id]: {
              config,
              status: 'added',
              statusUpdatedAt: Date.now(),
              agents: [],
              configYaml: null,
              lastError: null,
            },
          },
          activeServerId: state.activeServerId ?? config.id,
        }));
      },

      updateServer: (server: ServerState) => {
        set((state) => ({
          servers: {
            ...state.servers,
            [server.config.id]: server,
          },
        }));
      },

      updateServerConfig: (id: string, updates: Partial<ServerConfig>) => {
        set((state) => {
          const server = state.servers[id];
          if (!server) return state;
          return {
            servers: {
              ...state.servers,
              [id]: {
                ...server,
                config: { ...server.config, ...updates },
              },
            },
          };
        });
      },

      removeServer: (id: string) => {
        set((state) => {
          const { [id]: _removed, ...rest } = state.servers;
          const newActiveId = state.activeServerId === id
            ? Object.keys(rest)[0] ?? null
            : state.activeServerId;
          return {
            servers: rest,
            activeServerId: newActiveId,
          };
        });
      },

      selectServer: (id: string) => {
        set({ activeServerId: id });
      },

      setServerStatus: (id: string, status: ServerStatus, error?: string, timestamp?: number) => {
        set((state) => {
          const server = state.servers[id];
          if (!server) return state;

          const now = timestamp || Date.now();

          // Reject stale status updates
          if (timestamp && server.statusUpdatedAt && timestamp < server.statusUpdatedAt) {
            return state;
          }

          const startedAt = status === 'checking' && server.status !== 'checking'
            ? now
            : server.startedAt;

          return {
            servers: {
              ...state.servers,
              [id]: {
                ...server,
                status,
                statusUpdatedAt: now,
                startedAt,
                lastError: error ?? null,
              },
            },
          };
        });
      },

      setServerConfig: (id: string, yaml: string) => {
        set((state) => {
          const server = state.servers[id];
          if (!server) return state;
          return {
            servers: {
              ...state.servers,
              [id]: {
                ...server,
                configYaml: yaml,
              },
            },
          };
        });
      },
    }),
    {
      name: 'hector_servers_ui',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        servers: Object.entries(state.servers).reduce((acc, [id, s]) => {
          // Persist server configs so user doesn't re-enter URLs on reload
          acc[id] = {
            config: s.config,
            status: 'added' as ServerStatus,
            statusUpdatedAt: 0,
            agents: [],
            configYaml: null,
            lastError: null,
          };
          return acc;
        }, {} as Record<string, ServerState>),
        activeServerId: state.activeServerId,
      }),
    }
  )
);
