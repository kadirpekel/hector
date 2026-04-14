import React from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { ArrowRight } from "lucide-react";
import { cn } from "../../../lib/utils";
import { useStore } from "../../../store/useStore";
import { getAgentBranding } from "../../../lib/colors";

interface RunnerNodeData extends Record<string, unknown> {
    label: string;
    agentId?: string;
    description?: string;
    tools?: string[];
}

/**
 * RunnerNode - Visual representation for runner agents (LLM-less tool pipelines)
 */
export const RunnerNode: React.FC<NodeProps> = ({ data, selected }) => {
    const nodeData = data as RunnerNodeData;
    const activeAgentId = useStore((state) => state.activeAgentId);

    const compareId = nodeData.agentId || nodeData.label;
    const isActive = activeAgentId?.toLowerCase() === compareId?.toLowerCase();

    // Centralized branding (will return Orange/Play)
    const branding = getAgentBranding({ name: nodeData.label, type: 'runner' });
    const Icon = branding.Icon;

    // Get tool pipeline preview
    const tools = nodeData.tools || [];
    const toolPreview = tools.length > 0
        ? tools.length <= 3
            ? tools.join(" → ")
            : `${tools[0]} → ... → ${tools[tools.length - 1]}`
        : "No tools configured";

    return (
        <div className="relative group/node">
            {/* Active Pulse Effect */}
            {isActive && (
                <div className="absolute -inset-4 rounded-xl bg-gradient-to-r from-orange-500 via-amber-500 to-yellow-500 blur-xl opacity-60 animate-pulse transition duration-1000 group-hover/node:opacity-80 group-hover/node:duration-200 pointer-events-none"></div>
            )}

            <div
                className={cn(
                    "px-4 py-3 shadow-2xl rounded-xl border transition-all min-w-[220px] relative overflow-visible backdrop-blur-xl",
                    // Centralized styling
                    branding.classes.bg,
                    branding.classes.border,
                    isActive
                        ? "scale-110 z-50 ring-4 ring-white/60 brightness-110 translate-y-[-2px]"
                        : selected
                            ? "ring-2 ring-orange-400/30"
                            : "hover:border-white/50 hover:scale-105 hover:z-40",
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
                        <Icon size={18} className="text-white/90" />
                    </div>
                    <div className="flex-1 min-w-0">
                        <div className="font-bold truncate tracking-tight text-white text-base shadow-black/50 drop-shadow-sm">
                            {nodeData.label}
                        </div>
                        <div className="text-[10px] uppercase tracking-wider font-bold opacity-80 text-white flex items-center gap-1">
                            {branding.label || "Runner"}
                            {tools.length > 0 && (
                                <span className="opacity-70 font-medium ml-1">• {tools.length} apps</span>
                            )}
                            {isActive && (
                                <span className="w-1.5 h-1.5 rounded-full bg-orange-400 animate-ping inline-block ml-1" />
                            )}
                        </div>
                    </div>
                </div>

                {/* Tool Pipeline Preview */}
                <div className="flex items-center gap-1.5 text-xs text-white/70 font-mono bg-black/30 rounded px-2.5 py-1.5 border border-white/10 relative z-10 shadow-inner">
                    <ArrowRight size={12} className="text-white/50 flex-shrink-0" />
                    <span className="truncate opacity-80 group-hover/node:opacity-100 transition-opacity">
                        {toolPreview}
                    </span>
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
