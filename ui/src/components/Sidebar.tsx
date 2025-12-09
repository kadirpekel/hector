import React from 'react';
import { Plus, MessageSquare, Settings, ChevronDown, Bot, X } from 'lucide-react';
import { useStore } from '../store/useStore';
import { cn, formatTime } from '../lib/utils';
import { DeleteButton } from './DeleteButton';
import { useAgentSelection } from '../lib/hooks/useAgentSelection';

export const Sidebar: React.FC = () => {
    // Use selectors for better performance - only subscribe to specific state slices
    const sessions = useStore((state) => state.sessions);
    const currentSessionId = useStore((state) => state.currentSessionId);
    const availableAgents = useStore((state) => state.availableAgents);
    const setSidebarVisible = useStore((state) => state.setSidebarVisible);
    const createSession = useStore((state) => state.createSession);
    const selectSession = useStore((state) => state.selectSession);
    const deleteSession = useStore((state) => state.deleteSession);
    const selectedAgent = useStore((state) => state.selectedAgent);

    // Use shared agent selection hook
    const { handleAgentChange } = useAgentSelection();

    // Note: Agent loading is now centralized in App.tsx
    // Removed duplicate loadAgents() call to prevent wasteful network requests
    // Agent selection logic is now in useAgentSelection hook

    const sortedSessions = Object.values(sessions).sort(
        (a, b) => new Date(b.created).getTime() - new Date(a.created).getTime()
    );

    return (
        <div className="w-64 bg-black/40 border-r border-white/10 flex flex-col h-full backdrop-blur-md flex-shrink-0 transition-all duration-300">
            {/* Header */}
            <div className="p-4 border-b border-white/10">
                <div className="flex items-center justify-between mb-4">
                    <span className="text-xs text-gray-500 uppercase tracking-wider">Conversations</span>
                    <button
                        onClick={() => setSidebarVisible(false)}
                        className="hidden md:flex p-1 hover:bg-white/10 rounded transition-colors text-gray-400 hover:text-white"
                        title="Hide Sidebar"
                    >
                        <X size={18} />
                    </button>
                </div>
                <button
                    onClick={createSession}
                    className="w-full bg-white/5 hover:bg-white/10 border border-white/10 rounded-lg px-3 py-2 flex items-center justify-center gap-2 transition-all text-sm text-gray-300 hover:text-white"
                >
                    <Plus size={16} />
                    <span>New chat</span>
                </button>
            </div>

            {/* Session List */}
            <div className="flex-1 overflow-y-auto p-2 space-y-1 custom-scrollbar">
                {sortedSessions.length === 0 ? (
                    <div className="text-center text-gray-500 mt-10 text-sm">
                        No conversations yet
                    </div>
                ) : (
                    sortedSessions.map((session) => (
                        <div
                            key={session.id}
                            className={cn(
                                "group flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors text-sm",
                                currentSessionId === session.id
                                    ? "bg-white/10 text-white"
                                    : "text-gray-400 hover:bg-white/5 hover:text-gray-200"
                            )}
                            onClick={() => selectSession(session.id)}
                        >
                            <MessageSquare size={16} className="flex-shrink-0" />
                            <div className="flex-1 min-w-0">
                                <div className="truncate font-medium">{session.title}</div>
                                <div className="text-xs text-gray-500">{formatTime(session.created)}</div>
                            </div>
                            <DeleteButton
                                onDelete={() => deleteSession(session.id)}
                                sessionTitle={session.title}
                            />
                        </div>
                    ))
                )}
            </div>

            {/* Footer / Agent Selection */}
            <div className="p-4 border-t border-white/10 bg-black/20">
                <div className="mb-4">
                    <label className="text-xs text-gray-500 uppercase font-bold tracking-wider mb-2 flex items-center gap-2">
                        <Bot size={12} className="text-hector-green" />
                        Active Agent
                    </label>
                    <div className="relative">
                        <select
                            className="w-full bg-black/50 border border-white/10 rounded-lg p-2 text-sm text-gray-300 appearance-none focus:outline-none focus:border-hector-green"
                            onChange={handleAgentChange}
                            value={selectedAgent?.name || ''}
                        >
                            {availableAgents.length === 0 ? (
                                <option value="">Loading agents...</option>
                            ) : (
                                availableAgents.map((agent) => (
                                    <option key={agent.name} value={agent.name}>
                                        {agent.name}
                                    </option>
                                ))
                            )}
                        </select>
                        <ChevronDown size={14} className="absolute right-3 top-3 text-gray-500 pointer-events-none" />
                    </div>
                    {selectedAgent && (
                        <div className="mt-2 text-xs text-gray-500 truncate">
                            {selectedAgent.description || 'Ready to help'}
                        </div>
                    )}
                </div>

                <div className="flex items-center justify-between text-xs text-gray-600 pt-2 border-t border-white/5">
                    <span>A2A v0.3.0</span>
                    <button
                        className="hover:text-gray-400 transition-colors"
                        onClick={() => useStore.getState().setConfigVisible(true)}
                        title="Open settings"
                    >
                        <Settings size={14} />
                    </button>
                </div>
            </div>
        </div>
    );
};
