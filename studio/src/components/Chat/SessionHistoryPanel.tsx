/**
 * SessionHistoryPanel - Shows chat sessions for the current server only.
 * 
 * Simplified design:
 * - Only shows sessions belonging to the active server
 * - No grouping by server
 * - Includes "Clear All" button to wipe all sessions for current server
 */

import React from "react";
import { useStore } from "../../store/useStore";
import { useServersStore } from "../../store/serversStore";
import { useAppsStore } from "../../store/appsStore";
import { cn } from "../../lib/utils";
import { MessageSquare, Trash2, Pin, PinOff, X, Plus, Eye, RefreshCw, Loader2 } from "lucide-react";
import { DeleteButton } from "../DeleteButton";

interface SessionMeta {
    id: string;
    title: string;
    created: string;
    serverId?: string;
    appId?: string;
    agentName?: string;
}

interface SessionHistoryPanelProps {
    isPinned: boolean;
    onTogglePin: () => void;
    onClose: () => void;
}

export const SessionHistoryPanel: React.FC<SessionHistoryPanelProps> = ({
    isPinned,
    onTogglePin,
    onClose,
}) => {
    // Selectors
    const sessions = useStore((state) => state.sessions);
    const currentSessionId = useStore((state) => state.currentSessionId);
    const createSession = useStore((state) => state.createSession);
    const selectSession = useStore((state) => state.selectSession);
    const deleteSession = useStore((state) => state.deleteSession);
    const deleteAllSessionsForServer = useStore((state) => state.deleteAllSessionsForServer);

    const activeServerId = useServersStore((state) => state.activeServerId);
    const activeAppId = useAppsStore((state) =>
        activeServerId ? state.appsByServer[activeServerId]?.activeAppId : null
    );
    const selectedAgentName = useStore((state) => state.selectedAgent?.name);



    const setXRaySessionId = useStore((state) => state.setXRaySessionId);

    // Session sync
    const syncSessions = useStore((state) => state.syncSessions);
    const sessionsLoading = useStore((state) => state.sessionsLoading);

    // Sync sessions on mount and when server/agent/app changes
    React.useEffect(() => {
        syncSessions();
    }, [activeServerId, activeAppId, selectedAgentName, syncSessions]);

    // Filter sessions for current server, app, AND agent
    const effectiveAppId = activeAppId || 'default';
    const filteredSessions = React.useMemo(() => {
        const serverSessions: SessionMeta[] = [];
        Object.values(sessions).forEach((session) => {
            // Only include sessions for the active server
            const isServerMatch = (session.serverId || null) === (activeServerId || null);
            const isAppMatch = session.appId === effectiveAppId;
            // Include sessions for current agent OR unassigned sessions (from backend sync)
            const isAgentMatch = session.agentName === selectedAgentName || !session.agentName;
            if (isServerMatch && isAppMatch && isAgentMatch) {
                serverSessions.push(session);
            }
        });
        // Sort by creation date (newest first)
        serverSessions.sort((a, b) =>
            new Date(b.created).getTime() - new Date(a.created).getTime()
        );
        return serverSessions;
    }, [sessions, activeServerId, effectiveAppId, selectedAgentName]);

    const handleSessionClick = (session: SessionMeta) => {
        selectSession(session.id);
        // Only load events for sessions that haven't been loaded yet
        const fullSession = useStore.getState().sessions[session.id];
        if (fullSession && fullSession.messages.length === 0) {
            useStore.getState().loadSessionEvents(session.id);
        }
        if (!isPinned) onClose();
    };

    const handleClearAll = () => {
        if (!activeServerId) return;
        if (confirm(`Delete ALL chats for this server? This cannot be undone.`)) {
            deleteAllSessionsForServer(activeServerId);
        }
    };

    return (
        <div className={cn(
            "flex flex-col bg-black/80 backdrop-blur-xl border-l border-white/10 transition-all duration-300 z-20",
            isPinned ? "w-64 relative h-full flex-shrink-0" : "fixed top-14 right-4 w-72 h-[calc(100vh-100px)] rounded-xl shadow-2xl border border-white/10"
        )}>
            {/* Header */}
            <div className="p-3 border-b border-white/10 flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-wider text-gray-400">History</span>
                <div className="flex items-center gap-1">
                    <button
                        onClick={createSession}
                        className="p-1.5 hover:bg-white/10 rounded text-gray-400 hover:text-white transition-colors"
                        title="New Chat"
                    >
                        <Plus size={14} />
                    </button>
                    <button
                        onClick={() => syncSessions()}
                        disabled={sessionsLoading}
                        className={cn(
                            "p-1.5 hover:bg-white/10 rounded transition-colors",
                            sessionsLoading ? "text-hector-green animate-spin" : "text-gray-400 hover:text-white"
                        )}
                        title="Sync Sessions"
                    >
                        {sessionsLoading ? <Loader2 size={14} /> : <RefreshCw size={14} />}
                    </button>
                    <button
                        onClick={onTogglePin}
                        className={cn(
                            "p-1.5 hover:bg-white/10 rounded transition-colors",
                            isPinned ? "text-hector-green bg-white/5" : "text-gray-400 hover:text-white"
                        )}
                        title={isPinned ? "Unpin Panel" : "Pin Panel"}
                    >
                        {isPinned ? <PinOff size={14} /> : <Pin size={14} />}
                    </button>
                    {!isPinned && (
                        <button
                            onClick={onClose}
                            className="p-1.5 hover:bg-white/10 rounded text-gray-400 hover:text-white transition-colors"
                        >
                            <X size={14} />
                        </button>
                    )}
                </div>
            </div>

            {/* Session List */}
            <div className="flex-1 overflow-y-auto p-2 custom-scrollbar">
                {filteredSessions.length === 0 ? (
                    <div className="text-center text-gray-500 mt-10 text-sm">No chats yet</div>
                ) : (
                    <div className="space-y-0.5">
                        {filteredSessions.map(session => (
                            <div
                                key={session.id}
                                onClick={() => handleSessionClick(session)}
                                className={cn(
                                    "group flex items-center gap-2 p-2 rounded-md cursor-pointer transition-colors text-sm",
                                    currentSessionId === session.id ? "bg-white/10 text-white" : "text-gray-400 hover:bg-white/5 hover:text-gray-200"
                                )}
                            >
                                <MessageSquare size={12} className="shrink-0" />
                                <div className="flex-1 min-w-0">
                                    <div className="truncate text-xs">{session.title}</div>
                                </div>
                                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                    <button
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            setXRaySessionId(session.id);
                                        }}
                                        className="p-1 hover:text-white text-gray-400 rounded hover:bg-white/10"
                                        title="Inspect Session Events"
                                    >
                                        <Eye size={12} />
                                    </button>
                                    <DeleteButton onDelete={() => deleteSession(session.id)} sessionTitle={session.title} />
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Footer with Clear All */}
            {filteredSessions.length > 0 && (
                <div className="p-2 border-t border-white/10">
                    <button
                        onClick={handleClearAll}
                        className="w-full flex items-center justify-center gap-2 p-2 text-xs text-gray-500 hover:text-red-400 hover:bg-red-500/10 rounded-md transition-colors"
                    >
                        <Trash2 size={12} />
                        <span>Clear All Chats</span>
                    </button>
                </div>
            )}


        </div>
    );
};
