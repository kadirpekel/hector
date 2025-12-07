import React from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Workflow } from "lucide-react";
import { cn } from "../../../lib/utils";

interface AgentNodeData extends Record<string, unknown> {
  name?: string;
  label?: string;
  llm?: string;
  description?: string;
  instruction?: string;
  subAgents?: string[];
}

const getLLMBadgeColor = (llm?: string) => {
  switch (llm) {
    case "openai":
      return "bg-green-500/20 text-green-300 border-green-500/30";
    case "gemini":
      return "bg-purple-500/20 text-purple-300 border-purple-500/30";
    case "anthropic":
      return "bg-orange-500/20 text-orange-300 border-orange-500/30";
    case "ollama":
      return "bg-blue-500/20 text-blue-300 border-blue-500/30";
    default:
      return "bg-gray-500/20 text-gray-300 border-gray-500/30";
  }
};

export const AgentNode: React.FC<NodeProps> = ({ data, selected }) => {
  const nodeData = data as AgentNodeData;
  const displayName = nodeData.name || nodeData.label || "Unnamed Agent";

  return (
    <div
      className={cn(
        "px-4 py-3 shadow-lg rounded-lg border-2 transition-all min-w-[200px]",
        "bg-gradient-to-br from-green-500 to-green-600",
        selected ? "border-white ring-2 ring-white/50" : "border-green-700",
      )}
    >
      {/* Connection Handles - Simpler, consistent positioning */}
      <Handle
        type="target"
        position={Position.Left}
        id="left"
        className="!w-3 !h-3 !bg-green-400 hover:!bg-green-300 hover:!scale-150 !transition-all"
      />
      <Handle
        type="target"
        position={Position.Top}
        id="top"
        className="!w-3 !h-3 !bg-green-400 hover:!bg-green-300 hover:!scale-150 !transition-all"
      />

      {/* Header */}
      <div className="flex items-center gap-2 mb-2">
        <Workflow size={18} className="text-white flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="font-bold text-white truncate">{displayName}</div>
          <div className="text-xs text-green-100">LLM Agent</div>
        </div>
      </div>

      {/* LLM Badge */}
      {nodeData.llm && (
        <div className="mt-2">
          <div
            className={cn(
              "inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium border",
              getLLMBadgeColor(nodeData.llm),
            )}
          >
            <span className="w-1.5 h-1.5 rounded-full bg-current"></span>
            {nodeData.llm}
          </div>
        </div>
      )}

      {/* Stats */}
      {nodeData.subAgents && nodeData.subAgents.length > 0 && (
        <div className="mt-2 text-xs text-green-100">
          {nodeData.subAgents.length} connection
          {nodeData.subAgents.length > 1 ? "s" : ""}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
        id="right"
        className="!w-3 !h-3 !bg-green-400 hover:!bg-green-300 hover:!scale-150 !transition-all"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="bottom"
        className="!w-3 !h-3 !bg-green-400 hover:!bg-green-300 hover:!scale-150 !transition-all"
      />
    </div>
  );
};
