import React from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { cn } from "../../../lib/utils";
import { useStore } from "../../../store/useStore";
import { getAgentBranding } from "../../../lib/colors";

interface ConditionalNodeData extends Record<string, unknown> {
    label: string;
    agentId?: string;
    condition_agent?: string;
    condition_field?: string;
    on_true_agent?: string;
    on_false_response?: string;
}

/**
 * Conditional Agent Node - Visualizes conditional routing
 */
export const ConditionalNode: React.FC<NodeProps> = ({ data, selected }) => {
    const nodeData = data as ConditionalNodeData;
    const activeAgentId = useStore((state) => state.activeAgentId);

    const compareId = nodeData.agentId || nodeData.label;
    const isActive = activeAgentId?.toLowerCase() === compareId?.toLowerCase();

    // Centralized branding
    const branding = getAgentBranding({ name: nodeData.label, type: 'conditional' });
    const Icon = branding.Icon;

    return (
        <div className="relative group/node">
            {/* Active Pulse Effect */}
            {isActive && (
                <div className="absolute -inset-4 rounded-xl bg-gradient-to-r from-purple-500 via-pink-500 to-purple-500 blur-xl opacity-60 animate-pulse transition duration-1000 group-hover/node:opacity-80 pointer-events-none"></div>
            )}

            <div
                className={cn(
                    "px-4 py-3 shadow-2xl rounded-xl border transition-all min-w-[220px] relative overflow-visible backdrop-blur-xl",
                    // Use centralized styling
                    branding.classes.bg,
                    branding.classes.border,
                    isActive
                        ? "scale-110 z-50 ring-4 ring-white/60 brightness-110 translate-y-[-2px]"
                        : selected
                            ? "ring-2 ring-purple-400/50"
                            : "hover:border-purple-400/70 hover:scale-105 hover:z-40",
                )}
            >
                {/* Handles */}
                <Handle
                    type="target"
                    position={Position.Top}
                    id="top"
                    className="w-2 h-2 !opacity-0 !pointer-events-none !-top-2"
                />
                <Handle
                    type="target"
                    position={Position.Left}
                    id="left"
                    className="w-2 h-2 !opacity-0 !pointer-events-none !-left-2"
                />

                {/* Header */}
                <div className="flex items-center gap-3 mb-3 relative z-10">
                    <div
                        className={cn(
                            "w-9 h-9 rounded-lg flex items-center justify-center shadow-inner border border-white/20 transition-transform duration-500",
                            branding.classes.bg,
                            isActive && "rotate-6 scale-110",
                        )}
                    >
                        <Icon size={20} className="text-white/90" />
                    </div>
                    <div className="flex-1 min-w-0">
                        <div className="font-bold truncate tracking-tight text-white text-base shadow-black/50 drop-shadow-sm">
                            {nodeData.label}
                        </div>
                        <div className="text-[10px] uppercase tracking-wider font-bold opacity-70 text-purple-300 flex items-center gap-1">
                            {branding.label || "Conditional"}
                            {isActive && (
                                <span className="w-1.5 h-1.5 rounded-full bg-purple-400 animate-ping inline-block ml-1" />
                            )}
                        </div>
                    </div>
                </div>

                {/* Condition Info */}
                <div className="space-y-1.5 relative z-10">
                    <div className="flex items-center justify-between text-xs text-white/50 font-mono bg-black/30 rounded px-2.5 py-1.5 border border-white/5 shadow-inner">
                        <span className="text-purple-300/70 text-[10px]">field:</span>
                        <span className="truncate max-w-[120px] opacity-80 text-purple-200">
                            {nodeData.condition_field || "safe"}
                        </span>
                    </div>
                    {nodeData.on_false_response && (
                        <div className="text-[10px] text-red-400/70 truncate bg-red-500/10 rounded px-2 py-1 border border-red-500/20">
                            ✕ {nodeData.on_false_response.slice(0, 30)}...
                        </div>
                    )}
                </div>

                <Handle
                    type="source"
                    position={Position.Right}
                    id="right"
                    className="w-2 h-2 !opacity-0 !pointer-events-none !-right-2"
                />
                <Handle
                    type="source"
                    position={Position.Bottom}
                    id="bottom"
                    className="w-2 h-2 !opacity-0 !pointer-events-none !-bottom-2"
                />
            </div>
        </div>
    );
};
