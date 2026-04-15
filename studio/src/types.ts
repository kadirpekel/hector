// A2A spec compliant Agent type (from discovery endpoint)
export interface Agent {
  name: string;
  url: string;
  description: string;
  version: string;
  protocolVersion?: string;
  preferredTransport?: string;
  capabilities?: AgentCapabilities;
  defaultInputModes?: string[];
  defaultOutputModes?: string[];
  skills?: AgentSkill[];
  type?: string; // e.g. 'conditional', 'llm', 'remote'
}

// A2A spec compliant AgentCapabilities
export interface AgentCapabilities {
  streaming?: boolean;
  pushNotifications?: boolean;
  stateTransitionHistory?: boolean;
}

// A2A spec compliant AgentSkill
export interface AgentSkill {
  id: string;
  name: string;
  description: string;
  tags?: string[];
  examples?: string[];
  inputModes?: string[];
  outputModes?: string[];
}

// A2A spec compliant AgentCard (full details)
export interface AgentCard extends Agent {
  // AgentCard is Agent with all fields populated
}

export type Role = "user" | "agent" | "system";

export interface Attachment {
  id: string;
  file: File;
  preview: string;
  base64: string;
  mediaType: string;
}

export interface ApprovalRequest {
  id: string;
  toolName: string;
  toolInput: Record<string, any>;
  options?: string[];
  status: "pending" | "decided";
  decision?: "approve" | "deny";
}

export interface TodoItem {
  id: string;
  content: string;
  status: "pending" | "in_progress" | "completed" | "canceled";
}

export interface ImageWidgetData {
  id: string;
  url: string;
  revised_prompt?: string;
}

// Widget-specific data types for type safety
export interface ToolWidgetData {
  name: string;
  args: Record<string, unknown>;
  author?: string;
}

export interface ThinkingWidgetData {
  type: "todo" | "goal" | "reflection" | "default";
  todos?: TodoItem[];
  block_id?: string;
  author?: string;
}

export interface ApprovalWidgetData {
  // Required for specific tool approval
  toolName?: string;
  toolInput?: Record<string, unknown>;
  options?: string[];

  // HITL-specific fields
  task_id?: string;
  tool_call_ids?: string[];
  prompt?: string;
}

export interface TextWidgetData {
  // Text widgets store content in widget.content, not data
  author?: string;
}

export type WidgetType = "tool" | "thinking" | "approval" | "image" | "text";

// Widget status types for better type safety
export type ToolWidgetStatus = "pending" | "working" | "success" | "failed";
export type ThinkingWidgetStatus = "active" | "completed";
export type ApprovalWidgetStatus = "pending" | "decided";
export type ImageWidgetStatus = "loading" | "loaded" | "error";
export type TextWidgetStatus = "active" | "completed";

// Base widget interface
interface BaseWidget {
  id: string;
  content?: string;
  isExpanded: boolean;
}

// Specific widget types for component props
export type ToolWidget = BaseWidget & {
  type: "tool";
  data: ToolWidgetData;
  status: ToolWidgetStatus;
};

export type ThinkingWidget = BaseWidget & {
  type: "thinking";
  data: ThinkingWidgetData;
  status: ThinkingWidgetStatus;
};

export type ApprovalWidget = BaseWidget & {
  type: "approval";
  data: ApprovalWidgetData;
  status: ApprovalWidgetStatus;
  decision?: "approve" | "deny";
};

export type ImageWidget = BaseWidget & {
  type: "image";
  data: ImageWidgetData;
  status: ImageWidgetStatus;
};

export type TextWidget = BaseWidget & {
  type: "text";
  data: TextWidgetData;
  status: TextWidgetStatus;
};

export interface ProgressWidgetData {
  id: string;
  message: string;
  author?: string;
}

export type ProgressWidgetStatus = "active" | "completed" | "failed";

export type ProgressWidget = BaseWidget & {
  type: "progress";
  data: ProgressWidgetData;
  status: ProgressWidgetStatus;
};

// Discriminated union for type-safe widget access
export type Widget =
  | ToolWidget
  | ThinkingWidget
  | ApprovalWidget
  | ImageWidget
  | TextWidget
  | ProgressWidget;

export interface Message {
  id: string;
  role: Role;
  text: string;
  metadata: {
    taskId?: string;
    images?: Attachment[];
    contentOrder?: string[]; // Array of widget IDs in order of appearance
    [key: string]: unknown;
  };
  /** Primary data structure for all widget types (tools, thinking, approval, etc.) */
  widgets: Widget[];
  time: string;
  cancelled?: boolean;
}

export interface Session {
  id: string;
  title: string;
  created: string;
  messages: Message[];
  contextId: string;
  taskId: string | null;
  serverId?: string; // Optional for backward compatibility with legacy sessions
  appId?: string; // App this session belongs to (UUID from app token)
  agentName?: string; // Agent this session belongs to
  
  // Cursor tracking for stateless operation
  // Backend is single source of truth - cursor allows efficient resume
  lastEventId?: string; // Last processed event ID
  eventsLoaded?: boolean; // Whether events have been loaded from backend
}

// ============================================================================
// Session Event Types - A2A Native Format (lowercase keys)
// Backend returns events in this format directly.
// ============================================================================

export interface SessionEvent {
  id: string;
  timestamp: string;
  invocation_id?: string;
  author: string;
  agent_id?: string;
  branch?: string;
  partial?: boolean;
  turn_complete?: boolean;
  interrupted?: boolean;
  
  // Artifact contains message parts (A2A native)
  artifact?: {
    artifactId?: string;
    parts: Array<{ kind?: string; type?: string; text?: string; data?: any }>;
  };
  
  // Metadata contains contextual blocks (thinking, tool_calls, tool_results)
  metadata?: {
    thinking?: { id: string; content?: string; status?: string; type?: string };
    tool_calls?: Array<{ id: string; name: string; args?: any; status?: string }>;
    tool_results?: Array<{ tool_call_id: string; content: string; status?: string; is_error?: boolean }>;
    author?: string;
    agent_id?: string;
    [key: string]: any;
  };
}

// ============================================================================
// AG-UI Protocol Types
// This types define the structure of streaming data from the backend.
// ============================================================================

/** Metadata attached to each message part */
export interface AGUIPartMetadata {
  event_type?: string;
  block_type?: string;
  tool_call_id?: string;
  tool_name?: string;
  thinking_type?: string;
  block_id?: string;
  is_error?: boolean;
  url?: string;
  revised_prompt?: string;
}

/** Data payload within a message part */
export interface AGUIPartData {
  data?: {
    interaction_type?: string;
    approval_id?: string;
    tool_name?: string;
    tool_input?: Record<string, unknown>;
    options?: string[];
    id?: string;
    name?: string;
    arguments?: Record<string, unknown>;
    text?: string;
    content?: string;
    tool_call_id?: string;
    thinking_id?: string;
  };
}

/** A single part of a streamed message */
export interface AGUIPart {
  metadata?: AGUIPartMetadata;
  data?: AGUIPartData;
  text?: string;
}

/** Status update within a stream response */
export interface AGUIStatusUpdate {
  taskId?: string;
  status?: {
    state?: string;
    update?: {
      parts?: AGUIPart[];
      [key: string]: unknown;
    };
  };
}

/** Message payload within a stream response */
export interface AGUIMessage {
  taskId?: string;
  parts?: AGUIPart[];
  [key: string]: unknown;
}

/** Root structure of a stream data packet */
export interface AGUIStreamData {
  result?: {
    statusUpdate?: AGUIStatusUpdate;
    message?: AGUIMessage;
    parts?: AGUIPart[];
    [key: string]: unknown;
  };
  statusUpdate?: AGUIStatusUpdate;
  message?: AGUIMessage;
  parts?: AGUIPart[];
  [key: string]: unknown;
}

// ============================================================================
// Auth & Server Types
// ============================================================================

export interface AuthConfig {
  enabled: boolean;
  type: string;
  jwksUrl?: string;
  issuer: string;
  audience: string;
  clientId?: string;
}

// Runtime types
export type ServerType = 'remote' | 'managed-cloud';

// ============================================================================
// Server Configuration Types
// ============================================================================

export interface ServerConfig {
  id: string;
  name: string;
  url: string;
  adminKey?: string;    // Admin key for /admin/* endpoints
  type?: ServerType;
  cloud?: {
    instanceId?: string;
    region?: string;
  };
  envVars?: Record<string, string>;
}

// ============================================================================
// Cloud Instance Types (from hector-cloud API)
// ============================================================================

export interface CloudInstance {
  id: string;
  tenant_id: string;
  app_name: string;
  host: string;
  status: CloudInstanceStatus;
  external_id?: string;
  auth_secret?: string;    // Only returned on initial provision
  database?: string;       // Only returned on initial provision
  region?: string;
  created_at: string;
  config?: CloudInstanceConfig;
}

export type CloudInstanceStatus =
  | 'provisioning'
  | 'running'
  | 'suspended'
  | 'stopped'
  | 'failed';

export interface CloudInstanceConfig {
  env?: Record<string, string>;
  auth?: {
    jwks_url?: string;
    issuer?: string;
    audience?: string;
    client_id?: string;
  };
  server?: CloudServerConfig;
}

export interface CloudRateLimitRule {
  type: 'count' | 'token';
  window: 'minute' | 'hour' | 'day' | 'week' | 'month';
  limit: number;
}

export interface CloudRateLimitConfig {
  enabled: boolean;
  scope?: 'session' | 'user';
  limits?: CloudRateLimitRule[];
  ip_headers?: string[];
}

export interface CloudQueueConfig {
  workers?: number;
  max_retries?: number;
  initial_delay?: string;
  max_delay?: string;
  stale_threshold?: string;
}

export interface CloudObservabilityConfig {
  metrics_enabled?: boolean;
  tracing_endpoint?: string;
}

export interface CloudLoggingConfig {
  level?: 'debug' | 'info' | 'warn' | 'error';
  format?: 'json' | 'text';
}

export interface CloudServerConfig {
  rate_limit?: CloudRateLimitConfig;
  queue?: CloudQueueConfig;
  observability?: CloudObservabilityConfig;
  logging?: CloudLoggingConfig;
}

export interface CloudAuthToken {
  access_token: string;
  token_type: string;
  expires_in: number;
}

/**
 * App JWT token returned from Hector when creating an app
 */
export interface AppToken {
  appId: string;
  accessToken: string;  // JWT with tenant_id claim
  tokenType: string;
  issuer: string;
}

// Server Status for UI state machine
export type ServerStatus = 
  | 'added'           // Just added, discovering...
  | 'checking'        // Starting/connecting...
  | 'auth_required'   // Needs login
  | 'authenticated'   // Ready to use
  | 'disconnected'    // Connection lost
  | 'error'           // Error state
  | 'stopping'        // Stopping process
  | 'stopped'         // Gracefully stopped
  | 'unreachable';    // Server unavailable

// Per-server UI state (renderer-side)
export interface ServerState {
  config: ServerConfig;
  status: ServerStatus;
  statusUpdatedAt: number; // Timestamp to prevent stale status updates
  startedAt?: number; // Timestamp when status became 'checking' (for timeout calculation)
  agents: Agent[];
  configYaml: string | null;
  lastError: string | null;
}
