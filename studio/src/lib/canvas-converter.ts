import * as yaml from 'js-yaml';
import type { Node, Edge } from '@xyflow/react';
import Dagre from '@dagrejs/dagre';
import { parseConfig, type AgentConfig, type HectorConfig } from './config-utils';

export interface GraphData {
  nodes: Node[];
  edges: Edge[];
}

// Layout constants
const NODE_WIDTH = 220;
const NODE_HEIGHT = 80;
const HORIZONTAL_GAP = 40;
const VERTICAL_GAP = 40;
const GROUP_PADDING = 60;

/**
 * Apply dagre auto-layout to ROOT nodes only (those without parentId)
 */
const applyDagreLayout = (
  nodes: Node[], 
  edges: Edge[], 
  direction: 'TB' | 'LR' = 'TB'
): Node[] => {
  // Only layout root nodes
  const rootNodes = nodes.filter(n => !n.parentId);
  const childNodes = nodes.filter(n => n.parentId);
  
  if (rootNodes.length === 0) return nodes;

  const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: direction, nodesep: 80, ranksep: 100, marginx: 50, marginy: 50 });

  // Add root nodes to dagre
  rootNodes.forEach((node) => {
    const width = (node.style?.width as number) || NODE_WIDTH;
    const height = (node.style?.height as number) || NODE_HEIGHT;
    g.setNode(node.id, { width, height });
  });

  // Add edges between root nodes only
  edges.forEach((edge) => {
    const sourceIsRoot = rootNodes.some(n => n.id === edge.source);
    const targetIsRoot = rootNodes.some(n => n.id === edge.target);
    if (sourceIsRoot && targetIsRoot) {
      g.setEdge(edge.source, edge.target);
    }
  });

  // Run layout
  Dagre.layout(g);

  // Apply positions back to root nodes
  const positionedRootNodes = rootNodes.map((node) => {
    const nodeWithPosition = g.node(node.id);
    if (!nodeWithPosition) return node;
    
    const width = (node.style?.width as number) || NODE_WIDTH;
    const height = (node.style?.height as number) || NODE_HEIGHT;
    
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - width / 2,
        y: nodeWithPosition.y - height / 2,
      },
    };
  });

  return [...positionedRootNodes, ...childNodes];
};

/**
 * Calculate dimensions needed for a workflow container based on its children
 * Recursively handles nested workflows
 */
const calculateWorkflowDimensions = (
  subAgents: string[],
  workflowType: string,
  agents?: Record<string, AgentConfig>
): { width: number; height: number } => {
  if (!subAgents || subAgents.length === 0) {
    return { width: NODE_WIDTH + GROUP_PADDING * 2, height: NODE_HEIGHT + GROUP_PADDING * 2 };
  }

  // Calculate dimensions for each child (may be nested workflows)
  const childDimensions = subAgents.map(subAgentId => {
    const subAgent = agents?.[subAgentId];
    if (!subAgent) {
      return { width: NODE_WIDTH, height: NODE_HEIGHT };
    }
    
    const subType = subAgent.type || 'llm';
    const isNestedWorkflow = ['sequential', 'parallel', 'loop'].includes(subType);
    
    if (isNestedWorkflow && subAgent.sub_agents) {
      // Recursively calculate for nested workflow
      return calculateWorkflowDimensions(subAgent.sub_agents, subType, agents);
    }
    
    return { width: NODE_WIDTH, height: NODE_HEIGHT };
  });

  if (workflowType === 'sequential') {
    // Sequential: horizontal layout - sum widths, max height
    const totalWidth = childDimensions.reduce((sum, d) => sum + d.width, 0) 
      + (childDimensions.length - 1) * HORIZONTAL_GAP + GROUP_PADDING * 2;
    const maxHeight = Math.max(...childDimensions.map(d => d.height));
    const totalHeight = maxHeight + GROUP_PADDING * 2 + 30; // +30 for header
    return { width: totalWidth, height: totalHeight };
  } else {
    // Parallel/Loop: vertical layout - max width, sum heights
    const maxWidth = Math.max(...childDimensions.map(d => d.width));
    const totalWidth = maxWidth + GROUP_PADDING * 2;
    const totalHeight = childDimensions.reduce((sum, d) => sum + d.height, 0)
      + (childDimensions.length - 1) * VERTICAL_GAP + GROUP_PADDING * 2 + 30;
    return { width: totalWidth, height: totalHeight };
  }
};

/**
 * Recursively creates child nodes for a workflow container
 * Handles nested workflows by creating them as workflowGroup nodes with their own children
 */
const createWorkflowChildren = (
  parentAgentId: string,
  parentType: string,
  subAgentIds: string[],
  agents: Record<string, AgentConfig>,
  nodes: Node[]
): void => {
  let currentOffset = 0;

  subAgentIds.forEach((subAgentId: string) => {
    const subAgent = agents[subAgentId];
    if (!subAgent) return;

    const subAgentType = subAgent.type || 'llm';
    const isNestedWorkflow = ['sequential', 'parallel', 'loop'].includes(subAgentType);
    const isSubAgentRunner = subAgentType === 'runner';

    // Calculate this child's dimensions
    let childWidth = NODE_WIDTH;
    let childHeight = NODE_HEIGHT;

    if (isNestedWorkflow && subAgent.sub_agents) {
      const nestedDims = calculateWorkflowDimensions(
        subAgent.sub_agents,
        subAgentType,
        agents
      );
      childWidth = nestedDims.width;
      childHeight = nestedDims.height;
    }

    // Position based on parent layout direction
    let childX: number, childY: number;

    if (parentType === 'sequential') {
      // Horizontal layout - accumulate width
      childX = GROUP_PADDING + currentOffset;
      childY = GROUP_PADDING + 30;
      currentOffset += childWidth + HORIZONTAL_GAP;
    } else {
      // Vertical layout for parallel/loop - accumulate height
      childX = GROUP_PADDING;
      childY = GROUP_PADDING + 30 + currentOffset;
      currentOffset += childHeight + VERTICAL_GAP;
    }

    if (isNestedWorkflow) {
      // Create nested workflow container
      nodes.push({
        id: subAgentId,
        type: 'workflowGroup',
        position: { x: childX, y: childY },
        parentId: parentAgentId,
        extent: 'parent' as const,
        data: {
          label: subAgent.name || subAgentId,
          agentId: subAgentId,
          agentType: subAgentType,
          workflowType: subAgentType,
          subAgents: subAgent.sub_agents || [],
          maxIterations: (subAgent as any).loop?.max_iterations,
          config: subAgent,
        },
        style: {
          width: childWidth,
          height: childHeight,
        },
      });

      // Recursively create children of this nested workflow
      if (subAgent.sub_agents && subAgent.sub_agents.length > 0) {
        createWorkflowChildren(
          subAgentId,
          subAgentType,
          subAgent.sub_agents,
          agents,
          nodes
        );
      }
    } else {
      // Create regular agent node
      nodes.push({
        id: subAgentId,
        type: isSubAgentRunner ? 'runnerAgent' : 'agent',
        position: { x: childX, y: childY },
        parentId: parentAgentId,
        extent: 'parent' as const,
        data: {
          label: subAgent.name || subAgentId,
          agentId: subAgentId,
          agentType: subAgentType,
          llm: subAgent.llm,
          description: subAgent.description,
          instruction: subAgent.instruction,
          tools: subAgent.tools,
          subAgents: subAgent.sub_agents,
          agentTools: subAgent.agent_tools,
          guardrails: subAgent.guardrails,
          documentStores: subAgent.document_stores,
          isRemote: subAgent.type === 'remote',
          url: subAgent.url,
          agent_card_url: subAgent.agent_card_url,
          timeout: subAgent.timeout,
          headers: subAgent.headers,
          trigger: subAgent.trigger,
          notifications: subAgent.notifications,
          structured_output: subAgent.structured_output,
          prompt: subAgent.prompt,
          context: subAgent.context,
          reasoning: subAgent.reasoning,
          skills: subAgent.skills,
          input_modes: subAgent.input_modes,
          output_modes: subAgent.output_modes,
          config: subAgent,
        },
        style: {
          width: NODE_WIDTH,
          height: NODE_HEIGHT,
        },
      });
    }
  });
};

/**
 * Converts YAML config to React Flow visualization
 * Workflow agents are containers with sub-agents as child nodes
 */
export const yamlToGraph = (yamlContent: string): GraphData => {
  const nodes: Node[] = [];
  const edges: Edge[] = [];

  try {
    const config = parseConfig(yamlContent);

    if (!config || !config.agents || Object.keys(config.agents).length === 0) {
      return { nodes, edges };
    }

    // First pass: identify which agents are sub-agents of workflows
    const subAgentParents: Record<string, string> = {};
    Object.entries(config.agents).forEach(([agentId, agent]) => {
      const agentConfig = agent as AgentConfig;
      const type = agentConfig.type || 'llm';
      const isWorkflow = ['sequential', 'parallel', 'loop'].includes(type);
      
      if (isWorkflow && agentConfig.sub_agents) {
        agentConfig.sub_agents.forEach((subId: string) => {
          subAgentParents[subId] = agentId;
        });
      }
    });

    // Second pass: create all nodes
    Object.entries(config.agents).forEach(([agentId, agent]) => {
      const agentConfig = agent as AgentConfig;
      const type = agentConfig.type || 'llm';
      const isWorkflow = ['sequential', 'parallel', 'loop'].includes(type);
      const isRemote = type === 'remote';
      const parentId = subAgentParents[agentId];

      if (isWorkflow) {
        // Workflow container node
        const dimensions = calculateWorkflowDimensions(
          agentConfig.sub_agents || [],
          type,
          config.agents
        );

        nodes.push({
          id: agentId,
          type: 'workflowGroup',
          position: { x: 0, y: 0 },
          data: {
            label: agentConfig.name || agentId,
            agentId,
            agentType: type,
            workflowType: type,
            subAgents: agentConfig.sub_agents || [],
            maxIterations: (agentConfig as any).loop?.max_iterations,
            config: agentConfig,
          },
          style: {
            width: dimensions.width,
            height: dimensions.height,
          },
        });

        // Create child nodes positioned inside the container (recursive for nested workflows)
        if (agentConfig.sub_agents && config.agents) {
          createWorkflowChildren(
            agentId,
            type,
            agentConfig.sub_agents,
            config.agents,
            nodes
          );
        }
      } else if (!parentId) {
        // Regular agent (not a child of a workflow) - only add if not already added as child
        const isRunner = type === 'runner';
        const isConditional = type === 'conditional';
        
        // Determine node type
        let nodeType = 'agent';
        if (isRunner) nodeType = 'runnerAgent';
        else if (isConditional) nodeType = 'conditionalAgent';
        
        nodes.push({
          id: agentId,
          type: nodeType,
          position: { x: 0, y: 0 },
          data: {
            label: agentConfig.name || agentId,
            agentId,
            agentType: type,
            llm: agentConfig.llm,
            description: agentConfig.description,
            instruction: agentConfig.instruction,
            tools: agentConfig.tools,
            subAgents: agentConfig.sub_agents,
            agentTools: agentConfig.agent_tools,
            guardrails: agentConfig.guardrails,
            documentStores: agentConfig.document_stores,
            isRemote,
            url: agentConfig.url,
            agent_card_url: agentConfig.agent_card_url,
            timeout: agentConfig.timeout,
            headers: agentConfig.headers,
            trigger: agentConfig.trigger,
            notifications: agentConfig.notifications,
            structured_output: agentConfig.structured_output,
            prompt: agentConfig.prompt,
            context: agentConfig.context,
            reasoning: agentConfig.reasoning,
            skills: agentConfig.skills,
            input_modes: agentConfig.input_modes,
            output_modes: agentConfig.output_modes,
            // Conditional agent fields
            condition_agent: agentConfig.condition_agent,
            condition_field: agentConfig.condition_field,
            on_true_agent: agentConfig.on_true_agent,
            on_false_response: agentConfig.on_false_response,
            config: agentConfig,
          },
          style: {
            width: NODE_WIDTH,
            height: NODE_HEIGHT,
          },
        });
      }

      // Create edges for agent_tools (agents used as tools by other agents)
      if (agentConfig.agent_tools) {
        agentConfig.agent_tools.forEach((toolAgentId: string) => {
          edges.push({
            id: `${agentId}-tool->${toolAgentId}`,
            source: agentId,
            target: toolAgentId,
            type: 'smoothstep',
            style: { stroke: '#f59e0b', strokeWidth: 2, strokeDasharray: '3 3' },
            label: 'uses as tool',
            labelStyle: { fill: '#f59e0b', fontSize: 10 },
          });
        });
      }

      // Create edges for conditional agents
      if (type === 'conditional') {
        // Edge from condition_agent to this conditional (evaluates)
        if (agentConfig.condition_agent) {
          edges.push({
            id: `${agentConfig.condition_agent}-evaluates->${agentId}`,
            source: agentConfig.condition_agent,
            target: agentId,
            type: 'smoothstep',
            style: { stroke: '#a855f7', strokeWidth: 2 },
            label: 'evaluates',
            labelStyle: { fill: '#a855f7', fontSize: 10 },
          });
        }
        // Edge from this conditional to on_true_agent (if true)
        if (agentConfig.on_true_agent) {
          edges.push({
            id: `${agentId}-true->${agentConfig.on_true_agent}`,
            source: agentId,
            target: agentConfig.on_true_agent,
            type: 'smoothstep',
            style: { stroke: '#22c55e', strokeWidth: 2 },
            label: 'if true',
            labelStyle: { fill: '#22c55e', fontSize: 10 },
          });
        }
      }
    });

    // Apply dagre layout to root nodes only
    const layoutedNodes = applyDagreLayout(nodes, edges, 'TB');

    return { nodes: layoutedNodes, edges };
  } catch (error) {
    console.error('Failed to parse YAML for graph:', error);
    return { nodes: [], edges: [] };
  }
};

/**
 * Cleans trigger config by removing irrelevant fields based on type.
 * Also removes empty string values.
 */
const cleanTrigger = (trigger: any): any | undefined => {
  if (!trigger || !trigger.type) return undefined;

  const cleaned: any = { type: trigger.type };

  // Common fields
  if (trigger.enabled !== undefined) cleaned.enabled = trigger.enabled;
  if (trigger.input && trigger.input.trim()) cleaned.input = trigger.input;

  if (trigger.type === 'schedule') {
    // Schedule-specific fields
    if (trigger.cron && trigger.cron.trim()) cleaned.cron = trigger.cron;
    if (trigger.timezone && trigger.timezone.trim()) cleaned.timezone = trigger.timezone;
  } else if (trigger.type === 'webhook') {
    // Webhook-specific fields
    if (trigger.path && trigger.path.trim()) cleaned.path = trigger.path;
    if (trigger.response) {
      const resp: any = {};
      if (trigger.response.mode && trigger.response.mode.trim()) resp.mode = trigger.response.mode;
      if (trigger.response.callback_url && trigger.response.callback_url.trim()) {
        resp.callback_url = trigger.response.callback_url;
      }
      if (Object.keys(resp).length > 0) cleaned.response = resp;
    }
  }

  return Object.keys(cleaned).length > 1 ? cleaned : undefined; // Must have at least type + one other field
};

/**
 * Recursively removes empty values (undefined, null, empty string, empty array, empty object)
 * Preserves boolean false and number 0.
 */
const pruneEmpty = (obj: any): any => {
  if (obj === undefined || obj === null || obj === '') {
    return undefined;
  }
  
  if (Array.isArray(obj)) {
    const cleaned = obj.map(pruneEmpty).filter(v => v !== undefined);
    return cleaned.length > 0 ? cleaned : undefined;
  }
  
  if (typeof obj === 'object') {
    const cleaned: any = {};
    for (const [key, value] of Object.entries(obj)) {
      const pruned = pruneEmpty(value);
      if (pruned !== undefined) {
        cleaned[key] = pruned;
      }
    }
    return Object.keys(cleaned).length > 0 ? cleaned : undefined;
  }
  
  return obj;
};

/**
 * Converts React Flow graph back to YAML config
 * Note: This only updates the agents section, preserving other config
 */
export const graphToYaml = (nodes: Node[], existingYaml: string): string => {
  try {
    const config = parseConfig(existingYaml);
    
    // Update agents based on nodes
    const agents: Record<string, AgentConfig> = {};
    
    nodes.forEach((node) => {
      const nodeData = node.data as any;
      if (!nodeData.agentId) return;
      
      // Start with existing config or node config
      const existingAgent = config.agents?.[nodeData.agentId] || nodeData.config || {};
      
      // Clean up structured_output to remove internal schemaStr if a valid parsed schema exists
      // This prevents "duplicate" data in YAML while preserving drafts for invalid JSON
      const sourceStructuredOutput = nodeData.structured_output || existingAgent.structured_output;
      let structuredOutput = sourceStructuredOutput;
      
      if (sourceStructuredOutput && typeof sourceStructuredOutput === 'object') {
        structuredOutput = { ...sourceStructuredOutput };
        // If we have a valid parsed schema, we don't need the string representation
        if (structuredOutput.schema) {
          delete (structuredOutput as any).schemaStr;
        }
      }
      
      const rawAgent: AgentConfig = {
        ...existingAgent,
        name: nodeData.label || existingAgent.name,
        type: nodeData.agentType !== 'llm' ? nodeData.agentType : undefined,
        llm: nodeData.llm || existingAgent.llm,
        description: nodeData.description || existingAgent.description,
        instruction: nodeData.instruction || existingAgent.instruction,
        instruction_file: nodeData.instruction_file || existingAgent.instruction_file,
        tools: nodeData.tools || existingAgent.tools,
        sub_agents: nodeData.subAgents || existingAgent.sub_agents,
        agent_tools: nodeData.agentTools || existingAgent.agent_tools,
        guardrails: nodeData.guardrails || existingAgent.guardrails,
        disable_safety_protocols: nodeData.disable_safety_protocols !== undefined ? nodeData.disable_safety_protocols : existingAgent.disable_safety_protocols,
        document_stores: nodeData.documentStores || existingAgent.document_stores,
        url: nodeData.url || existingAgent.url,
        agent_card_url: nodeData.agent_card_url || existingAgent.agent_card_url,
        timeout: nodeData.timeout || existingAgent.timeout,
        headers: nodeData.headers || existingAgent.headers,
        trigger: cleanTrigger(nodeData.trigger) || cleanTrigger(existingAgent.trigger),
        notifications: (nodeData.notifications?.length > 0 ? nodeData.notifications : undefined) 
          || (existingAgent.notifications?.length > 0 ? existingAgent.notifications : undefined),
        structured_output: structuredOutput,
        prompt: nodeData.prompt || existingAgent.prompt,
        context: nodeData.context || existingAgent.context,
        reasoning: nodeData.reasoning || existingAgent.reasoning,
        skills: nodeData.skills || existingAgent.skills,
        input_modes: nodeData.input_modes || existingAgent.input_modes,
        output_modes: nodeData.output_modes || existingAgent.output_modes,
      };
      
      // Clean undefined and empty values
      const cleanedAgent = pruneEmpty(rawAgent);
      
      if (cleanedAgent) {
        agents[nodeData.agentId] = cleanedAgent;
      }
    });
    
    // Auto-inject built-in tool definitions if referenced but not defined
    const builtInTools = {
      'todo_write': {
        type: 'function',
        handler: 'todo_write',
        description: 'Manage a structured task list to track progress.'
      }
    };

    // Collect all referenced tools across all agents
    const referencedTools = new Set<string>();
    Object.values(agents).forEach(agent => {
      agent.tools?.forEach(tool => referencedTools.add(tool));
    });

    // Ensure built-in tools are defined in the tools section if referenced
    const tools = { ...config.tools };
    let toolsModified = false;

    referencedTools.forEach(toolName => {
      if (!tools[toolName] && (builtInTools as Record<string, any>)[toolName]) {
        tools[toolName] = (builtInTools as Record<string, any>)[toolName];
        toolsModified = true;
      }
    });

    // Preserve non-agent config
    const newConfig: HectorConfig = {
      ...config,
      agents,
      tools: toolsModified ? tools : config.tools,
    };
    
    return yaml.dump(newConfig, { indent: 2, lineWidth: -1, noRefs: true });
  } catch (error) {
    console.error('Failed to convert graph to YAML:', error);
    return existingYaml;
  }
};

/**
 * Validates YAML for canvas visualization
 */
export const validateYAMLForCanvas = (yamlContent: string): { valid: boolean; error?: string } => {
  try {
    const config = parseConfig(yamlContent);
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
