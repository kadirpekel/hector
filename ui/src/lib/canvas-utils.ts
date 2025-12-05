import type { Node, Edge } from '@xyflow/react';

/**
 * Calculate usage count for infrastructure items
 */
export const calculateUsageCount = (
  infrastructureId: string,
  infrastructureType: 'llm' | 'database' | 'embedder' | 'vector_store',
  nodes: Node[]
): number => {
  if (infrastructureType === 'llm') {
    return nodes.filter((n) => n.data.llm === infrastructureId).length;
  }
  // TODO: Add usage tracking for other infrastructure types
  return 0;
};

/**
 * Get provider icon/color for LLM
 */
export const getProviderIcon = (provider: string): string => {
  switch (provider) {
    case 'openai':
      return '🟢';
    case 'gemini':
      return '🟣';
    case 'anthropic':
      return '🟠';
    case 'ollama':
      return '🔵';
    default:
      return '⚪';
  }
};

/**
 * Auto-layout nodes using simple grid algorithm
 * TODO: Replace with Dagre or ELK for better layouts
 */
export const autoLayoutNodes = (nodes: Node[], edges: Edge[]): Node[] => {
  // Separate LLMs and Agents
  const llmNodes = nodes.filter((n) => n.type === 'llm');
  const agentNodes = nodes.filter((n) => n.type !== 'llm');

  // Layout LLMs in left column
  const layoutLLMs = llmNodes.map((node, i) => ({
    ...node,
    position: { x: 100, y: 100 + i * 180 },
  }));

  // Build agent hierarchy using edges
  const rootAgents = agentNodes.filter((node) => {
    // Root agents are those not referenced as sub-agents
    return !edges.some((e) => e.target === node.id && e.source.startsWith('agent-'));
  });

  const layoutAgents: Node[] = [];
  let currentY = 100;

  const layoutAgentTree = (agentNode: Node, x: number, depth: number) => {
    const positioned = {
      ...agentNode,
      position: { x, y: currentY },
    };
    layoutAgents.push(positioned);
    currentY += 180;

    // Find sub-agents
    const subAgentIds = edges
      .filter((e) => e.source === agentNode.id && e.target.startsWith('agent-'))
      .map((e) => e.target);

    subAgentIds.forEach((subId) => {
      const subNode = agentNodes.find((n) => n.id === subId);
      if (subNode) {
        layoutAgentTree(subNode, x + 300, depth + 1);
      }
    });
  };

  // Layout root agents and their trees
  rootAgents.forEach((rootAgent) => {
    layoutAgentTree(rootAgent, 500, 0);
  });

  return [...layoutLLMs, ...layoutAgents];
};

/**
 * Validate node connections
 */
export const validateConnection = (
  source: Node,
  target: Node,
  edges: Edge[]
): { valid: boolean; error?: string } => {
  // Can't connect to self
  if (source.id === target.id) {
    return { valid: false, error: 'Cannot connect node to itself' };
  }

  // Check for circular dependencies
  const visited = new Set<string>();
  const stack = [target.id];

  while (stack.length > 0) {
    const currentId = stack.pop()!;
    if (currentId === source.id) {
      return { valid: false, error: 'Circular dependency detected' };
    }
    if (visited.has(currentId)) continue;
    visited.add(currentId);

    // Find all outgoing edges from current node
    const outgoing = edges.filter((e) => e.source === currentId);
    outgoing.forEach((e) => stack.push(e.target));
  }

  return { valid: true };
};

