import React, { useState, useMemo } from "react";
import { Database, Cpu, Box, Layers } from "lucide-react";
import * as yaml from "js-yaml";
import type { Node } from "@xyflow/react";

interface InfrastructureSidebarProps {
  yamlContent: string;
  nodes: Node[];
  collapsed?: boolean;
}

export const InfrastructureSidebar: React.FC<InfrastructureSidebarProps> = ({
  yamlContent,
  nodes,
  collapsed = false,
}) => {
  const [expandedSection, setExpandedSection] = useState<string | null>("llms");

  const infrastructure = useMemo(() => {
    try {
      const config = (yaml.load(yamlContent) || {}) as any;
      const llms = config.llms || {};
      const databases = config.databases || {};
      const embedders = config.embedders || {};
      const vectorStores = config.vector_stores || {};

      // Calculate LLM usage from nodes
      const llmUsage: Record<string, number> = {};
      nodes.forEach((node) => {
        if (node.data && typeof node.data === "object" && "llm" in node.data) {
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
        llmDetails: llms,
        databaseDetails: databases,
        llmUsage,
      };
    } catch (e) {
      return {
        llms: 0,
        databases: 0,
        embedders: 0,
        vectorStores: 0,
        llmDetails: {},
        databaseDetails: {},
        llmUsage: {},
      };
    }
  }, [yamlContent, nodes]);

  if (collapsed) {
    // Collapsed view: Just icons and counts
    return (
      <div className="w-16 bg-black/40 border-r border-white/10 flex flex-col items-center py-4 gap-6">
        <div className="flex flex-col items-center gap-1" title="LLMs">
          <Cpu size={20} className="text-blue-400" />
          <span className="text-xs text-gray-400">{infrastructure.llms}</span>
        </div>
        <div className="flex flex-col items-center gap-1" title="Databases">
          <Database size={20} className="text-green-400" />
          <span className="text-xs text-gray-400">
            {infrastructure.databases}
          </span>
        </div>
        <div className="flex flex-col items-center gap-1" title="Embedders">
          <Layers size={20} className="text-purple-400" />
          <span className="text-xs text-gray-400">
            {infrastructure.embedders}
          </span>
        </div>
        <div className="flex flex-col items-center gap-1" title="Vector Stores">
          <Box size={20} className="text-orange-400" />
          <span className="text-xs text-gray-400">
            {infrastructure.vectorStores}
          </span>
        </div>
      </div>
    );
  }

  // Expanded view: Full details
  return (
    <div className="w-64 bg-black/40 border-r border-white/10 flex flex-col overflow-hidden">
      <div className="flex-none px-4 py-3 border-b border-white/10">
        <h2 className="text-sm font-semibold">Infrastructure</h2>
      </div>

      <div className="flex-1 overflow-y-auto">
        {/* LLMs */}
        <div className="border-b border-white/10">
          <button
            onClick={() =>
              setExpandedSection(expandedSection === "llms" ? null : "llms")
            }
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Cpu size={16} className="text-blue-400" />
              <span className="text-sm font-medium">LLMs</span>
            </div>
            <span className="text-xs px-2 py-0.5 bg-blue-500/20 text-blue-300 rounded">
              {infrastructure.llms}
            </span>
          </button>
          {expandedSection === "llms" && (
            <div className="px-4 py-2 space-y-2 bg-black/20">
              {Object.entries(infrastructure.llmDetails).map(
                ([name, config]: [string, any]) => (
                  <div key={name} className="text-xs">
                    <div className="font-medium text-gray-300">{name}</div>
                    <div className="text-gray-500">
                      {config.provider} / {config.model}
                    </div>
                  </div>
                ),
              )}
              {infrastructure.llms === 0 && (
                <div className="text-xs text-gray-500">No LLMs configured</div>
              )}
            </div>
          )}
        </div>

        {/* Databases */}
        <div className="border-b border-white/10">
          <button
            onClick={() =>
              setExpandedSection(
                expandedSection === "databases" ? null : "databases",
              )
            }
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Database size={16} className="text-green-400" />
              <span className="text-sm font-medium">Databases</span>
            </div>
            <span className="text-xs px-2 py-0.5 bg-green-500/20 text-green-300 rounded">
              {infrastructure.databases}
            </span>
          </button>
          {expandedSection === "databases" &&
            infrastructure.databases === 0 && (
              <div className="px-4 py-2 text-xs text-gray-500 bg-black/20">
                No databases configured
              </div>
            )}
        </div>

        {/* Embedders */}
        <div className="border-b border-white/10">
          <button
            onClick={() =>
              setExpandedSection(
                expandedSection === "embedders" ? null : "embedders",
              )
            }
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Layers size={16} className="text-purple-400" />
              <span className="text-sm font-medium">Embedders</span>
            </div>
            <span className="text-xs px-2 py-0.5 bg-purple-500/20 text-purple-300 rounded">
              {infrastructure.embedders}
            </span>
          </button>
          {expandedSection === "embedders" &&
            infrastructure.embedders === 0 && (
              <div className="px-4 py-2 text-xs text-gray-500 bg-black/20">
                No embedders configured
              </div>
            )}
        </div>

        {/* Vector Stores */}
        <div className="border-b border-white/10">
          <button
            onClick={() =>
              setExpandedSection(
                expandedSection === "vectorStores" ? null : "vectorStores",
              )
            }
            className="w-full px-4 py-3 flex items-center justify-between hover:bg-white/5 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Box size={16} className="text-orange-400" />
              <span className="text-sm font-medium">Vector Stores</span>
            </div>
            <span className="text-xs px-2 py-0.5 bg-orange-500/20 text-orange-300 rounded">
              {infrastructure.vectorStores}
            </span>
          </button>
          {expandedSection === "vectorStores" &&
            infrastructure.vectorStores === 0 && (
              <div className="px-4 py-2 text-xs text-gray-500 bg-black/20">
                No vector stores configured
              </div>
            )}
        </div>
      </div>
    </div>
  );
};
