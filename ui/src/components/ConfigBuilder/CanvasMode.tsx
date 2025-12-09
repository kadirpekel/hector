import React from "react";
import {
  ReactFlow,
  Background,
  type Node,
  type Edge,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";


import { PropertiesPanel } from "./PropertiesPanel";
import { AgentNode } from "./nodes/AgentNode";
import { WorkflowGroupNode } from "./nodes/WorkflowGroupNode";
import { yamlToGraph } from "../../lib/canvas-converter";
import { useStore } from "../../store/useStore";

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

  // Use global store for selection
  const selectedNodeId = useStore((state) => state.selectedNodeId);
  const setSelectedNodeId = useStore((state) => state.setSelectedNodeId);

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




      {/* Main Canvas */}
      <div className="flex-1 relative">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          nodeTypes={nodeTypes}
          defaultViewport={{ x: 0, y: 0, zoom: 1.0 }}
          minZoom={0.1}
          maxZoom={2.0}
          fitView={false}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={true}
          className="bg-gradient-to-br from-gray-900 to-gray-800"
          proOptions={{ hideAttribution: true }}
          disableKeyboardA11y={true} // CRITICAL: Stop stealing focus/keys from Editor
          panOnScroll={true}
          panOnDrag={true}
          zoomOnScroll={true}
          preventScrolling={false}
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={24}
            size={1}
            color="rgba(255, 255, 255, 0.1)"
            style={{ backgroundColor: "#111" }}
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
