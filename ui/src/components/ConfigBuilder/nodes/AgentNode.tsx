import React from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { Workflow } from 'lucide-react';
import { cn } from '../../../lib/utils';

interface AgentNodeData extends Record<string, unknown> {
  label: string;
  llm?: string;
  description?: string;
  instruction?: string;
}

const getLLMBadgeColor = (llm: string) => {
  const lowerLLM = llm.toLowerCase();
  if (lowerLLM.includes('openai') || lowerLLM.includes('gpt')) {
    return 'bg-blue-500/20 border-blue-500 text-blue-300';
  }
  if (lowerLLM.includes('anthropic') || lowerLLM.includes('claude')) {
    return 'bg-purple-500/20 border-purple-500 text-purple-300';
  }
  if (lowerLLM.includes('gemini') || lowerLLM.includes('google')) {
    return 'bg-orange-500/20 border-orange-500 text-orange-300';
  }
  if (lowerLLM.includes('ollama')) {
    return 'bg-teal-500/20 border-teal-500 text-teal-300';
  }
  return 'bg-gray-500/20 border-gray-500 text-gray-300';
};

/**
 * Agent node - displays agent information in read-only mode
 * Handles are visible but non-interactive
 */
export const AgentNode: React.FC<NodeProps> = ({ data, selected }) => {
  const nodeData = data as AgentNodeData;

  return (
    <div
      className={cn(
        'px-4 py-3 shadow-lg rounded-lg border-2 transition-all min-w-[200px]',
        'bg-gradient-to-br from-green-500 to-green-600',
        selected ? 'border-white ring-2 ring-white/50' : 'border-green-700'
      )}
    >
      {/* Handles - read-only, just for visualization */}
      <Handle 
        type="target" 
        position={Position.Top} 
        className="w-3 h-3 !bg-gray-400 !border-2 !border-gray-600"
        isConnectable={false}
      />
      <Handle 
        type="target" 
        position={Position.Left} 
        className="w-3 h-3 !bg-gray-400 !border-2 !border-gray-600"
        isConnectable={false}
      />

      {/* Header */}
      <div className="flex items-center gap-2 mb-2">
        <Workflow size={18} className="text-white flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="font-bold text-white truncate">{nodeData.label || 'Unnamed Agent'}</div>
          <div className="text-xs text-green-100">Agent</div>
        </div>
      </div>

      {/* LLM Badge */}
      {nodeData.llm && (
        <div className="mt-2">
          <div
            className={cn(
              'inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium border',
              getLLMBadgeColor(nodeData.llm)
            )}
          >
            <span className="w-1.5 h-1.5 rounded-full bg-current"></span>
            {nodeData.llm}
          </div>
        </div>
      )}

      <Handle 
        type="source" 
        position={Position.Right} 
        className="w-3 h-3 !bg-gray-400 !border-2 !border-gray-600"
        isConnectable={false}
      />
      <Handle 
        type="source" 
        position={Position.Bottom} 
        className="w-3 h-3 !bg-gray-400 !border-2 !border-gray-600"
        isConnectable={false}
      />
    </div>
  );
};
