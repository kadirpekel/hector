import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { v4 as uuidv4 } from 'uuid';
import type { Session, Message, Agent, AgentCard } from '../types';
import { StreamParser as StreamParserClass } from '../lib/stream-parser';

type StreamParser = StreamParserClass;

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

    // Config State
    endpointUrl: string;
    setEndpointUrl: (url: string) => void;
    protocol: 'jsonrpc' | 'rest';
    setProtocol: (protocol: 'jsonrpc' | 'rest') => void;
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

    // Actions
    setAvailableAgents: (agents: Agent[]) => void;
    setSelectedAgent: (agent: Agent | null) => void;
    setAgentCard: (card: AgentCard | null) => void;
    setSupportedFileTypes: (types: string[]) => void;

    createSession: () => string;
    selectSession: (sessionId: string) => void;
    deleteSession: (sessionId: string) => void;
    updateSessionTitle: (sessionId: string, title: string) => void;

    addMessage: (sessionId: string, message: Message) => void;
    updateMessage: (sessionId: string, messageId: string, updates: Partial<Message>) => void;
    setSessionTaskId: (sessionId: string, taskId: string | null) => void;
    
    // Widget state persistence
    setWidgetExpanded: (sessionId: string, messageId: string, widgetId: string, expanded: boolean) => void;
    getWidgetExpanded: (sessionId: string, messageId: string, widgetId: string) => boolean;
}

export const useStore = create<AppState>()(
    persist(
        (set, get) => ({
            sidebarVisible: true,
            setSidebarVisible: (visible) => set({ sidebarVisible: visible }),
            minimalMode: false,
            setMinimalMode: (enabled) => set({ minimalMode: enabled, sidebarVisible: !enabled }),
            configVisible: false,
            setConfigVisible: (visible) => set({ configVisible: visible }),
            isGenerating: false,
            setIsGenerating: (generating) => set({ isGenerating: generating }),
            error: null,
            setError: (error) => set({ error }),

            // Config defaults
            endpointUrl: typeof window !== 'undefined' ? window.location.origin : '',
            setEndpointUrl: (url) => set({ endpointUrl: url }),
            protocol: 'jsonrpc' as const,
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
                                if (session.messages[i].role === 'agent') {
                                    state.updateMessage(state.currentSessionId, session.messages[i].id, { cancelled: true });
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
            agentCard: null,
            supportedFileTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],

            setAvailableAgents: (agents) => set({ availableAgents: agents }),
            setSelectedAgent: (agent) => set({ selectedAgent: agent }),
            setAgentCard: (card) => {
                set({ agentCard: card });
                // Update supported file types from agent card
                if (card && card.default_input_modes && Array.isArray(card.default_input_modes)) {
                    // Filter to only file/media types (exclude text/plain, application/json)
                    const fileTypes = card.default_input_modes.filter((mode: string) => 
                        mode.startsWith('image/') || 
                        mode.startsWith('video/') || 
                        mode.startsWith('audio/')
                    );
                    
                    // If no file types found, fall back to image defaults
                    if (fileTypes.length === 0) {
                        set({ supportedFileTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'] });
                    } else {
                        set({ supportedFileTypes: fileTypes });
                    }
                } else {
                    // No input modes specified, use defaults
                    set({ supportedFileTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'] });
                }
            },
            setSupportedFileTypes: (types) => set({ supportedFileTypes: types }),

            createSession: () => {
                const id = `session-${uuidv4()}`;
                const newSession: Session = {
                    id,
                    title: 'New conversation',
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
                        msg.id === messageId ? { ...msg, ...updates } : msg
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

                    const message = session.messages.find(m => m.id === messageId);
                    if (!message) return state;

                    const widget = message.widgets.find(w => w.id === widgetId);
                    if (widget) {
                        widget.isExpanded = expanded;
                    }

                    return {
                        sessions: {
                            ...state.sessions,
                            [sessionId]: {
                                ...session,
                                messages: session.messages.map(m =>
                                    m.id === messageId ? { ...m, widgets: [...m.widgets] } : m
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

                const message = session.messages.find(m => m.id === messageId);
                if (!message) return false;

                const widget = message.widgets.find(w => w.id === widgetId);
                return widget?.isExpanded ?? false;
            },
        }),
        {
            name: 'hector_sessions',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                sessions: state.sessions,
                currentSessionId: state.currentSessionId,
                sidebarVisible: state.sidebarVisible,
                minimalMode: state.minimalMode,
                endpointUrl: state.endpointUrl,
                protocol: state.protocol,
                streamingEnabled: state.streamingEnabled,
            }),
            // Custom storage handling to avoid quota errors (simplified version of original)
            onRehydrateStorage: () => (state) => {
                if (state) {
                    // Clean up any potential issues on load
                    console.log('State rehydrated');
                }
            },
        }
    )
);
