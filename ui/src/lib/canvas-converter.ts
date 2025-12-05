import * as yaml from 'js-yaml';
import type { Node, Edge } from '@xyflow/react';

interface Config {
  version?: string;
  name?: string;
  llms?: Record<string, any>;
  agents?: Record<string, any>;
  databases?: Record<string, any>;
  embedders?: Record<string, any>;
  vector_stores?: Record<string, any>;
  [key: string]: any;
}

export interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

/**
 * Converts YAML config to React Flow visualization
 * Optimized layout with proper workflow group sizing
 */
export const yamlToGraph = (yamlContent: string): GraphData => {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  try {
    const config = yaml.load(yamlContent) as Config;

    if (!config || !config.agents) {
      return { nodes, edges };
    }

    // Layout constants
    const NODE_WIDTH = 220;
    const NODE_HEIGHT = 80;
    const HORIZONTAL_GAP = 80;
    const VERTICAL_GAP = 50;
    const GROUP_PADDING = 60;
    const WORKFLOW_VERTICAL_SPACING = 100;

    let workflowYOffset = 50;

    // Track which agents are in workflows
    const agentsInWorkflows = new Set<string>();
    const workflowNodes = new Map<string, { type: string; subAgents: string[]; maxIterations?: number }>();

    // First pass: Identify workflow agents
    Object.entries(config.agents).forEach(([id, agentConfig]) => {
      if (agentConfig.type && ['sequential', 'parallel', 'loop'].includes(agentConfig.type)) {
        const subAgents = agentConfig.sub_agents || [];
        workflowNodes.set(id, {
          type: agentConfig.type,
          subAgents,
          maxIterations: agentConfig.max_iterations,
        });
        subAgents.forEach((subId: string) => agentsInWorkflows.add(subId));
      }
    });

    // Second pass: Create workflow groups with proper sizing
    workflowNodes.forEach(({ type, subAgents, maxIterations }, workflowId) => {
      const numChildren = subAgents.length;
      
      if (numChildren === 0) return;

      // Calculate group dimensions based on layout type
      let groupWidth: number;
      let groupHeight: number;
      
      if (type === 'sequential') {
        // Horizontal layout: width grows with children
        groupWidth = (numChildren * NODE_WIDTH) + ((numChildren - 1) * HORIZONTAL_GAP) + (GROUP_PADDING * 2);
        groupHeight = NODE_HEIGHT + (GROUP_PADDING * 2) + 40; // Extra for header
      } else if (type === 'parallel') {
        // Vertical layout: height grows with children
        groupWidth = NODE_WIDTH + (GROUP_PADDING * 2);
        groupHeight = (numChildren * NODE_HEIGHT) + ((numChildren - 1) * VERTICAL_GAP) + (GROUP_PADDING * 2) + 40;
      } else {
        // Loop: circular/grid layout
        const cols = Math.ceil(Math.sqrt(numChildren));
        const rows = Math.ceil(numChildren / cols);
        groupWidth = (cols * NODE_WIDTH) + ((cols - 1) * HORIZONTAL_GAP) + (GROUP_PADDING * 2);
        groupHeight = (rows * NODE_HEIGHT) + ((rows - 1) * VERTICAL_GAP) + (GROUP_PADDING * 2) + 40;
      }

      // Create workflow group node
      const groupNode: Node = {
        id: workflowId,
        type: 'workflowGroup',
        position: { x: 100, y: workflowYOffset },
        data: {
          label: workflowId,
          workflowType: type,
          subAgents: subAgents,
          maxIterations: maxIterations,
        },
        style: {
          width: groupWidth,
          height: groupHeight,
          zIndex: -1, // Behind child nodes
        },
      };
      nodes.push(groupNode);

      // Layout child nodes inside the group
      subAgents.forEach((subId, index) => {
        const agentConfig = config.agents![subId];
        if (!agentConfig) return;

        let childX: number;
        let childY: number;

        if (type === 'sequential') {
          // Sequential: horizontal row
          childX = GROUP_PADDING + (index * (NODE_WIDTH + HORIZONTAL_GAP));
          childY = GROUP_PADDING + 40; // Below header
        } else if (type === 'parallel') {
          // Parallel: vertical column
          childX = GROUP_PADDING;
          childY = GROUP_PADDING + 40 + (index * (NODE_HEIGHT + VERTICAL_GAP));
        } else {
          // Loop: grid layout
          const cols = Math.ceil(Math.sqrt(numChildren));
          const col = index % cols;
          const row = Math.floor(index / cols);
          childX = GROUP_PADDING + (col * (NODE_WIDTH + HORIZONTAL_GAP));
          childY = GROUP_PADDING + 40 + (row * (NODE_HEIGHT + VERTICAL_GAP));
        }

        const childNode: Node = {
          id: subId,
          type: 'agent',
          position: { x: childX, y: childY },
          parentId: workflowId,
          extent: 'parent' as const,
          data: {
            label: agentConfig.name || subId,
            llm: agentConfig.llm,
            description: agentConfig.description,
            instruction: agentConfig.instruction,
          },
          style: {
            width: NODE_WIDTH,
            height: NODE_HEIGHT,
          },
        };
        nodes.push(childNode);

        // Create edges within workflow
        if (type === 'sequential' && index < subAgents.length - 1) {
          // Sequential: left to right
          edges.push({
            id: `edge-${subId}-${subAgents[index + 1]}`,
            source: subId,
            target: subAgents[index + 1],
            type: 'smoothstep',
            animated: false,
            style: { stroke: '#3b82f6', strokeWidth: 2 },
            sourceHandle: 'right',
            targetHandle: 'left',
          });
        } else if (type === 'parallel') {
          // Parallel: no edges between children (they run in parallel)
          // Could add edges from a parent trigger if needed
        } else if (type === 'loop' && index < subAgents.length - 1) {
          // Loop: connect in sequence
          edges.push({
            id: `edge-${subId}-${subAgents[index + 1]}`,
            source: subId,
            target: subAgents[index + 1],
            type: 'smoothstep',
            animated: false,
            style: { stroke: '#14b8a6', strokeWidth: 2 },
          });
        }
      });

      // Loop back edge for loop workflows
      if (type === 'loop' && subAgents.length > 1) {
        edges.push({
          id: `edge-loop-back-${workflowId}`,
          source: subAgents[subAgents.length - 1],
          target: subAgents[0],
          type: 'smoothstep',
          animated: true,
          style: { stroke: '#14b8a6', strokeWidth: 2, strokeDasharray: '5 5' },
        });
      }

      // Move Y offset for next workflow
      workflowYOffset += groupHeight + WORKFLOW_VERTICAL_SPACING;
    });

    // Third pass: Standalone agents (not in workflows)
    let standaloneY = workflowYOffset;
    Object.entries(config.agents).forEach(([id, agentConfig]) => {
      if (!agentsInWorkflows.has(id) && !workflowNodes.has(id)) {
        nodes.push({
          id,
          type: 'agent',
          position: { x: 100, y: standaloneY },
          data: {
            label: agentConfig.name || id,
            llm: agentConfig.llm,
            description: agentConfig.description,
            instruction: agentConfig.instruction,
          },
          style: {
            width: NODE_WIDTH,
            height: NODE_HEIGHT,
          },
        });
        standaloneY += NODE_HEIGHT + 50;

        // Create edges for sub_agents
        if (agentConfig.sub_agents) {
          agentConfig.sub_agents.forEach((subId: string) => {
            edges.push({
              id: `edge-${id}-${subId}`,
              source: id,
              target: subId,
              type: 'smoothstep',
              animated: false,
              style: { stroke: '#6b7280', strokeWidth: 2 },
            });
          });
        }
      }
    });

    return { nodes, edges };
  } catch (error) {
    console.error('Failed to parse YAML:', error);
    return { nodes: [], edges: [] };
  }
};

/**
 * Validates YAML for canvas visualization
 */
export const validateYAMLForCanvas = (yamlContent: string): { valid: boolean; error?: string } => {
  try {
    const config = yaml.load(yamlContent) as Config;
    if (!config) {
      return { valid: false, error: 'Empty configuration' };
    }
    return { valid: true };
  } catch (error) {
    return {
      valid: false,
      error: error instanceof Error ? error.message : 'Invalid YAML'
    };
  }
};
