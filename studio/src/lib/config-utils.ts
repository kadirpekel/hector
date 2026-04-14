import * as yaml from 'js-yaml';


export interface HectorConfig {
  version?: string;
  name?: string;
  description?: string;
  llms?: Record<string, LLMConfig>;
  tools?: Record<string, ToolConfig>;
  guardrails?: Record<string, GuardrailConfig>;
  embedders?: Record<string, EmbedderConfig>;
  vector_stores?: Record<string, VectorStoreConfig>;
  document_stores?: Record<string, DocumentStoreConfig>;
  agents?: Record<string, AgentConfig>;
  defaults?: {
    llm?: string;
  };
}







export interface LLMConfig {
  provider?: 'anthropic' | 'openai' | 'gemini' | 'ollama';
  model?: string;
  api_key?: string;
  base_url?: string;
  temperature?: number;
  max_tokens?: number;
  max_tool_output_length?: number;
  thinking?: {
    enabled?: boolean;
    budget_tokens?: number;
  };
}

export interface ToolConfig {
  type?: 'mcp' | 'function' | 'command';
  enabled?: boolean;
  description?: string;
  url?: string;
  transport?: 'stdio' | 'sse' | 'streamable-http';
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  filter?: string[];
  handler?: string;
  parameters?: Record<string, any>;
  allowed_commands?: string[];
  denied_commands?: string[];
  working_directory?: string;
  max_execution_time?: string;
  deny_by_default?: boolean;
  require_approval?: boolean;
  approval_prompt?: string;
}

export interface GuardrailConfig {
  enabled?: boolean;
  input?: {
    chain_mode?: 'fail_fast' | 'collect_all';
    length?: { enabled?: boolean; min_length?: number; max_length?: number; action?: 'allow' | 'block' | 'warn'; severity?: 'low' | 'medium' | 'high' | 'critical' };
    injection?: { enabled?: boolean; patterns?: string[]; case_sensitive?: boolean; action?: 'allow' | 'block' | 'warn'; severity?: 'low' | 'medium' | 'high' | 'critical' };
    sanitizer?: { enabled?: boolean; trim_whitespace?: boolean; normalize_unicode?: boolean; max_length?: number; strip_html?: boolean };
    pattern?: { enabled?: boolean; allow_patterns?: string[]; block_patterns?: string[]; action?: 'allow' | 'block' | 'warn'; severity?: 'low' | 'medium' | 'high' | 'critical' };
  };
  output?: {
    chain_mode?: 'fail_fast' | 'collect_all';
    pii?: {
      enabled?: boolean;
      detect_email?: boolean;
      detect_phone?: boolean;
      detect_ssn?: boolean;
      detect_credit_card?: boolean;
      redact_mode?: 'mask' | 'remove' | 'hash';
      action?: 'allow' | 'block' | 'modify' | 'warn';
      severity?: 'low' | 'medium' | 'high' | 'critical';
    };
    content?: {
      enabled?: boolean;
      blocked_keywords?: string[];
      blocked_patterns?: string[];
      action?: 'allow' | 'block' | 'warn';
      severity?: 'low' | 'medium' | 'high' | 'critical';
    };
  };
  tool?: {
    chain_mode?: 'fail_fast' | 'collect_all';
    authorization?: {
      enabled?: boolean;
      allowed_tools?: string[];
      blocked_tools?: string[];
      action?: 'allow' | 'block' | 'warn';
      severity?: 'low' | 'medium' | 'high' | 'critical';
    };
  };
  moderation?: {
    enabled?: boolean;
    strategy?: 'none' | 'openai' | 'lakera' | 'prompt';
    action?: 'block' | 'warn';
    openai?: { model?: string; threshold?: number };
    lakera?: { project_id?: string; breakdown?: boolean; endpoint?: string };
    prompt?: { llm?: string; template?: string; safe_field?: string };
  };
}



export interface EmbedderConfig {
  provider?: string;
  model?: string;
  api_key?: string;
  base_url?: string;
  dimension?: number;
  timeout?: number;
  batch_size?: number;
  encoding_format?: string;
  user?: string;
  input_type?: string;
  output_dimension?: number;
  truncate?: string;
}

// Top-level vector store config (flat format for v1.14.0+)
export interface VectorStoreConfig {
  type: string;
  host?: string;
  port?: number;
  api_key?: string;
  enable_tls?: boolean;
  persist_path?: string;
  compress?: boolean;
  collection?: string;
  index_name?: string;
  environment?: string;
  // New in v1.18.0: Control persistence for chromem (default: true)
  enable_persistence?: boolean;
}

export interface DocumentStoreConfig {
  embedder?: string;
  vector_store?: string;
  collection?: string;
  watch?: boolean;
  incremental_indexing?: boolean;
  source?: {
    type?: string;
    include?: string[];
    exclude?: string[];
    max_file_size?: number;
    sql?: {
      dsn?: string;
      tables?: Array<{
        table: string;
        columns: string[];
        id_column: string;
        updated_column?: string;
        where_clause?: string;
        metadata_columns?: string[];
      }>;
    };
    api?: {
      url: string;
      headers?: Record<string, string>;
      id_field: string;
      content_field: string;
    };
    blob?: {
      url: string;
      prefix?: string;
    };
    collection?: string;
  };
  chunking?: {
    strategy?: string;
    size?: number;
    overlap?: number;
    min_size?: number;
    max_size?: number;
    preserve_words?: boolean;
  };
  search?: {
    top_k?: number;
    threshold?: number;
    enable_hyde?: boolean;
    hyde_llm?: string;
    enable_rerank?: boolean;
    rerank_llm?: string;
    rerank_max_results?: number;
    enable_multi_query?: boolean;
    multi_query_llm?: string;
    multi_query_count?: number;
  };
  indexing?: {
    max_concurrent?: number;
    retry?: { max_retries?: number; base_delay?: number; max_delay?: number; jitter?: number };
  };
  mcp_parsers?: {
    tool_names: string[];
    extensions?: string[];
    priority?: number;
    prefer_native?: boolean;
    path_prefix?: string;
  };
}

export interface AgentConfig {
  name?: string;
  description?: string;
  type?: 'llm' | 'sequential' | 'parallel' | 'loop' | 'remote' | 'runner' | 'conditional';
  llm?: string;
  tools?: string[];
  sub_agents?: string[];
  agent_tools?: string[];
  instruction?: string;
  instruction_file?: string;
  global_instruction?: string;
  guardrails?: string;
  disable_safety_protocols?: boolean;
  document_stores?: string[];
  visibility?: 'public' | 'internal' | 'private';
  streaming?: boolean;
  include_context?: boolean;
  include_context_limit?: number;
  include_context_max_length?: number;
  prompt?: {
    system_prompt?: string;
    role?: string;
    guidance?: string;
  };
  context?: {
    strategy?: 'none' | 'buffer_window' | 'token_window' | 'summary_buffer';
    window_size?: number;
    budget?: number;
    threshold?: number;
    target?: number;
    preserve_recent?: number;
    summarizer_llm?: string;
  };
  reasoning?: {
    max_iterations?: number;
    enable_exit_tool?: boolean;
    enable_escalate_tool?: boolean;
    termination_conditions?: string[];
    completion_instruction?: string;
  };
  structured_output?: { schema?: Record<string, any>; strict?: boolean; name?: string };
  skills?: Array<{ id?: string; name?: string; description?: string; tags?: string[]; examples?: string[] }>;
  input_modes?: string[];
  output_modes?: string[];
  // Remote agent
  url?: string;
  agent_card_url?: string;
  agent_card_file?: string;
  headers?: Record<string, string>;
  timeout?: string;
  // Workflow
  max_iterations?: number;
  // Conditional Agent
  condition_agent?: string;
  condition_field?: string;
  on_true_agent?: string;
  on_false_response?: string;
  // Trigger
  trigger?: TriggerConfig;
  // Per-agent notifications
  notifications?: NotificationConfig[];
}

export interface TriggerConfig {
  type: 'schedule' | 'webhook';
  enabled?: boolean;
  // Schedule trigger fields
  cron?: string;
  timezone?: string;
  input?: string;
  // Webhook trigger fields
  path?: string;
  methods?: string[];
  secret?: string;
  signature_header?: string;
  webhook_input?: WebhookInputConfig;
  response?: WebhookResponseConfig;
}

export interface WebhookInputConfig {
  template?: string;
  extract_fields?: { path: string; as: string }[];
}

export interface WebhookResponseConfig {
  mode?: 'sync' | 'async' | 'callback';
  timeout?: number;
  callback_url?: string;
}

export interface NotificationConfig {
  id: string;
  type?: 'webhook';
  enabled?: boolean;
  events?: string[];
  url: string;
  headers?: Record<string, string>;
  payload?: { template?: string };
  retry?: {
    max_attempts?: number;
    initial_delay?: number;
    max_delay?: number;
  };
}

/**
 * Parse YAML string to HectorConfig
 */
export function parseConfig(yamlContent: string): HectorConfig {
  try {
    return (yaml.load(yamlContent) || {}) as HectorConfig;
  } catch (e) {
    console.error('Failed to parse YAML:', e);
    return {};
  }
}

/**
 * Serialize HectorConfig to YAML string
 */
export function serializeConfig(config: HectorConfig): string {
  return yaml.dump(config, { indent: 2, lineWidth: -1, noRefs: true });
}

/**
 * Update a specific section in the config
 */
export function updateConfigSection(
  yamlContent: string,
  section: keyof HectorConfig,
  key: string,
  value: any
): string {
  const config = parseConfig(yamlContent);
  if (!config[section]) {
    (config as any)[section] = {};
  }
  (config[section] as Record<string, any>)[key] = value;
  return serializeConfig(config);
}

/**
 * Remove an item from a config section
 */
export function removeFromConfigSection(
  yamlContent: string,
  section: keyof HectorConfig,
  key: string
): string {
  const config = parseConfig(yamlContent);
  if (config[section] && (config[section] as Record<string, any>)[key]) {
    delete (config[section] as Record<string, any>)[key];
  }
  return serializeConfig(config);
}

/**
 * Add a new agent to the config
 */
export function addAgent(yamlContent: string, agentId: string, agentConfig: AgentConfig): string {
  return updateConfigSection(yamlContent, 'agents', agentId, agentConfig);
}

/**
 * Remove an agent from the config
 */
export function removeAgent(yamlContent: string, agentId: string): string {
  const config = parseConfig(yamlContent);
  
  // Remove the agent
  if (config.agents?.[agentId]) {
    delete config.agents[agentId];
  }
  
  // Also remove references to this agent in other agents' sub_agents and agent_tools
  if (config.agents) {
    for (const agent of Object.values(config.agents)) {
      if (agent.sub_agents) {
        agent.sub_agents = agent.sub_agents.filter(id => id !== agentId);
      }
      if (agent.agent_tools) {
        agent.agent_tools = agent.agent_tools.filter(id => id !== agentId);
      }
    }
  }
  
  return serializeConfig(config);
}

/**
 * Update an agent in the config
 */
export function updateAgent(yamlContent: string, agentId: string, updates: Partial<AgentConfig>): string {
  const config = parseConfig(yamlContent);
  if (config.agents?.[agentId]) {
    // Clean up context fields based on strategy
    if (updates.context) {
      const ctx = { ...updates.context };
      // Only keep relevant fields for each strategy
      if (ctx.strategy === 'none' || !ctx.strategy) {
        // None strategy - remove all context config
        delete updates.context;
      } else if (ctx.strategy === 'buffer_window') {
        // Buffer window only needs window_size
        delete ctx.budget;
        delete ctx.threshold;
        delete ctx.target;
        delete ctx.preserve_recent;
        delete ctx.summarizer_llm;
        if (!ctx.window_size) delete ctx.window_size;
        updates.context = ctx;
      } else if (ctx.strategy === 'token_window') {
        // Token window needs budget and optionally preserve_recent
        delete ctx.window_size;
        delete ctx.threshold;
        delete ctx.target;
        delete ctx.summarizer_llm;
        if (!ctx.budget) delete ctx.budget;
        if (!ctx.preserve_recent) delete ctx.preserve_recent;
        updates.context = ctx;
      } else if (ctx.strategy === 'summary_buffer') {
        // Summary buffer needs budget, threshold, target, summarizer_llm
        delete ctx.window_size;
        if (!ctx.budget) delete ctx.budget;
        if (!ctx.threshold) delete ctx.threshold;
        if (!ctx.target) delete ctx.target;
        if (!ctx.preserve_recent) delete ctx.preserve_recent;
        updates.context = ctx;
      }
    }
    config.agents[agentId] = { ...config.agents[agentId], ...updates };
  }
  return serializeConfig(config);
}

/**
 * Get list of LLM names from config
 */
export function getLLMNames(yamlContent: string): string[] {
  const config = parseConfig(yamlContent);
  return Object.keys(config.llms || {});
}

/**
 * Get list of Tool names from config
 */
export function getToolNames(yamlContent: string): string[] {
  const config = parseConfig(yamlContent);
  return Object.keys(config.tools || {});
}

/**
 * Get list of Guardrail names from config
 */
export function getGuardrailNames(yamlContent: string): string[] {
  const config = parseConfig(yamlContent);
  return Object.keys(config.guardrails || {});
}

/**
 * Get list of Document Store names from config
 */
export function getDocumentStoreNames(yamlContent: string): string[] {
  const config = parseConfig(yamlContent);
  return Object.keys(config.document_stores || {});
}

/**
 * Get list of Agent names from config
 */
export function getAgentNames(yamlContent: string): string[] {
  const config = parseConfig(yamlContent);
  return Object.keys(config.agents || {});
}

/**
 * Generate a unique ID for a new resource
 */
export function generateResourceId(prefix: string, existingIds: string[]): string {
  let counter = 1;
  let id = prefix;
  while (existingIds.includes(id)) {
    id = `${prefix}_${counter}`;
    counter++;
  }
  return id;
}
