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

export type Role = 'user' | 'agent' | 'system';

export interface Attachment {
    id: string;
    file: File;
    preview: string;
    base64: string;
    mediaType: string;
}

export interface ToolCall {
    id: string;
    name: string;
    args: Record<string, any>;
    status: 'pending' | 'working' | 'success' | 'failed';
    result?: string;
    error?: string;
}

export interface ThinkingBlock {
    id: string;
    type: 'todo' | 'goal' | 'reflection' | 'default';
    content: string;
    status: 'active' | 'completed';
}

export interface ApprovalRequest {
    id: string;
    toolName: string;
    toolInput: Record<string, any>;
    options?: string[];
    status: 'pending' | 'decided';
    decision?: 'approve' | 'deny';
}

export interface ImageWidget {
    id: string;
    url: string;
    revised_prompt?: string;
}

export type WidgetType = 'tool' | 'thinking' | 'approval' | 'image';

export interface Widget {
    id: string;
    type: WidgetType;
    data: any;
    status: string;
    content?: string;
    isExpanded: boolean;
    decision?: string; // For approval
}

export interface Message {
    id: string;
    role: Role;
    text: string;
    metadata: {
        taskId?: string;
        images?: Attachment[];
        contentOrder?: string[]; // Array of widget IDs in order of appearance
        [key: string]: any;
    };
    toolCalls: ToolCall[];
    thinkingBlocks: ThinkingBlock[];
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

// AG-UI Event Types
export interface AGUIEvent {
    event_id: string;
    type: string;
    timestamp: string;
    payload: any;
    metadata?: Record<string, any>;
}
