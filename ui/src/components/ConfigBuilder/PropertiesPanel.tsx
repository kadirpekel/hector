import React from "react";
import { X, Settings } from "lucide-react";
import type { Node } from "@xyflow/react";

interface PropertiesPanelProps {
  nodeId: string;
  node?: Node;
  onClose: () => void;
  onUpdate?: (updates: Record<string, any>) => void;
  readonly?: boolean;
}

export const PropertiesPanel: React.FC<PropertiesPanelProps> = ({
  nodeId,
  node,
  onClose,
  onUpdate,
  readonly = false,
}) => {
  if (!node) return null;

  const [localData, setLocalData] = React.useState(node.data);

  React.useEffect(() => {
    setLocalData(node.data);
  }, [node.data]);

  const handleInputChange = (field: string, value: any) => {
    if (!readonly && onUpdate) {
      onUpdate({ [field]: value });
    }
  };

  return (
    <div className="w-80 bg-black/40 border-l border-white/10 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-white/10">
        <div className="flex items-center gap-2">
          <Settings size={18} className="text-gray-400" />
          <h3 className="font-semibold text-sm">Properties</h3>
          {readonly && (
            <span className="text-xs px-2 py-0.5 bg-blue-500/20 text-blue-300 rounded">
              Read-only
            </span>
          )}
        </div>
        <button
          onClick={onClose}
          className="p-1 hover:bg-white/10 rounded transition-colors"
          title="Close"
        >
          <X size={18} />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Node ID */}
        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">
            Node ID
          </label>
          <div className="px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm font-mono text-gray-300">
            {nodeId}
          </div>
        </div>

        {/* Node Type */}
        <div>
          <label className="block text-sm font-medium text-gray-400 mb-1">
            Type
          </label>
          <div className="px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-sm capitalize">
            {node.type || "Unknown"}
          </div>
        </div>

        {/* Agent-specific fields */}
        {node.type === "agent" && (
          <>
            <div>
              <label className="block text-sm font-medium mb-2">
                Agent Name
              </label>
              <input
                type="text"
                value={String(localData?.name || "")}
                onChange={(e) =>
                  setLocalData({ ...localData, name: e.target.value })
                }
                onBlur={() => handleInputChange("name", localData?.name)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50 disabled:cursor-not-allowed"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">LLM</label>
              <select
                value={String((node.data as any)?.llm || "")}
                onChange={(e) => handleInputChange("llm", e.target.value)}
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded text-sm focus:outline-none focus:border-hector-green disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <option value="">Select LLM</option>
                <option value="default">default</option>
                <option value="openai">openai</option>
                <option value="anthropic">anthropic</option>
                <option value="gemini">gemini</option>
                <option value="ollama">ollama</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">
                Description
              </label>
              <textarea
                value={String((node.data as any)?.description || "")}
                onChange={(e) =>
                  handleInputChange("description", e.target.value)
                }
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg focus:outline-none focus:border-hector-green transition-colors resize-none disabled:opacity-50 disabled:cursor-not-allowed"
                rows={3}
                placeholder="Describe what this agent does..."
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">
                Instruction
              </label>
              <textarea
                value={String((node.data as any)?.instruction || "")}
                onChange={(e) =>
                  handleInputChange("instruction", e.target.value)
                }
                disabled={readonly}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg focus:outline-none focus:border-hector-green transition-colors resize-none font-mono text-xs disabled:opacity-50 disabled:cursor-not-allowed"
                rows={6}
                placeholder="System prompt for this agent..."
              />
            </div>
          </>
        )}

        {/* Loop-specific fields */}
        {node.type === "loop" && (
          <div>
            <label className="block text-sm font-medium mb-2">
              Max Iterations
            </label>
            <input
              type="number"
              value={(node.data as any)?.maxIterations || 3}
              onChange={(e) =>
                handleInputChange("maxIterations", parseInt(e.target.value))
              }
              disabled={readonly}
              className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg focus:outline-none focus:border-hector-green transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              min={0}
              placeholder="Number of iterations"
            />
          </div>
        )}
      </div>
    </div>
  );
};
