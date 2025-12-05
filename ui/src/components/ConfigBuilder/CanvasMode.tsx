import React from "react";
import {
  ReactFlow,
  Background,
  type Node,
  type Edge,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { ChevronRight, ChevronLeft } from "lucide-react";

import { InfrastructureSidebar } from "./InfrastructureSidebar";
import { PropertiesPanel } from "./PropertiesPanel";
import { AgentNode } from "./nodes/AgentNode";
import { WorkflowGroupNode } from "./nodes/WorkflowGroupNode";
import { yamlToGraph } from "../../lib/canvas-converter";

// Custom node types for read-only visualization
const nodeTypes = {
  agent: AgentNode,
  workflowGroup: WorkflowGroupNode,
};

interface CanvasModeProps {
  yamlContent: string;
}

/**
 * Read-only canvas visualization of YAML configuration
 * Shows workflow patterns (sequential, parallel, loop) as visual groups
 */
export const CanvasMode: React.FC<CanvasModeProps> = ({ yamlContent }) => {
  const [nodes, setNodes] = React.useState<Node[]>([]);
  const [edges, setEdges] = React.useState<Edge[]>([]);
  const [selectedNodeId, setSelectedNodeId] = React.useState<string | null>(
    null,
  );
  const [sidebarCollapsed, setSidebarCollapsed] = React.useState(true); // Start collapsed

  // Update visualization when YAML changes
  React.useEffect(() => {
    const { nodes: newNodes, edges: newEdges } = yamlToGraph(yamlContent);
    setNodes(newNodes);
    setEdges(newEdges);
  }, [yamlContent]);

  // Handle node selection
  const onNodeClick = React.useCallback(
    (_event: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
    },
    [],
  );

  // Handle pane click (deselect)
  const onPaneClick = React.useCallback(() => {
    setSelectedNodeId(null);
  }, []);

  return (
    <div className="flex h-full bg-gradient-to-br from-hector-darker to-black relative">
      {/* Collapsible Infrastructure Sidebar */}
      <InfrastructureSidebar
        yamlContent={yamlContent}
        nodes={nodes}
        collapsed={sidebarCollapsed}
      />

      {/* Toggle Button for Sidebar */}
      <button
        onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
        className="absolute left-0 top-1/2 -translate-y-1/2 z-20 bg-black/60 hover:bg-black/80 border border-white/20 rounded-r-lg p-2 transition-all"
        style={{ left: sidebarCollapsed ? "64px" : "256px" }}
        title={
          sidebarCollapsed
            ? "Show Infrastructure Details"
            : "Hide Infrastructure Details"
        }
      >
        {sidebarCollapsed ? (
          <ChevronRight size={20} className="text-gray-300" />
        ) : (
          <ChevronLeft size={20} className="text-gray-300" />
        )}
      </button>

      {/* Main Canvas */}
      <div className="flex-1 relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          nodeTypes={nodeTypes}
          defaultViewport={{ x: 0, y: 0, zoom: 0.75 }}
          minZoom={0.25}
          maxZoom={1.5}
          fitView={false}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={true}
          className="bg-gradient-to-br from-gray-900 to-gray-800"
          proOptions={{ hideAttribution: true }}
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1}
            color="rgba(255, 255, 255, 0.15)"
          />
        </ReactFlow>
      </div>

      {/* Properties Panel - Read-only */}
      {selectedNodeId && (
        <PropertiesPanel
          nodeId={selectedNodeId}
          node={nodes.find((n) => n.id === selectedNodeId)}
          onClose={() => setSelectedNodeId(null)}
          readonly={true}
        />
      )}
    </div>
  );
};
