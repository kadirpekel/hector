import React from "react";
import { type NodeProps, Handle, Position } from "@xyflow/react";
import { GitBranch, Zap, Repeat } from "lucide-react";

interface WorkflowGroupNodeData extends Record<string, unknown> {
  label: string;
  workflowType: "sequential" | "parallel" | "loop";
  subAgents: string[];
  maxIterations?: number;
}

const getWorkflowIcon = (type: string) => {
  switch (type) {
    case "sequential":
      return GitBranch;
    case "parallel":
      return Zap;
    case "loop":
      return Repeat;
    default:
      return GitBranch;
  }
};

const getWorkflowStyles = (type: string) => {
  switch (type) {
    case "sequential":
      return {
        border: "2px solid #3b82f6",
        backgroundColor: "rgba(59, 130, 246, 0.05)",
        color: "#60a5fa",
      };
    case "parallel":
      return {
        border: "2px solid #f97316",
        backgroundColor: "rgba(249, 115, 22, 0.05)",
        color: "#fb923c",
      };
    case "loop":
      return {
        border: "2px solid #14b8a6",
        backgroundColor: "rgba(20, 184, 166, 0.05)",
        color: "#2dd4bf",
      };
    default:
      return {
        border: "2px solid #6b7280",
        backgroundColor: "rgba(107, 114, 128, 0.05)",
        color: "#9ca3af",
      };
  }
};

/**
 * Workflow group node - clean container for workflow patterns
 * Sized dynamically to fit all child nodes
 */
export const WorkflowGroupNode: React.FC<NodeProps> = ({ data }) => {
  const nodeData = data as WorkflowGroupNodeData;
  const Icon = getWorkflowIcon(nodeData.workflowType);
  const styles = getWorkflowStyles(nodeData.workflowType);

  return (
    <div
      className="rounded-xl overflow-visible"
      style={{
        width: "100%",
        height: "100%",
        ...styles,
        position: "relative",
        pointerEvents: "none", // Let clicks pass through to children
      }}
    >
      {/* Header */}
      <div
        className="absolute top-3 left-3 flex items-center gap-2 px-3 py-1.5 rounded-lg"
        style={{
          backgroundColor: "rgba(0, 0, 0, 0.3)",
          pointerEvents: "auto",
        }}
      >
        <Icon size={16} />
        <span className="text-xs font-bold uppercase tracking-wider">
          {nodeData.workflowType}
        </span>
        {nodeData.maxIterations !== undefined && (
          <span className="text-xs opacity-70">
            (max: {nodeData.maxIterations || "∞"})
          </span>
        )}
      </div>

      {/* Child nodes render here automatically by React Flow */}

      {/* Handles - Hidden but functional */}
      <Handle
        type="target"
        position={Position.Top}
        className="w-2 h-2 !opacity-0 !pointer-events-none !-top-2"
        id="top"
      />
      <Handle
        type="target"
        position={Position.Left}
        className="w-2 h-2 !opacity-0 !pointer-events-none !-left-2"
        id="left"
      />
      <Handle
        type="source"
        position={Position.Right}
        className="w-2 h-2 !opacity-0 !pointer-events-none !-right-2"
        id="right"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        className="w-2 h-2 !opacity-0 !pointer-events-none !-bottom-2"
        id="bottom"
      />
    </div>
  );
};
