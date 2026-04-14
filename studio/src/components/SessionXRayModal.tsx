import React, { useEffect, useState, Component, ErrorInfo, ReactNode } from 'react';
import { X, Clock, User, Cpu, Shield, Database, Activity, MessageSquarePlus, Code, Eye, AlertTriangle } from 'lucide-react';
import { api } from '../services/api';
import { useServersStore } from '../store/serversStore';
import { useAppsStore } from '../store/appsStore';
import { useStore } from '../store/useStore';
import { SessionEvent } from '../types';
import { cn } from '../lib/utils';
import { getAgentBranding } from '../lib/colors';

// Error Boundary for catching render errors in event content
interface ErrorBoundaryState { hasError: boolean; error?: Error; }
class EventContentErrorBoundary extends Component<{ children: ReactNode; fallback?: ReactNode }, ErrorBoundaryState> {
    state: ErrorBoundaryState = { hasError: false };
    static getDerivedStateFromError(error: Error) { return { hasError: true, error }; }
    componentDidCatch(error: Error, info: ErrorInfo) { console.error('SessionXRay render error:', error, info); }
    render() {
        if (this.state.hasError) {
            return this.props.fallback || (
                <div className="p-4 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm flex items-center gap-2">
                    <AlertTriangle size={16} />
                    <span>Failed to render event content. Try switching to Raw JSON view.</span>
                </div>
            );
        }
        return this.props.children;
    }
}

interface SessionXRayModalProps {
    sessionId: string;
    onClose: () => void;
}

export const SessionXRayModal: React.FC<SessionXRayModalProps> = ({ sessionId, onClose }) => {
    const [events, setEvents] = useState<SessionEvent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
    const [viewMode, setViewMode] = useState<'visual' | 'raw'>('visual');

    useEffect(() => {
        loadEvents();
    }, [sessionId]);

    const loadEvents = async () => {
        try {
            setLoading(true);
            // Prefer the session's own appId (set when synced from backend)
            const session = useStore.getState().sessions[sessionId];
            const serverId = useServersStore.getState().activeServerId;
            const appId = session?.appId
                || (serverId ? useAppsStore.getState().getActiveAppId(serverId) : null)
                || 'default';
            // Fetch events - backend returns native A2A format (lowercase keys)
            const { events: fetchedEvents } = await api.fetchSessionEvents(sessionId, { limit: 500, appId });
            setEvents(fetchedEvents);
            if (fetchedEvents.length > 0 && !selectedEventId) {
                setSelectedEventId((fetchedEvents[0] as any).id);
            }
        } catch (err: any) {
            setError(err.message || 'Failed to load events');
        } finally {
            setLoading(false);
        }
    };

    // Returns the appropriate icon for an event type (color is applied externally via getEventStyle)
    const getEventIcon = (event: SessionEvent) => {
        if (event.metadata?.type === 'guardrail' || event.metadata?.type === 'moderation') {
            return <Shield size={16} />;
        }
        if (event.metadata?.tool_calls && event.metadata.tool_calls.length > 0) {
            return <Cpu size={16} />;
        }
        if (event.metadata?.tool_results && event.metadata.tool_results.length > 0) {
            return <Database size={16} />;
        }
        if (event.metadata?.thinking) {
            return <Activity size={16} />;
        }
        return event.author === 'user' ? <User size={16} /> : <Cpu size={16} />;
    };

    const formatTime = (ts: string) => {
        return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3 });
    };

    const getDuration = (currentTs: string, prevTs?: string) => {
        if (!prevTs) return null;
        const diff = new Date(currentTs).getTime() - new Date(prevTs).getTime();
        if (diff < 1000) return `+${diff}ms`;
        return `+${(diff / 1000).toFixed(2)}s`;
    };

    // Centralized style helper - ensures consistent colors everywhere
    const getEventStyle = (event: SessionEvent) => {
        if (event.author === 'user') {
            return { bg: "bg-slate-500", text: "text-slate-300", iconColor: "text-slate-300", border: "border-slate-500/30", ring: "" };
        }
        if (event.metadata?.type === 'guardrail') {
            return { bg: "bg-red-500", text: "text-red-400", iconColor: "text-red-400", border: "border-red-500/30", ring: "" };
        }
        // Use Type from metadata or fallback
        const branding = getAgentBranding({ name: event.author, type: event.metadata?.agentType });
        return { bg: branding.classes.bg, text: branding.classes.text, iconColor: branding.classes.text, border: branding.classes.border, ring: branding.classes.ring };
    };

    const renderEventContent = (event: SessionEvent) => {
        if (viewMode === 'raw') {
            return (
                <div className="bg-black/30 rounded-xl border border-white/10 overflow-hidden">
                    <div className="px-4 py-2 border-b border-white/10 bg-white/5 text-xs font-bold text-gray-400 uppercase flex justify-between items-center">
                        <span>Raw JSON Payload</span>
                        <div className="flex gap-2">
                            <button onClick={() => navigator.clipboard.writeText(JSON.stringify(event, null, 2))} className="hover:text-white transition-colors">Copy</button>
                        </div>
                    </div>
                    <div className="p-4 overflow-x-auto">
                        <pre className="text-xs text-green-300 font-mono">
                            {JSON.stringify(event, null, 2)}
                        </pre>
                    </div>
                </div>
            );
        }

        // Visual Mode - native A2A format
        const hasTextContent = event.artifact?.parts?.some(p => p.kind === 'text' || p.type === 'text');
        const toolCalls = event.metadata?.tool_calls;
        const toolResults = event.metadata?.tool_results;
        const thinking = event.metadata?.thinking;

        return (
            <div className="space-y-6">
                {/* Artifact Content (Text) */}
                {hasTextContent && (
                    <div className="space-y-2">
                        <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Content</div>
                        <div className="p-4 bg-white/5 rounded-lg border border-white/10 text-sm leading-relaxed text-gray-200 whitespace-pre-wrap font-sans">
                            {event.artifact?.parts?.map((p, i) => (
                                <span key={i}>{p.text || (p.data ? JSON.stringify(p.data) : '')}</span>
                            )) || <span className="text-gray-500 italic">(Empty)</span>}
                        </div>
                    </div>
                )}

                {/* Thinking */}
                {thinking && (
                    <div className="space-y-2">
                        <div className="text-xs font-bold text-yellow-400 uppercase tracking-wider flex items-center gap-2">
                            <Activity size={12} /> Thinking ({thinking.status || 'unknown'})
                        </div>
                        <div className="p-4 bg-yellow-500/5 rounded-lg border border-yellow-500/20 text-sm text-yellow-200/80 whitespace-pre-wrap">
                            {thinking.content || <span className="italic">(No content)</span>}
                        </div>
                    </div>
                )}

                {/* Tool Calls */}
                {toolCalls?.map((tool, i) => (
                    <div key={i} className="space-y-2">
                        <div className="text-xs font-bold text-blue-400 uppercase tracking-wider flex items-center gap-2">
                            <Cpu size={12} /> Tool Call: {tool.name}
                        </div>
                        <div className="p-4 bg-blue-500/5 rounded-lg border border-blue-500/20 font-mono text-xs">
                            <div className="text-gray-400 mb-1">// Arguments</div>
                            <pre className="text-blue-300 overflow-x-auto">{JSON.stringify(tool.args, null, 2)}</pre>
                        </div>
                    </div>
                ))}

                {/* Tool Results */}
                {toolResults?.map((res, i) => (
                    <div key={i} className="space-y-2">
                        <div className="text-xs font-bold text-purple-400 uppercase tracking-wider flex items-center gap-2">
                            <Database size={12} /> Tool Result ({res.status || 'done'})
                        </div>
                        <div className="p-4 bg-purple-500/5 rounded-lg border border-purple-500/20 font-mono text-xs">
                            <div className="text-gray-400 mb-1">// Output</div>
                            <pre className="text-purple-300 overflow-x-auto whitespace-pre-wrap max-h-60 overflow-y-auto custom-scrollbar">
                                {res.content}
                            </pre>
                        </div>
                    </div>
                ))}

                {/* Fallback if nothing specific to render */}
                {!hasTextContent && !toolCalls && !toolResults && !thinking && (
                    <div className="p-8 text-center text-gray-500 text-sm italic">
                        No distinct visual content. Switch to Raw JSON view to inspect metadata.
                    </div>
                )}
            </div>
        );
    };

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-8 animate-in fade-in duration-200">
            <div className="bg-[#1e1e1e] w-full max-w-6xl h-[85vh] rounded-xl border border-white/10 shadow-2xl flex flex-col overflow-hidden">
                {/* Header */}
                <div className="p-4 border-b border-white/10 flex justify-between items-center bg-black/20">
                    <div>
                        <h2 className="text-lg font-bold text-white flex items-center gap-2">
                            <Activity className="text-hector-green" />
                            Session X-Ray
                        </h2>
                        <div className="text-xs text-gray-500 font-mono mt-1 flex items-center gap-2">
                            <span>{sessionId}</span>
                            <span className="w-1 h-1 bg-gray-600 rounded-full"></span>
                            <span>{events.length} Events</span>
                        </div>
                    </div>
                    <div className="flex items-center gap-4">
                        <div className="flex bg-black/40 rounded-lg p-1 border border-white/5">
                            <button
                                onClick={() => setViewMode('visual')}
                                className={cn(
                                    "px-3 py-1.5 rounded-md text-xs font-medium transition-all flex items-center gap-2",
                                    viewMode === 'visual' ? "bg-white/10 text-white shadow-sm" : "text-gray-500 hover:text-gray-300"
                                )}
                            >
                                <Eye size={12} /> Visual
                            </button>
                            <button
                                onClick={() => setViewMode('raw')}
                                className={cn(
                                    "px-3 py-1.5 rounded-md text-xs font-medium transition-all flex items-center gap-2",
                                    viewMode === 'raw' ? "bg-white/10 text-white shadow-sm" : "text-gray-500 hover:text-gray-300"
                                )}
                            >
                                <Code size={12} /> Raw JSON
                            </button>
                        </div>
                        <div className="w-px h-6 bg-white/10 mx-2"></div>
                        <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-full transition-colors group">
                            <X size={20} className="text-gray-400 group-hover:text-white" />
                        </button>
                    </div>
                </div>

                {/* Content */}
                <div className="flex-1 flex overflow-hidden">
                    {loading ? (
                        <div className="flex-1 flex items-center justify-center text-gray-400 gap-2">
                            <Clock className="animate-spin" /> Loading timeline...
                        </div>
                    ) : error ? (
                        error.includes("404") ? (
                            <div className="flex-1 flex flex-col items-center justify-center text-gray-400 gap-3">
                                <div className="p-4 bg-white/5 rounded-full border border-white/10">
                                    <MessageSquarePlus size={32} className="opacity-50" />
                                </div>
                                <div className="text-center">
                                    <h3 className="font-medium text-white mb-1">Session Not Started</h3>
                                    <p className="text-sm max-w-xs mx-auto opacity-70">
                                        Send a message in the chat to initialize the session history X-Ray.
                                    </p>
                                </div>
                            </div>
                        ) : (
                            <div className="flex-1 flex items-center justify-center text-red-400 gap-2">
                                <Shield /> {error}
                            </div>
                        )
                    ) : (
                        <>
                            {/* Timeline List (Left) */}
                            <div className="w-80 border-r border-white/10 overflow-y-auto custom-scrollbar bg-black/10 flex-shrink-0">
                                {events.map((event, idx) => {
                                    const isActive = selectedEventId === event.id;
                                    const duration = getDuration(event.timestamp, events[idx - 1]?.timestamp);
                                    const styles = getEventStyle(event);
                                    const textContent = event.artifact?.parts?.find(p => p.kind === 'text' || p.type === 'text')?.text;

                                    return (
                                        <div
                                            key={event.id || idx}
                                            onClick={() => setSelectedEventId(event.id)}
                                            className={cn(
                                                "relative p-3 border-b border-white/5 cursor-pointer transition-all group",
                                                isActive ? "bg-white/5" : "hover:bg-white/[0.02]"
                                            )}
                                        >
                                            {/* Active Indicator Bar */}
                                            {isActive && <div className={cn("absolute left-0 top-0 bottom-0 w-0.5 shadow-[0_0_10px_rgba(255,255,255,0.2)]", styles.bg)}></div>}

                                            {/* Duration Badge */}
                                            {duration && (
                                                <div className="absolute top-0 right-2 -translate-y-1/2 bg-[#111] border border-white/10 px-1.5 py-0.5 rounded text-[10px] text-gray-500 font-mono shadow-sm z-10">
                                                    {duration}
                                                </div>
                                            )}

                                            <div className="flex gap-3">
                                                <div className={cn(
                                                    "mt-1 opacity-80 p-1.5 rounded-lg bg-white/5 self-start",
                                                    styles.iconColor, styles.border
                                                )}>{getEventIcon(event)}</div>

                                                <div className="flex-1 min-w-0">
                                                    <div className="flex justify-between items-baseline mb-1">
                                                        <span className={cn(
                                                            "text-[10px] font-bold uppercase tracking-wider",
                                                            styles.text
                                                        )}>{event.author}</span>
                                                        <span className="text-[10px] text-gray-600 font-mono">{formatTime(event.timestamp)}</span>
                                                    </div>

                                                    <div className="text-xs text-gray-300 font-medium truncate leading-relaxed opacity-90">
                                                        {event.metadata?.type === 'guardrail' ? 'Guardrail Intervention' :
                                                            event.metadata?.thinking ? 'Thinking Process' :
                                                                event.metadata?.tool_calls ? `Tool: ${event.metadata.tool_calls[0]?.name}` :
                                                                    event.metadata?.tool_results ? `Result: ${event.metadata.tool_results[0]?.tool_call_id.slice(0, 8)}...` :
                                                                        textContent || <span className="text-gray-600 italic">No Content</span>}
                                                    </div>

                                                    {event.metadata?.intervention && (
                                                        <div className="text-[10px] text-red-400 mt-1.5 flex items-center gap-1.5 bg-red-400/10 px-2 py-1 rounded border border-red-400/20">
                                                            <Shield size={10} /> Blocked: {(event.metadata.intervention as any).reason}
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>

                            {/* Detail View (Right) */}
                            <div className="flex-1 overflow-y-auto custom-scrollbar bg-[#111] relative">
                                {selectedEventId ? (() => {
                                    const event = events.find(e => e.id === selectedEventId);
                                    if (!event) return null;
                                    const styles = getEventStyle(event);
                                    return (
                                        <div className="p-8 max-w-4xl mx-auto">
                                            {/* Detail Header */}
                                            <div className="mb-8 flex items-start justify-between">
                                                <div className="flex items-center gap-4">
                                                    <div className={cn(
                                                        "p-4 rounded-xl shadow-lg bg-white/5",
                                                        styles.iconColor, styles.border
                                                    )}>
                                                        {getEventIcon(event)}
                                                    </div>
                                                    <div>
                                                        <h3 className={cn("text-2xl font-bold mb-2 tracking-tight", styles.text)}>
                                                            {(event.metadata?.type as string)?.toUpperCase() || event.author.toUpperCase()}
                                                        </h3>
                                                        <div className="flex items-center gap-4 text-xs text-gray-500 font-mono border-l-2 border-white/10 pl-3">
                                                            <span>ID: <span className="text-gray-400">{event.id}</span></span>
                                                            <span>TIME: <span className="text-gray-400">{event.timestamp}</span></span>
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>

                                            {/* Render Content based on Mode */}
                                            <div className="animate-in fade-in slide-in-from-bottom-2 duration-300">
                                                <EventContentErrorBoundary>
                                                    {renderEventContent(event)}
                                                </EventContentErrorBoundary>
                                            </div>
                                        </div>
                                    );
                                })() : (
                                    <div className="h-full flex flex-col items-center justify-center text-gray-600">
                                        <div className="w-24 h-24 bg-white/5 rounded-full flex items-center justify-center mb-6">
                                            <Activity size={48} className="opacity-20" />
                                        </div>
                                        <p className="text-lg font-medium text-gray-400">Select an event to inspect details</p>
                                        <p className="text-sm opacity-50 mt-2">View timelines, payloads, and execution traces</p>
                                    </div>
                                )}
                            </div>
                        </>
                    )}
                </div>

                {/* Timeline Visualizer Footer */}
                {
                    events.length > 0 && !loading && !error && (
                        <div className="h-20 bg-[#111] border-t border-white/10 flex items-start pt-4 px-4 gap-4 flex-shrink-0">
                            <div className="text-[10px] font-mono text-gray-500 w-16 text-right">
                                {formatTime(events[0].timestamp)}
                            </div>

                            {/* Bar Container - No overflow-hidden to allow labels below */}
                            <div className="flex-1 h-3 bg-white/5 rounded-full flex relative group/timeline">
                                {(() => {
                                    // Calculate Durations
                                    let totalMs = 0;
                                    const segments = events.map((e, i) => {
                                        const next = events[i + 1];
                                        const start = new Date(e.timestamp).getTime();
                                        const end = next ? new Date(next.timestamp).getTime() : start + 1000;
                                        const dur = Math.max(end - start, 100);
                                        totalMs += dur;

                                        // Use centralized styling
                                        const styles = getEventStyle(e);
                                        let bgClass = styles.bg;
                                        // Modifiers for thinking/tools
                                        if (e.metadata?.thinking) bgClass += " opacity-60";
                                        if (e.metadata?.tool_calls || e.metadata?.tool_results) bgClass += " brightness-75 saturate-150";

                                        return {
                                            event: e,
                                            dur,
                                            color: bgClass,
                                            styles
                                        };
                                    });

                                    return segments.map((seg, i) => {
                                        const widthPercent = (seg.dur / totalMs) * 100;
                                        const showLabel = widthPercent > 8;

                                        return (
                                            <div
                                                key={i}
                                                onClick={() => setSelectedEventId(seg.event.id)}
                                                className={cn(
                                                    "h-full transition-all cursor-pointer relative group/segment border-r border-black/10 last:border-0",
                                                    seg.color,
                                                    selectedEventId === seg.event.id ? "brightness-125 saturate-150 z-10 shadow-xl scale-y-110" : "opacity-90 hover:opacity-100 hover:brightness-110"
                                                )}
                                                style={{ width: `${widthPercent}%` }}
                                            >
                                                {/* Label Underneath */}
                                                {showLabel && (
                                                    <div className="absolute top-full mt-2 left-1/2 -translate-x-1/2 flex flex-col items-center pointer-events-none z-20">
                                                        <div className="w-px h-2 bg-white/20"></div>
                                                        <div className="px-1.5 py-0.5 rounded bg-[#1e1e1e] border border-white/10 text-[9px] font-bold leading-none whitespace-nowrap shadow-lg flex flex-col items-center">
                                                            <span className={cn("uppercase tracking-tight", seg.styles.text)}>{seg.event.author}</span>
                                                            <span className="font-mono text-[8px] text-gray-500 mt-0.5">
                                                                {(seg.dur < 1000) ? `${seg.dur}ms` : `${(seg.dur / 1000).toFixed(1)}s`}
                                                            </span>
                                                        </div>
                                                    </div>
                                                )}

                                                {/* Tooltip on Hover */}
                                                <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover/segment:block z-20 whitespace-nowrap">
                                                    <div className="bg-black/90 text-white text-[10px] px-2 py-1 rounded border border-white/10 shadow-xl transform translate-y-1">
                                                        <div className="font-bold flex items-center gap-1.5 uppercase">
                                                            <span className={cn("w-1.5 h-1.5 rounded-full", seg.color.split(' ')[0])}></span>
                                                            {seg.event.author}
                                                        </div>
                                                        <div className="text-gray-400 font-mono">
                                                            {(seg.dur < 1000) ? `${seg.dur}ms` : `${(seg.dur / 1000).toFixed(2)}s`}
                                                        </div>
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    });
                                })()}
                            </div>

                            <div className="text-[10px] font-mono text-gray-500 w-16">
                                {formatTime(events[events.length - 1].timestamp)}
                            </div>
                        </div>
                    )
                }
            </div>
        </div>
    );
};
