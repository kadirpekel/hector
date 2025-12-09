import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { api } from "../services/api";
import { v4 as uuidv4 } from "uuid";
import type { Session, Message, Agent, AgentCard, Widget } from "../types";
import { StreamParser as StreamParserClass } from "../lib/stream-parser";
import { DEFAULT_SUPPORTED_FILE_TYPES } from "../lib/constants";
import { logger } from "../lib/logger";

type StreamParser = StreamParserClass;

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
  configVisible: boolean;
  setConfigVisible: (visible: boolean) => void;
  isGenerating: boolean;
  setIsGenerating: (generating: boolean) => void;
  error: string | null;
  setError: (error: string | null) => void;
  successMessage: string | null;
  setSuccessMessage: (message: string | null) => void;

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
  currentSessionId: string | null;
  availableAgents: Agent[];
  selectedAgent: Agent | null;
  agentCard: AgentCard | null;
  supportedFileTypes: string[];
  selectedNodeId: string | null;
  schema: any;
  activeAgentId: string | null;
  agentsLoaded: boolean;

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
  selectSession: (sessionId: string) => void;
  deleteSession: (sessionId: string) => void;
  updateSessionTitle: (sessionId: string, title: string) => void;

  addMessage: (sessionId: string, message: Message) => void;
  updateMessage: (
    sessionId: string,
    messageId: string,
    updates: Partial<Message>,
  ) => void;
  setSessionTaskId: (sessionId: string, taskId: string | null) => void;

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
}

export const useStore = create<AppState>()(
  persist(
    (set, get) => ({
      sidebarVisible: true,
      setSidebarVisible: (visible) => set({ sidebarVisible: visible }),
      minimalMode: false,
      setMinimalMode: (enabled) =>
        set({ minimalMode: enabled, sidebarVisible: !enabled }),
      configVisible: false,
      setConfigVisible: (visible) => set({ configVisible: visible }),
      isGenerating: false,
      setIsGenerating: (generating) => set({ isGenerating: generating }),
      error: null,
      setError: (error) => set({ error }),
      successMessage: null,
      setSuccessMessage: (successMessage) => set({ successMessage }),

      // Config defaults
      endpointUrl: typeof window !== "undefined" ? window.location.origin : "",
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
      currentSessionId: null,
      availableAgents: [],
      selectedAgent: null,
      selectedNodeId: null,
      agentCard: null,
      supportedFileTypes: [...DEFAULT_SUPPORTED_FILE_TYPES],
      schema: null,
      activeAgentId: null,
      agentsLoaded: false,

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
          if (persistedName && !get().selectedAgent) {
            const restoredAgent = agents.find((a) => a.name === persistedName);
            if (restoredAgent) {
              logger.log(
                `Restored agent selection from localStorage: ${persistedName}`,
              );
              set({ selectedAgent: restoredAgent });
            } else {
              logger.log(
                `Persisted agent "${persistedName}" not found, will select first available`,
              );
            }
          }
        } catch (e) {
          logger.error("Failed to load agents", e);
          set({ error: "Failed to load agents. Please check connection." });
        }
      },
      reloadAgents: async () => {
        logger.log("Forcing agent reload (deploy/config change)");
        // Reset agentsLoaded to bypass idempotency guard
        set({ agentsLoaded: false });
        // Call loadAgents which will now run
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
        const newSession: Session = {
          id,
          title: "New conversation",
          created: new Date().toISOString(),
          messages: [],
          contextId: `hector-web-${uuidv4()}`,
          taskId: null,
        };

        set((state) => ({
          sessions: { ...state.sessions, [id]: newSession },
          currentSessionId: id,
        }));

        return id;
      },

      selectSession: (sessionId) => set({ currentSessionId: sessionId }),

      deleteSession: (sessionId) => {
        set((state) => {
          const newSessions = { ...state.sessions };
          delete newSessions[sessionId];

          let newCurrentId = state.currentSessionId;
          if (state.currentSessionId === sessionId) {
            const remainingIds = Object.keys(newSessions);
            newCurrentId = remainingIds.length > 0 ? remainingIds[0] : null;
          }

          return {
            sessions: newSessions,
            currentSessionId: newCurrentId,
          };
        });
      },

      updateSessionTitle: (sessionId, title) => {
        set((state) => ({
          sessions: {
            ...state.sessions,
            [sessionId]: { ...state.sessions[sessionId], title },
          },
        }));
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
        setItem: (name: string, value: string) => {
          const MAX_RETRIES = 1;
          let retryCount = 0;

          const attemptSave = (): void => {
            try {
              localStorage.setItem(name, value);
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
                    localStorage.removeItem(name);
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
                  // Gracefully degrade - app continues without persistence
                }
              } else {
                logger.error("Failed to write to localStorage:", error);
              }
            }
          };

          attemptSave();
        },
        removeItem: (name: string) => {
          try {
            localStorage.removeItem(name);
          } catch (error) {
            logger.error("Failed to remove from localStorage:", error);
          }
        },
      })),
      partialize: (state) => ({
        sessions: state.sessions,
        currentSessionId: state.currentSessionId,
        sidebarVisible: state.sidebarVisible,
        minimalMode: state.minimalMode,
        endpointUrl: state.endpointUrl,
        protocol: state.protocol,
        streamingEnabled: state.streamingEnabled,
        selectedAgentName: state.selectedAgent?.name || null, // Persist agent selection
      }),
      onRehydrateStorage: () => (_state, error) => {
        if (error) {
          logger.error("Failed to rehydrate state from localStorage:", error);
          // State will be initialized with defaults
        }
        // State successfully rehydrated (or using defaults after error)
      },
    },
  ),
);
