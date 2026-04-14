import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { api } from "../services/api";
import { v4 as uuidv4 } from "uuid";
import type { Session, Message, Agent, AgentCard, Widget } from "../types";
import { StreamParser as StreamParserClass } from "../lib/stream-parser";
import { DEFAULT_SUPPORTED_FILE_TYPES } from "../lib/constants";
import { logger } from "../lib/logger";
import { useServersStore } from "./serversStore";
import { useAppsStore } from "./appsStore";
import { processSessionEvents } from "../lib/event-processor";

type StreamParser = StreamParserClass;

/** Get the active app ID for the current server (used for session API calls) */
function getActiveAppId(): string {
  const serverId = useServersStore.getState().activeServerId;
  if (!serverId) return 'default';
  return useAppsStore.getState().getActiveAppId(serverId) || 'default';
}

/** Get the app ID for a specific session, preferring its stored appId over the global active */
function getSessionAppId(session: Session | undefined): string {
  return session?.appId || getActiveAppId();
}

// Type for persisted state fields (added by zustand persist middleware)
interface PersistedState {
  selectedAgentName: string | null;
}

interface AppState {
  // UI State
  sidebarVisible: boolean;
  setSidebarVisible: (visible: boolean) => void;
  minimalMode: boolean;
  setMinimalMode: (enabled: boolean) => void;
  
  isHistoryPinned: boolean;
  setIsHistoryPinned: (pinned: boolean) => void;

  configVisible: boolean;
  setConfigVisible: (visible: boolean) => void;
  isGenerating: boolean;
  setIsGenerating: (generating: boolean) => void;
  error: string | null;
  setError: (error: string | null) => void;
  successMessage: string | null;
  setSuccessMessage: (message: string | null) => void;
  editorTheme: 'vs-dark' | 'vs-light' | 'hc-black';
  setEditorTheme: (theme: 'vs-dark' | 'vs-light' | 'hc-black') => void;

  // Streaming optimization: separate buffer for high-frequency text updates
  // This prevents re-rendering entire message structure on every token
  streamingTextContent: Record<string, string>; // widgetId -> accumulating text
  setStreamingTextContent: (widgetId: string, content: string) => void;
  clearStreamingTextContent: (widgetId: string) => void;

  // Config State
  endpointUrl: string;
  setEndpointUrl: (url: string) => void;
  protocol: "jsonrpc" | "rest";
  setProtocol: (protocol: "jsonrpc" | "rest") => void;
  streamingEnabled: boolean;
  setStreamingEnabled: (enabled: boolean) => void;

  // Stream cancellation
  activeStreamParser: StreamParser | null;
  setActiveStreamParser: (parser: StreamParser | null) => void;
  cancelGeneration: () => void;

  // Data State
  sessions: Record<string, Session>;
  sessionsLoading: boolean;
  currentSessionId: string | null;
  availableAgents: Agent[];
  selectedAgent: Agent | null;
  agentCard: AgentCard | null;
  supportedFileTypes: string[];
  selectedNodeId: string | null;
  schema: any;
  activeAgentId: string | null;
  agentsLoaded: boolean;
  
  // Session X-Ray (Full Screen Modal)
  xRaySessionId: string | null;
  setXRaySessionId: (id: string | null) => void;

  // Actions
  setAvailableAgents: (agents: Agent[]) => void;
  loadAgents: () => Promise<void>;
  reloadAgents: () => Promise<void>; // Force reload agents (for deploy)
  setSelectedNodeId: (id: string | null) => void;
  setSelectedAgent: (agent: Agent | null) => void;
  setActiveAgentId: (id: string | null) => void;
  setAgentCard: (card: AgentCard | null) => void;
  setSupportedFileTypes: (types: string[]) => void;
  setSchema: (schema: any) => void;

  createSession: () => string;
  selectSession: (sessionId: string | null) => void;
  deleteSession: (sessionId: string) => void;
  deleteAllSessionsForServer: (serverId: string) => void;
  updateSessionTitle: (sessionId: string, title: string) => void;
  syncSessions: () => Promise<void>;
  loadSessionEvents: (sessionId: string) => Promise<void>;
  refreshSessionEvents: (sessionId: string) => Promise<void>; // Force reload from backend
  pollSessionForNewEvents: (sessionId: string) => Promise<void>; // Cursor-based incremental poll

  addMessage: (sessionId: string, message: Message) => void;
  updateMessage: (
    sessionId: string,
    messageId: string,
    updates: Partial<Message>,
  ) => void;
  setSessionTaskId: (sessionId: string, taskId: string | null) => void;
  updateSessionCursor: (sessionId: string, lastEventId: string) => void;

  // Widget state persistence
  setWidgetExpanded: (
    sessionId: string,
    messageId: string,
    widgetId: string,
    expanded: boolean,
  ) => void;
  getWidgetExpanded: (
    sessionId: string,
    messageId: string,
    widgetId: string,
  ) => boolean;

  // Widget management for contextual blocks (thinking, tools, etc.)
  addWidget: (
    sessionId: string,
    messageId: string,
    widget: Widget,
  ) => void;
  updateWidget: (
    sessionId: string,
    messageId: string,
    widgetId: string,
    updates: Partial<Widget>,
  ) => void;
  addToContentOrder: (
    sessionId: string,
    messageId: string,
    widgetId: string,
  ) => void;
  // Optimized action for high-frequency stream updates
  appendTextWidgetContent: (
    sessionId: string,
    messageId: string,
    widgetId: string,
    textDelta: string,
  ) => void;

  // Finalize streaming content (commit buffer to message)
  finalizeStreamingText: (
    sessionId: string,
    messageId: string,
    widgetId: string,
  ) => void;

  // Studio UI State
  studioViewMode: 'design' | 'split' | 'chat';
  studioDesignView: 'editor' | 'canvas';
  studioRightPanelWidth: number;
  studioYamlContent: string;
  studioIsValidYaml: boolean;
  studioValidationError: string;
  studioIsDeploying: boolean;
  isServerStudioEnabled: boolean; // Whether connected server has studio mode enabled

  setStudioViewMode: (mode: 'design' | 'split' | 'chat') => void;
  setStudioDesignView: (view: 'editor' | 'canvas') => void;
  setStudioRightPanelWidth: (width: number) => void;
  setStudioYamlContent: (content: string) => void;
  setStudioValidationStatus: (isValid: boolean, error?: string) => void;
  setStudioIsDeploying: (isDeploying: boolean) => void;
  setIsServerStudioEnabled: (enabled: boolean) => void;

  // RAG Indexing Status
  ragStatus: {
    isIndexing: boolean;
    stores: Record<string, { indexed: number; total: number; indexing?: boolean }>;
  } | null;
  setRagStatus: (status: { isIndexing: boolean; stores: Record<string, { indexed: number; total: number; indexing?: boolean }> } | null) => void;

  // Hydration State
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useStore = create<AppState>()(
  persist(
    (set, get) => ({
      sidebarVisible: true,
      setSidebarVisible: (visible) => set({ sidebarVisible: visible }),
      minimalMode: false,
      setMinimalMode: (enabled) =>
        set({ minimalMode: enabled, sidebarVisible: !enabled }),
      
      // History UI State
      isHistoryPinned: false,
      setIsHistoryPinned: (pinned) => set({ isHistoryPinned: pinned }),

      configVisible: false,
      setConfigVisible: (visible) => set({ configVisible: visible }),
      isGenerating: false,
      setIsGenerating: (generating) => set({ isGenerating: generating }),
      error: null,
      setError: (error) => set({ error }),
      successMessage: null,
      setSuccessMessage: (successMessage) => set({ successMessage }),
      editorTheme: 'hc-black',
      setEditorTheme: (theme) => set({ editorTheme: theme }),

      // Streaming text buffer (optimized for high-frequency updates)
      streamingTextContent: {},
      setStreamingTextContent: (widgetId, content) =>
        set((state) => ({
          streamingTextContent: {
            ...state.streamingTextContent,
            [widgetId]: content,
          },
        })),
      clearStreamingTextContent: (widgetId) =>
        set((state) => {
          const newContent = { ...state.streamingTextContent };
          delete newContent[widgetId];
          return { streamingTextContent: newContent };
        }),

      // Config defaults
      endpointUrl: "http://localhost:8080",
      setEndpointUrl: (url) => set({ endpointUrl: url }),
      protocol: "jsonrpc" as const,
      setProtocol: (protocol) => set({ protocol }),
      streamingEnabled: true,
      setStreamingEnabled: (enabled) => set({ streamingEnabled: enabled }),

      // Stream cancellation
      activeStreamParser: null,
      setActiveStreamParser: (parser) => set({ activeStreamParser: parser }),
      cancelGeneration: () => {
        const state = get();
        const parser = state.activeStreamParser;
        if (parser) {
          parser.abort();
          state.setActiveStreamParser(null);
          state.setIsGenerating(false);

          // Mark last agent message as cancelled
          if (state.currentSessionId) {
            const session = state.sessions[state.currentSessionId];
            if (session && session.messages.length > 0) {
              for (let i = session.messages.length - 1; i >= 0; i--) {
                if (session.messages[i].role === "agent") {
                  state.updateMessage(
                    state.currentSessionId,
                    session.messages[i].id,
                    { cancelled: true },
                  );
                  break;
                }
              }
            }
          }
        }
      },

      sessions: {},
      sessionsLoading: false,
      currentSessionId: null,
      availableAgents: [],
      selectedAgent: null,
      selectedNodeId: null,
      agentCard: null,
      supportedFileTypes: [...DEFAULT_SUPPORTED_FILE_TYPES],
      schema: null,
      activeAgentId: null,
      agentsLoaded: false,
      
      xRaySessionId: null,
      setXRaySessionId: (id) => set({ xRaySessionId: id }),

      setAvailableAgents: (agents) =>
        set((state) => {
          let newSelectedAgent = state.selectedAgent;

          // CRITICAL: Preserve referential identity if agent still exists
          // This ensures dropdown and other components using object reference equality work correctly
          if (newSelectedAgent) {
            const match = agents.find((a) => a.name === newSelectedAgent?.name);
            if (match) {
              // Agent still exists - update to new object reference from fresh agents list
              newSelectedAgent = match;
              logger.log(
                `Agent "${match.name}" still available after reload, preserving selection`,
              );
            } else {
              // Agent disappeared - clear selection
              logger.log(
                `Agent "${newSelectedAgent.name}" no longer available, clearing selection`,
              );
              newSelectedAgent = null;
            }
          }

          return {
            availableAgents: agents,
            selectedAgent: newSelectedAgent,
            agentsLoaded: true,
          };
        }),
      loadAgents: async () => {
        const state = get();

        // Idempotency guard - prevent multiple simultaneous loads
        if (state.agentsLoaded) {
          logger.log("Agents already loaded, skipping");
          return;
        }

        // Capture current endpoint to detect server switch during fetch
        const currentEndpoint = state.endpointUrl;

        try {
          const response = await api.fetchAgents();
          let agents: Agent[] = [];

          if (response && Array.isArray(response.agents)) {
            agents = response.agents;
          } else if (Array.isArray(response)) {
            // Fallback if backend changes signature to return direct array
            agents = response;
          }

          // Use setAvailableAgents to preserve referential identity
          get().setAvailableAgents(agents);

          // Restore persisted selection after loading agents (only on initial load)
          const persistedName = (get() as AppState & PersistedState).selectedAgentName;
          
          if (!get().selectedAgent && agents.length > 0) {
             if (persistedName) {
                const restoredAgent = agents.find((a) => a.name === persistedName);
                if (restoredAgent) {
                  logger.log(
                    `Restored agent selection from localStorage: ${persistedName}`,
                  );
                  set({ selectedAgent: restoredAgent });
                } else {
                  logger.log(
                    `Persisted agent "${persistedName}" not found, selecting first available`,
                  );
                  set({ selectedAgent: agents[0] });
                }
             } else {
                // No persistence, select first match
                logger.log(`No active agent, selecting first available: ${agents[0].name}`);
                set({ selectedAgent: agents[0] });
             }
          }
        } catch (e) {
          logger.error("Failed to load agents", e);
          // Only show error toast if endpoint hasn't changed (i.e., we haven't switched server)
          if (get().endpointUrl === currentEndpoint) {
            set({ error: "Failed to load agents. Please check connection." });
          }
        }
      },
      reloadAgents: async () => {
        logger.log("Forcing agent reload (deploy/config change)");
        // Reset state so loadAgents picks a fresh agent for the new context
        set({ agentsLoaded: false, selectedAgent: null });
        await get().loadAgents();
      },
      setSelectedNodeId: (id) => set({ selectedNodeId: id }),
      setSelectedAgent: (agent) => set({ selectedAgent: agent }),
      setActiveAgentId: (id) => set({ activeAgentId: id }),
      setAgentCard: (card) => {
        set({ agentCard: card });
        // Update supported file types from agent card
        if (
          card &&
          card.defaultInputModes &&
          Array.isArray(card.defaultInputModes)
        ) {
          // Filter to only file/media types (exclude text/plain, application/json)
          const fileTypes = card.defaultInputModes.filter(
            (mode: string) =>
              mode.startsWith("image/") ||
              mode.startsWith("video/") ||
              mode.startsWith("audio/"),
          );

          // If no file types found, fall back to image defaults
          if (fileTypes.length === 0) {
            set({ supportedFileTypes: [...DEFAULT_SUPPORTED_FILE_TYPES] });
          } else {
            set({ supportedFileTypes: fileTypes });
          }
        } else {
          // No input modes specified, use defaults
          set({ supportedFileTypes: [...DEFAULT_SUPPORTED_FILE_TYPES] });
        }
      },
      setSupportedFileTypes: (types) => set({ supportedFileTypes: types }),
      setSchema: (schema) => set({ schema }),

      createSession: () => {
        const id = `session-${uuidv4()}`;
        // Get active server ID from servers store
        const activeServerId = useServersStore.getState().activeServerId || undefined;
        // Get current agent name
        const agentName = get().selectedAgent?.name || undefined;
        
        const appId = getActiveAppId();
        const newSession: Session = {
          id,
          title: "New conversation",
          created: new Date().toISOString(),
          messages: [],
          contextId: id, // Sync with session ID for X-Ray / Persistence consistency
          taskId: null,
          serverId: activeServerId,
          appId: appId,
          agentName: agentName,
        };

        set((state) => ({
          sessions: { ...state.sessions, [id]: newSession },
          currentSessionId: id,
        }));

        return id;
      },

      selectSession: (sessionId) => set({ currentSessionId: sessionId }),

      deleteSession: (sessionId) => {
        const activeServerId = useServersStore.getState().activeServerId;
        const session = get().sessions[sessionId];
        const sessionAppId = session?.appId;
        
        // Delete from backend (fire and forget - don't block UI)
        api.deleteSessionOnBackend(sessionId, getSessionAppId(session)).catch((err) => {
          logger.log(`deleteSession: Backend delete failed (may not exist) - ${err}`);
        });
        
        set((state) => {
          const newSessions = { ...state.sessions };
          delete newSessions[sessionId];

          let newCurrentId = state.currentSessionId;
          if (state.currentSessionId === sessionId) {
            // Find fallback session for same server + app
            const fallbackSessions = Object.values(newSessions)
              .filter(s => {
                if (s.serverId !== activeServerId) return false;
                if (s.appId !== sessionAppId) return false;
                return true;
              })
              .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime());
            
            newCurrentId = fallbackSessions.length > 0 ? fallbackSessions[0].id : null;
          }

          return {
            sessions: newSessions,
            currentSessionId: newCurrentId,
          };
        });
        
        // If no session left for current agent, create a new one
        if (get().currentSessionId === null) {
          get().createSession();
        }
      },

      deleteAllSessionsForServer: (serverId) => {
        const activeAppId = getActiveAppId();
        // Collect sessions to delete - scoped to current app
        const sessionsToDelete = Object.values(get().sessions)
          .filter(s => {
            if (s.serverId !== serverId) return false;
            if (s.appId !== activeAppId) return false;
            return true;
          });
        
        // Delete from backend (fire and forget - don't block UI)
        for (const session of sessionsToDelete) {
          api.deleteSessionOnBackend(session.id, getSessionAppId(session)).catch((err) => {
            logger.log(`deleteAllSessionsForServer: Backend delete failed for ${session.id} - ${err}`);
          });
        }
        
        const deleteIds = new Set(sessionsToDelete.map(s => s.id));
        
        set((state) => {
          const newSessions = { ...state.sessions };
          let newCurrentId = state.currentSessionId;

          for (const sid of deleteIds) {
            delete newSessions[sid];
            if (state.currentSessionId === sid) {
              newCurrentId = null;
            }
          }

          // If we lost our current session, find another for same server
          if (newCurrentId === null) {
            const remaining = Object.values(newSessions)
              .filter(s => s.serverId === serverId)
              .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime());
            newCurrentId = remaining.length > 0 ? remaining[0].id : null;
          }
          
          logger.log(`Deleted ${deleteIds.size} sessions for server ${serverId}`);

          return {
            sessions: newSessions,
            currentSessionId: newCurrentId,
          };
        });

        // Ensure we always have at least one session
        if (get().currentSessionId === null) {
          get().createSession();
        }
      },

      updateSessionTitle: (sessionId, title) => {
        set((state) => ({
          sessions: {
            ...state.sessions,
            [sessionId]: { ...state.sessions[sessionId], title },
          },
        }));
      },

      syncSessions: async () => {
        const activeServerId = useServersStore.getState().activeServerId;
        
        if (!activeServerId) {
          logger.log('syncSessions: No active server, skipping');
          return;
        }

        set({ sessionsLoading: true });

        try {
          // Fetch sessions from backend
          const appId = getActiveAppId();
          const response = await api.fetchSessions(appId, appId);
          
          // Merge backend sessions with local sessions
          // Backend sessions that don't exist locally get created as stubs
          // Local sessions preserve their messages/widgets (not stored on backend)
          set((state) => {
            const newSessions = { ...state.sessions };
            
            for (const backendSession of response.sessions) {
              const sessionId = backendSession.id;
              
              if (!newSessions[sessionId]) {
                // New session from backend (webhook/headless) - create local stub
                newSessions[sessionId] = {
                  id: sessionId,
                  title: `Session ${sessionId.slice(-8)}`, // Short ID as default title
                  created: backendSession.last_update,
                  messages: [], // Messages will be loaded on demand
                  contextId: sessionId,
                  taskId: null,
                  serverId: activeServerId,
                  appId: appId,
                  agentName: undefined, // Will be set when events are loaded
                };
                logger.log(`syncSessions: Discovered new session ${sessionId}`);
              } else {
                // Existing local session - backfill appId if missing
                if (!newSessions[sessionId].appId) {
                  newSessions[sessionId] = { ...newSessions[sessionId], appId };
                }
              }
            }
            
            return { sessions: newSessions };
          });
          
          logger.log(`syncSessions: Synced ${response.sessions.length} sessions`);
        } catch (error) {
          logger.log(`syncSessions: Error - ${error}`);
          // Silent fail - don't break the UI if backend sync fails
        } finally {
          set({ sessionsLoading: false });
        }
      },

      /**
       * Load session events from backend using unified EventProcessor
       * 
       * This is the stateless approach:
       * - Backend is single source of truth
       * - Events are fetched and processed through unified renderer
       * - Cursor (lastEventId) is tracked for efficient resume
       */
      loadSessionEvents: async (sessionId: string) => {
        const session = get().sessions[sessionId];
        if (!session) {
          logger.log(`loadSessionEvents: Session ${sessionId} not found`);
          return;
        }
        
        // Skip if session already has messages (from streaming or prior load)
        if (session.messages.length > 0) {
          logger.log(`loadSessionEvents: Session ${sessionId} already has messages, skipping (use refreshSessionEvents to reload)`);
          // Mark as loaded so polling can start
          if (!session.eventsLoaded) {
            set((state) => ({
              sessions: {
                ...state.sessions,
                [sessionId]: { ...state.sessions[sessionId], eventsLoaded: true },
              },
            }));
          }
          return;
        }
        
        try {
          const appId = getSessionAppId(session);
          logger.log(`loadSessionEvents: Fetching events for ${sessionId} (appId: ${appId})`);
          const { events } = await api.fetchSessionEvents(sessionId, { limit: 500, appId });
          
          if (events.length === 0) {
            logger.log(`loadSessionEvents: No events for ${sessionId}`);
            // Mark as loaded even if empty
            set((state) => ({
              sessions: {
                ...state.sessions,
                [sessionId]: {
                  ...state.sessions[sessionId],
                  eventsLoaded: true,
                },
              },
            }));
            return;
          }
          
          // Process events through unified EventProcessor
          const { messages, lastEventId } = processSessionEvents(events);
          
          // Update session with processed messages and cursor
          if (messages.length > 0) {
            set((state) => ({
              sessions: {
                ...state.sessions,
                [sessionId]: {
                  ...state.sessions[sessionId],
                  messages: messages.map(m => ({
                    id: m.id,
                    role: m.role,
                    text: m.text,
                    metadata: m.metadata,
                    widgets: m.widgets,
                    time: m.time,
                  })),
                  title: messages[0]?.text?.slice(0, 50) || state.sessions[sessionId].title,
                  lastEventId: lastEventId || undefined,
                  eventsLoaded: true,
                },
              },
            }));
            logger.log(`loadSessionEvents: Loaded ${messages.length} messages for ${sessionId} (cursor: ${lastEventId})`);
          } else {
            set((state) => ({
              sessions: {
                ...state.sessions,
                [sessionId]: {
                  ...state.sessions[sessionId],
                  eventsLoaded: true,
                },
              },
            }));
          }
        } catch (error) {
          logger.log(`loadSessionEvents: Error - ${error}`);
        }
      },

      /**
       * Force refresh session events from backend (full reload)
       * 
       * Fetches ALL events and replaces existing messages.
       * Use this when you need to completely reload a session.
       */
      refreshSessionEvents: async (sessionId: string) => {
        const session = get().sessions[sessionId];
        if (!session) {
          logger.log(`refreshSessionEvents: Session ${sessionId} not found`);
          return;
        }
        
        try {
          const appId = getSessionAppId(session);
          logger.log(`refreshSessionEvents: Full reload for ${sessionId} (appId: ${appId})`);
          
          const { events } = await api.fetchSessionEvents(sessionId, { limit: 500, appId });
          
          if (events.length === 0) {
            logger.log(`refreshSessionEvents: No events for ${sessionId}`);
            set((state) => ({
              sessions: {
                ...state.sessions,
                [sessionId]: {
                  ...state.sessions[sessionId],
                  messages: [],
                  lastEventId: undefined,
                  eventsLoaded: true,
                },
              },
            }));
            return;
          }
          
          // Process all events
          const { messages, lastEventId } = processSessionEvents(events);
          
          // Replace all messages
          set((state) => ({
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...state.sessions[sessionId],
                messages: messages.map(m => ({
                  id: m.id,
                  role: m.role,
                  text: m.text,
                  metadata: m.metadata,
                  widgets: m.widgets,
                  time: m.time,
                })),
                title: messages[0]?.text?.slice(0, 50) || state.sessions[sessionId].title,
                lastEventId: lastEventId || undefined,
                eventsLoaded: true,
              },
            },
          }));
          
          logger.log(`refreshSessionEvents: Loaded ${messages.length} messages for ${sessionId}`);
        } catch (error) {
          logger.log(`refreshSessionEvents: Error - ${error}`);
        }
      },
      
      /**
       * Poll for new events (cursor-based incremental fetch)
       * 
       * ONLY fetches events AFTER the last event ID.
       * Must have a cursor (lastEventId) to work.
       * Appends new messages to existing ones.
       */
      pollSessionForNewEvents: async (sessionId: string) => {
        const session = get().sessions[sessionId];
        if (!session) {
          return;
        }
        
        // Must have cursor to poll
        if (!session.lastEventId) {
          return;
        }
        
        try {
          const appId = getSessionAppId(session);
          const { events } = await api.fetchSessionEvents(sessionId, {
            limit: 100,
            afterEventId: session.lastEventId,
            appId,
          });
          
          // No new events - nothing to do
          if (events.length === 0) {
            return;
          }
          
          logger.log(`pollSessionForNewEvents: Got ${events.length} new events for ${sessionId}`);
          
          // Process new events
          const { messages: newMessages, lastEventId } = processSessionEvents(events);
          
          if (newMessages.length === 0) {
            // Update cursor even if no messages (events might be metadata-only)
            if (lastEventId) {
              set((state) => ({
                sessions: {
                  ...state.sessions,
                  [sessionId]: {
                    ...state.sessions[sessionId],
                    lastEventId,
                  },
                },
              }));
            }
            return;
          }
          
          // Append new messages (with duplicate filtering as safety)
          set((state) => {
            const currentSession = state.sessions[sessionId];
            if (!currentSession) return state;
            
            const existingIds = new Set(currentSession.messages.map(m => m.id));
            const uniqueNew = newMessages.filter(m => !existingIds.has(m.id));
            
            if (uniqueNew.length === 0) {
              // Still update cursor
              return {
                sessions: {
                  ...state.sessions,
                  [sessionId]: {
                    ...currentSession,
                    lastEventId: lastEventId || currentSession.lastEventId,
                  },
                },
              };
            }
            
            return {
              sessions: {
                ...state.sessions,
                [sessionId]: {
                  ...currentSession,
                  messages: [...currentSession.messages, ...uniqueNew.map(m => ({
                    id: m.id,
                    role: m.role,
                    text: m.text,
                    metadata: m.metadata,
                    widgets: m.widgets,
                    time: m.time,
                  }))],
                  lastEventId: lastEventId || currentSession.lastEventId,
                },
              },
            };
          });
          
          logger.log(`pollSessionForNewEvents: Appended ${newMessages.length} messages to ${sessionId}`);
        } catch (error) {
          // Silent fail for polling
        }
      },

      addMessage: (sessionId, message) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: [...session.messages, message],
              },
            },
          };
        });
      },

      updateMessage: (sessionId, messageId, updates) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          const newMessages = session.messages.map((msg) =>
            msg.id === messageId ? { ...msg, ...updates } : msg,
          );

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: newMessages,
              },
            },
          };
        });
      },

      setSessionTaskId: (sessionId, taskId) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                taskId,
              },
            },
          };
        });
      },

      /**
       * Update session cursor for stateless operation
       * Called by StreamParser after each event to track progress
       */
      updateSessionCursor: (sessionId, lastEventId) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                lastEventId,
                eventsLoaded: true, // Mark as loaded since we're receiving events
              },
            },
          };
        });
      },

      // Widget state persistence
      setWidgetExpanded: (sessionId, messageId, widgetId, expanded) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          const message = session.messages.find((m) => m.id === messageId);
          if (!message) return state;

          // Create new widgets array with updated widget (immutable update)
          const updatedWidgets = message.widgets.map((w) =>
            w.id === widgetId ? { ...w, isExpanded: expanded } : w,
          );

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: session.messages.map((m) =>
                  m.id === messageId ? { ...m, widgets: updatedWidgets } : m,
                ),
              },
            },
          };
        });
      },

      getWidgetExpanded: (sessionId, messageId, widgetId) => {
        const state = get();
        const session = state.sessions[sessionId];
        if (!session) return false;

        const message = session.messages.find((m) => m.id === messageId);
        if (!message) return false;

        const widget = message.widgets.find((w) => w.id === widgetId);
        return widget?.isExpanded ?? false;
      },

      // Add a new widget to a message
      addWidget: (sessionId, messageId, widget) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          const message = session.messages.find((m) => m.id === messageId);
          if (!message) return state;

          // Check if widget already exists
          if (message.widgets.some((w) => w.id === widget.id)) {
            return state; // Don't add duplicate
          }

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: session.messages.map((m) =>
                  m.id === messageId
                    ? { ...m, widgets: [...m.widgets, widget] }
                    : m
                ),
              },
            },
          };
        });
      },

      // Update an existing widget
      updateWidget: (sessionId, messageId, widgetId, updates) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          const message = session.messages.find((m) => m.id === messageId);
          if (!message) return state;

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: session.messages.map((m) =>
                  m.id === messageId
                    ? {
                      ...m,
                      widgets: m.widgets.map((w) =>
                        w.id === widgetId
                          ? ({ ...w, ...updates } as Widget)
                          : w
                      ),
                    }
                    : m
                ),
              },
            },
          };
        });
      },

      // Add widget ID to content order for proper rendering sequence
      addToContentOrder: (sessionId, messageId, widgetId) => {
        set((state) => {
          const session = state.sessions[sessionId];
          if (!session) return state;

          const message = session.messages.find((m) => m.id === messageId);
          if (!message) return state;

          const currentOrder = message.metadata?.contentOrder || [];
          if (currentOrder.includes(widgetId)) {
            return state; // Already in order
          }

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: session.messages.map((m) =>
                  m.id === messageId
                    ? {
                      ...m,
                      metadata: {
                        ...m.metadata,
                        contentOrder: [...currentOrder, widgetId],
                      },
                    }
                    : m
                ),
              },
            },
          };
        });
      },

      appendTextWidgetContent: (_sessionId, _messageId, widgetId, textDelta) => {
        // PERFORMANCE OPTIMIZATION: During streaming, accumulate text in separate buffer
        // This prevents creating 6 new objects on every token flush (20fps)
        // The buffer is committed to the message on stream completion
        // Note: sessionId and messageId are unused during buffering but kept for interface consistency
        set((state) => {
          const currentContent = state.streamingTextContent[widgetId] || "";
          return {
            streamingTextContent: {
              ...state.streamingTextContent,
              [widgetId]: currentContent + textDelta,
            },
          };
        });
      },

      finalizeStreamingText: (sessionId, messageId, widgetId) => {
        set((state) => {
          const streamedContent = state.streamingTextContent[widgetId];
          if (!streamedContent) return state; // Nothing to finalize

          const session = state.sessions[sessionId];
          if (!session) return state;

          const msgIndex = session.messages.findIndex((m) => m.id === messageId);
          if (msgIndex === -1) return state;

          const message = session.messages[msgIndex];
          const widgetIndex = message.widgets.findIndex((w) => w.id === widgetId);
          if (widgetIndex === -1) return state;

          const widget = message.widgets[widgetIndex];
          if (widget.type !== "text") return state;

          // Commit streamed content to widget
          const newWidgets = [...message.widgets];
          newWidgets[widgetIndex] = {
            ...widget,
            content: streamedContent,
            status: "completed" as const,
          };

          const newMessages = [...session.messages];
          newMessages[msgIndex] = {
            ...message,
            text: streamedContent, // Also update message.text for compatibility
            widgets: newWidgets,
          };

          // Clear the streaming buffer
          const newStreamingContent = { ...state.streamingTextContent };
          delete newStreamingContent[widgetId];

          return {
            sessions: {
              ...state.sessions,
              [sessionId]: {
                ...session,
                messages: newMessages,
              },
            },
            streamingTextContent: newStreamingContent,
          };
        });
      },

      // Studio UI State Implementation
      studioViewMode: 'split',
      studioDesignView: 'editor',
      studioRightPanelWidth: 500,
      studioYamlContent: '',
      studioIsValidYaml: true,
      studioValidationError: '',
      studioIsDeploying: false,
      isServerStudioEnabled: true, // Default to true, updated on server connect

      setStudioViewMode: (mode) => set({ studioViewMode: mode }),
      setStudioDesignView: (view) => set({ studioDesignView: view }),
      setStudioRightPanelWidth: (width) => set({ studioRightPanelWidth: width }),
      setStudioYamlContent: (content) => set({ studioYamlContent: content }),
      setStudioValidationStatus: (isValid, error) => set({ 
        studioIsValidYaml: isValid, 
        studioValidationError: error || '' 
      }),
      setStudioIsDeploying: (isDeploying) => set({ studioIsDeploying: isDeploying }),
      setIsServerStudioEnabled: (enabled) => set({ isServerStudioEnabled: enabled }),

      // RAG Indexing Status
      ragStatus: null,
      setRagStatus: (status) => set({ ragStatus: status }),

      _hasHydrated: false,
      setHasHydrated: (state) => set({ _hasHydrated: state }),
    }),
    {
      name: "hector_sessions",
      storage: createJSONStorage(() => ({
        getItem: (name: string) => {
          try {
            return localStorage.getItem(name);
          } catch (error) {
            logger.error("Failed to read from localStorage:", error);
            return null;
          }
        },
        setItem: (() => {
          // Debounce writes to prevent freezing UI during high-frequency streaming
          let timeoutId: any = null;
          let pendingKey: string | null = null;
          let pendingValue: string | null = null;

          return (name: string, value: string) => {
            pendingKey = name;
            pendingValue = value;

            if (timeoutId) return;

            timeoutId = setTimeout(() => {
              timeoutId = null;
              if (!pendingKey || !pendingValue) return;

              const key = pendingKey;
              const val = pendingValue;
              pendingKey = null;
              pendingValue = null;

              const MAX_RETRIES = 1;
              let retryCount = 0;

              const attemptSave = (): void => {
                try {
                  localStorage.setItem(key, val);
                } catch (error) {
                  // Handle quota exceeded error
                  if (
                    error instanceof DOMException &&
                    (error.code === 22 || // Legacy quota exceeded
                      error.code === 1014 || // Firefox
                      error.name === "QuotaExceededError" ||
                      error.name === "NS_ERROR_DOM_QUOTA_REACHED")
                  ) {
                    if (retryCount < MAX_RETRIES) {
                      retryCount++;
                      logger.warn(
                        `localStorage quota exceeded (attempt ${retryCount}/${MAX_RETRIES}), clearing old sessions`,
                      );
                      try {
                        localStorage.removeItem(key);
                        attemptSave(); // Recursive retry
                      } catch (retryError) {
                        logger.error(
                          "Failed to save to localStorage even after clearing:",
                          retryError,
                        );
                        // Gracefully degrade - app continues without persistence
                      }
                    } else {
                      logger.error(
                        "localStorage quota exceeded and max retries reached. Persistence disabled.",
                      );
                      // Notify user about storage issue
                      useStore.getState().setError(
                        "Chat history storage is full. Some conversations may not be saved. Please clear old sessions or browser data."
                      );
                    }
                  } else {
                    logger.error("Failed to write to localStorage:", error);
                  }
                }
              };

              attemptSave();
            }, 1000); // 1 second debounce
          };
        })(),
        removeItem: (name: string) => {
          try {
            localStorage.removeItem(name);
          } catch (error) {
            logger.error("Failed to remove from localStorage:", error);
          }
        },
      })),
      partialize: (state) => ({
        // STATELESS SESSION ARCHITECTURE:
        // Sessions are ephemeral - backend is single source of truth
        // Only persist session metadata (id, title, serverId, agentName, cursor)
        // Messages/widgets are NOT persisted - they are fetched from backend
        sessions: Object.fromEntries(
          Object.entries(state.sessions).map(([id, session]) => [
            id,
            {
              id: session.id,
              title: session.title,
              created: session.created,
              contextId: session.contextId,
              taskId: session.taskId,
              serverId: session.serverId,
              appId: session.appId,
              agentName: session.agentName,
              // Cursor for efficient resume (not full events)
              lastEventId: session.lastEventId,
              // Empty messages - will be loaded from backend
              messages: [],
              eventsLoaded: false,
            },
          ])
        ),
        currentSessionId: state.currentSessionId,
        sidebarVisible: state.sidebarVisible,
        minimalMode: state.minimalMode,
        endpointUrl: state.endpointUrl,
        protocol: state.protocol,
        streamingEnabled: state.streamingEnabled,
        selectedAgentName: state.selectedAgent?.name || null, // Persist agent selection
        studioViewMode: state.studioViewMode,
        studioDesignView: state.studioDesignView,
        studioRightPanelWidth: state.studioRightPanelWidth,
      }),
      onRehydrateStorage: () => (state) => {
        if (!state) {
           return;
        }
        // Backfill appId on legacy sessions that have undefined
        const sessions = state.sessions;
        for (const id in sessions) {
          if (!sessions[id].appId) {
            sessions[id] = { ...sessions[id], appId: 'default' };
          }
        }
        // State successfully rehydrated (or using defaults after error)
        state.setHasHydrated(true);
      },
    },
  ),
);
