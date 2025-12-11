import React, { useState, useMemo } from 'react';
import { Database, Cpu, Box, Layers, Wrench } from 'lucide-react';
import * as yaml from 'js-yaml';
import type { Node } from '@xyflow/react';

interface InfrastructureSidebarProps {
  yamlContent: string;
  nodes: Node[];
  collapsed?: boolean;
  onToggle?: () => void;
}

export const InfrastructureSidebar: React.FC<InfrastructureSidebarProps> = React.memo(({
  yamlContent,
  nodes,
  collapsed = false,
  onToggle
}) => {
  console.log("RENDER: InfrastructureSidebar (Canvas)");
  const [expandedSection, setExpandedSection] = useState<string | null>('llms');

  const infrastructure = useMemo(() => {
    try {
      const config = (yaml.load(yamlContent) || {}) as any;
      const llms = config.llms || {};
      const databases = config.databases || {};
      const embedders = config.embedders || {};
      const vectorStores = config.vector_stores || {};
      const tools = config.tools || {};

      // Calculate LLM usage from nodes
      const llmUsage: Record<string, number> = {};
      nodes.forEach((node) => {
        if (node.data && typeof node.data === 'object' && 'llm' in node.data) {
          const llmName = (node.data as any).llm;
          if (llmName) {
            llmUsage[llmName] = (llmUsage[llmName] || 0) + 1;
          }
        }
      });

      return {
        llms: Object.keys(llms).length,
        databases: Object.keys(databases).length,
        embedders: Object.keys(embedders).length,
        vectorStores: Object.keys(vectorStores).length,
        tools: Object.keys(tools).length,
        llmDetails: llms,
        databaseDetails: databases,
        toolDetails: tools,
        llmUsage,
      };
    } catch (e) {
      return {
        llms: 0,
        databases: 0,
        embedders: 0,
        vectorStores: 0,
        tools: 0,
        llmDetails: {},
        databaseDetails: {},
        toolDetails: {},
        llmUsage: {},
      };
    }
  }, [yamlContent, nodes]);

  if (collapsed) {
    // Collapsed view: Just icons and counts
    return (
      <div className="w-16 bg-black/40 border-r border-white/10 flex flex-col items-center py-4 gap-6">
        <button
          onClick={onToggle}
          className="flex flex-col items-center gap-1 hover:bg-white/5 p-2 rounded transition-colors w-full"
          title="LLMs - Click to expand"
        >
          <Cpu size={20} className="text-blue-400" />
          <span className="text-xs text-gray-400">{infrastructure.llms}</span>
        </button>
        <button
          onClick={onToggle}
          className="flex flex-col items-center gap-1 hover:bg-white/5 p-2 rounded transition-colors w-full"
          title="Tools - Click to expand"
        >
          <Wrench size={20} className="text-yellow-400" />
          <span className="text-xs text-gray-400">{infrastructure.tools}</span>
        </button>
        <button
          onClick={onToggle}
          className="flex flex-col items-center gap-1 hover:bg-white/5 p-2 rounded transition-colors w-full"
          title="Databases - Click to expand"
        >
          <Database size={20} className="text-green-400" />
          <span className="text-xs text-gray-400">{infrastructure.databases}</span>
        </button>
        <button
          onClick={onToggle}
          className="flex flex-col items-center gap-1 hover:bg-white/5 p-2 rounded transition-colors w-full"
          title="Embedders - Click to expand"
        >
          <Layers size={20} className="text-purple-400" />
          <span className="text-xs text-gray-400">{infrastructure.embedders}</span>
        </button>
        <button
          onClick={onToggle}
          className="flex flex-col items-center gap-1 hover:bg-white/5 p-2 rounded transition-colors w-full"
          title="Vector Stores - Click to expand"
        >
          <Box size={20} className="text-orange-400" />
          <span className="text-xs text-gray-400">{infrastructure.vectorStores}</span>
        </button>
      </div>
    );
  }

  // Expanded view: Full details
  return (
    <div className="w-64 bg-black/40 border-r border-white/10 flex flex-col overflow-hidden">
      <div className="flex-none px-4 py-3 border-b border-white/10 flex items-center justify-between">
        <h2 className="text-sm font-semibold">Infrastructure</h2>
        <button
          onClick={onToggle}
          className="text-gray-400 hover:text-white transition-colors"
          title="Collapse sidebar"
        >
          <Layers size={16} className="rotate-90" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto custom-scrollbar">
        {/* LLMs */}
        <div className="border-b border-white/10">
          <button
            onClick={() => setExpandedSection(expandedSection === 'llms' ? null : 'llms')}
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded bg-blue-500/10 flex items-center justify-center group-hover:bg-blue-500/20 transition-colors">
                <Cpu size={16} className="text-blue-400" />
              </div>
              <div className="text-left">
                <div className="text-sm font-medium text-gray-200">LLMs</div>
                <div className="text-[10px] text-gray-500">{infrastructure.llms} configured</div>
              </div>
            </div>
            {expandedSection === 'llms' ? (
              <div className="w-1.5 h-1.5 rounded-full bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.8)]" />
            ) : null}
          </button>

          {expandedSection === 'llms' && (
            <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
              {Object.entries(infrastructure.llmDetails).map(([name, config]: [string, any]) => (
                <div key={name} className="p-2 rounded border border-white/5 bg-white/5 hover:bg-white/10 transition-colors">
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-heading font-medium text-xs text-blue-300">{name}</span>
                    {infrastructure.llmUsage[name] > 0 && (
                      <span className="text-[10px] bg-white/10 px-1.5 py-0.5 rounded text-gray-400">
                        {infrastructure.llmUsage[name]} refs
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-1.5 text-[10px] text-gray-400">
                    <span className="px-1 py-px rounded bg-white/5 border border-white/5">{config.provider}</span>
                    <span className="truncate max-w-[120px]" title={config.model}>{config.model}</span>
                  </div>
                </div>
              ))}
              {infrastructure.llms === 0 && (
                <div className="text-xs text-center py-2 text-gray-600 italic">No LLMs defined</div>
              )}
            </div>
          )}
        </div>

        {/* Tools */}
        <div className="border-b border-white/10">
          <button
            onClick={() => setExpandedSection(expandedSection === 'tools' ? null : 'tools')}
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded bg-yellow-500/10 flex items-center justify-center group-hover:bg-yellow-500/20 transition-colors">
                <Wrench size={16} className="text-yellow-400" />
              </div>
              <div className="text-left">
                <div className="text-sm font-medium text-gray-200">Tools</div>
                <div className="text-[10px] text-gray-500">{infrastructure.tools} configured</div>
              </div>
            </div>
            {expandedSection === 'tools' ? (
              <div className="w-1.5 h-1.5 rounded-full bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.8)]" />
            ) : null}
          </button>
          {expandedSection === 'tools' && (
            <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
              {Object.entries(infrastructure.toolDetails).map(([name, config]: [string, any]) => (
                <div key={name} className="p-2 rounded border border-white/5 bg-white/5 hover:bg-white/10 transition-colors flex items-center justify-between">
                  <div>
                    <div className="font-heading font-medium text-xs text-yellow-300">{name}</div>
                    <div className="text-[10px] text-yellow-500/70 capitalize">{config.type || 'mcp'}</div>
                  </div>
                  {config.url && <div className="text-[9px] text-gray-600 truncate max-w-[80px]">{config.url}</div>}
                </div>
              ))}
              {infrastructure.tools === 0 && (
                <div className="text-xs text-center py-2 text-gray-600 italic">No tools defined</div>
              )}
            </div>
          )}
        </div>

        {/* Databases */}
        <div className="border-b border-white/10">
          <button
            onClick={() => setExpandedSection(expandedSection === 'databases' ? null : 'databases')}
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded bg-green-500/10 flex items-center justify-center group-hover:bg-green-500/20 transition-colors">
                <Database size={16} className="text-green-400" />
              </div>
              <div className="text-left">
                <div className="text-sm font-medium text-gray-200">Databases</div>
                <div className="text-[10px] text-gray-500">{infrastructure.databases} configured</div>
              </div>
            </div>
            {expandedSection === 'databases' ? (
              <div className="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.8)]" />
            ) : null}
          </button>
          {expandedSection === 'databases' && (
            <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
              {Object.entries(infrastructure.databaseDetails).map(([name, config]: [string, any]) => (
                <div key={name} className="p-2 rounded border border-white/5 bg-white/5 hover:bg-white/10 transition-colors">
                  <div className="font-heading font-medium text-xs text-green-300">{name}</div>
                  <div className="text-[10px] text-gray-500 uppercase tracking-wider mt-1">{config.driver || config.type || 'postgres'}</div>
                </div>
              ))}
              {infrastructure.databases === 0 && (
                <div className="text-xs text-center py-2 text-gray-600 italic">No databases defined</div>
              )}
            </div>
          )}
        </div>

        {/* Embedders */}
        <div className="border-b border-white/10">
          <button
            onClick={() => setExpandedSection(expandedSection === 'embedders' ? null : 'embedders')}
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded bg-purple-500/10 flex items-center justify-center group-hover:bg-purple-500/20 transition-colors">
                <Layers size={16} className="text-purple-400" />
              </div>
              <div className="text-left">
                <div className="text-sm font-medium text-gray-200">Embedders</div>
                <div className="text-[10px] text-gray-500">{infrastructure.embedders} configured</div>
              </div>
            </div>
            {expandedSection === 'embedders' ? (
              <div className="w-1.5 h-1.5 rounded-full bg-purple-500 shadow-[0_0_8px_rgba(168,85,247,0.8)]" />
            ) : null}
          </button>
          {expandedSection === 'embedders' && (
            <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
              {infrastructure.embedders === 0 && (
                <div className="text-xs text-center py-2 text-gray-600 italic">No embedders defined</div>
              )}
              {/* Embedder specific rendering can be added here similar to LLMs if needed */}
            </div>
          )}
        </div>

        {/* Vector Stores */}
        <div className="border-b border-white/10">
          <button
            onClick={() => setExpandedSection(expandedSection === 'vectorStores' ? null : 'vectorStores')}
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors group"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded bg-orange-500/10 flex items-center justify-center group-hover:bg-orange-500/20 transition-colors">
                <Box size={16} className="text-orange-400" />
              </div>
              <div className="text-left">
                <div className="text-sm font-medium text-gray-200">Vector Stores</div>
                <div className="text-[10px] text-gray-500">{infrastructure.vectorStores} configured</div>
              </div>
            </div>
            {expandedSection === 'vectorStores' ? (
              <div className="w-1.5 h-1.5 rounded-full bg-orange-500 shadow-[0_0_8px_rgba(249,115,22,0.8)]" />
            ) : null}
          </button>
          {expandedSection === 'vectorStores' && (
            <div className="px-4 pb-3 space-y-2 bg-black/20 border-t border-white/5 pt-2">
              {infrastructure.vectorStores === 0 && (
                <div className="text-xs text-center py-2 text-gray-600 italic">No vector stores defined</div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
