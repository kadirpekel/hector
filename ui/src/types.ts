export interface Agent {
  name: string;
  url: string;
  description?: string;
}

export interface AgentCard {
  name: string;
  description: string;
  version: string;
  default_input_modes: string[];
  capabilities?: string[];
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
}

export interface ThinkingWidgetData {
  type: "todo" | "goal" | "reflection" | "default";
  todos?: TodoItem[];
  block_id?: string;
}

export interface ApprovalWidgetData {
  toolName: string;
  toolInput: Record<string, unknown>;
  options?: string[];
}

export interface TextWidgetData {
  // Text widgets store content in widget.content, not data
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

// Discriminated union for type-safe widget access
export type Widget =
  | ToolWidget
  | ThinkingWidget
  | ApprovalWidget
  | ImageWidget
  | TextWidget;

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
}

// ============================================================================
// AG-UI Protocol Types
// These types define the structure of streaming data from the backend.
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
