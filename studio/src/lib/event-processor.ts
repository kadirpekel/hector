/**
 * EventProcessor - Unified Event Rendering Engine
 *
 * Processes A2A events into UI widgets regardless of source (streaming or fetched).
 * This is the single source of truth for event-to-widget conversion, ensuring
 * consistent rendering whether events come from:
 *   - Live SSE stream (real-time)
 *   - Backend fetch (historical/resume)
 *
 * Key Architecture:
 * - Stateless processing: Each event is processed independently
 * - Cursor tracking: Maintains lastEventId for efficient resume
 * - Widget deduplication: Uses event IDs for stable widget IDs
 */

import type {
  Widget,
  ToolWidget,
  ThinkingWidget,
  TextWidget,
  ProgressWidget,
  SessionEvent,
  ToolWidgetStatus,
} from "../types";

// A2A Event types (matches backend format)
export interface A2AEvent {
  id: string;
  timestamp: string;
  invocation_id?: string;
  author: string;
  agent_id?: string;
  branch?: string;
  partial?: boolean;
  turn_complete?: boolean;
  interrupted?: boolean;
  artifact?: {
    parts: Array<{
      type?: string;
      kind?: string;
      text?: string;
      data?: Record<string, unknown>;
    }>;
  };
  metadata?: {
    thinking?: {
      id: string;
      status?: string;
      content?: string;
      type?: string;
    };
    tool_calls?: Array<{
      id: string;
      name: string;
      args?: Record<string, unknown>;
      status?: string;
    }>;
    tool_results?: Array<{
      tool_call_id: string;
      content: string | Record<string, unknown>;
      status?: string;
      is_error?: boolean;
    }>;
    progress?: {
      id: string;
      message: string;
      status?: string;
    };
    author?: string;
    agent_id?: string;
    [key: string]: unknown;
  };
}

// Processing result for a single event
export interface ProcessedEvent {
  widgets: Widget[];
  contentOrder: string[];
  text: string;
  author: string;
  invocationId?: string;
  isPartial: boolean;
  isTurnComplete: boolean;
}

// Message grouping result
export interface ProcessedMessage {
  id: string;
  role: "user" | "agent";
  text: string;
  widgets: Widget[];
  metadata: {
    contentOrder?: string[];
    author?: string;
    invocationId?: string;
    [key: string]: unknown;
  };
  time: string;
}

/**
 * Normalize a backend event to A2AEvent format
 * 
 * SessionEvent and A2AEvent now share the same A2A native format (lowercase keys).
 * This function provides type safety and default values.
 */
export function normalizeSessionEvent(event: SessionEvent): A2AEvent {
  return {
    id: event.id,
    timestamp: event.timestamp,
    invocation_id: event.invocation_id,
    author: event.author,
    agent_id: event.agent_id,
    branch: event.branch,
    partial: event.partial ?? false,
    turn_complete: event.turn_complete ?? false,
    interrupted: event.interrupted ?? false,
    artifact: event.artifact,
    metadata: event.metadata ?? {},
  };
}

/**
 * EventProcessor - Core event processing engine
 *
 * Processes events into widgets, maintaining state for widget deduplication
 * and content ordering. Can be used for both streaming and batch processing.
 */
export class EventProcessor {
  // Track created widgets to avoid duplicates
  private createdToolWidgets = new Set<string>();
  private createdThinkingWidgets = new Set<string>();
  private textSegmentCounter = 0;

  // Widget state
  private widgetMap = new Map<string, Widget>();
  private contentOrder: string[] = [];
  private accumulatedText = "";

  // Cursor tracking
  private lastEventId: string | null = null;

  constructor() {
    this.reset();
  }

  /**
   * Reset processor state for a new session/message
   */
  reset(): void {
    this.createdToolWidgets.clear();
    this.createdThinkingWidgets.clear();
    this.textSegmentCounter = 0;
    this.widgetMap.clear();
    this.contentOrder = [];
    this.accumulatedText = "";
    this.lastEventId = null;
  }

  /**
   * Get the cursor for resume operations
   */
  getCursor(): { lastEventId: string | null } {
    return {
      lastEventId: this.lastEventId,
    };
  }

  /**
   * Process a single A2A event and update internal state
   */
  processEvent(event: A2AEvent): void {
    // Update cursor
    this.lastEventId = event.id;

    const invocationId = event.invocation_id || event.metadata?.invocation_id as string;
    const isPartial = event.partial === true;
    const author = event.metadata?.author as string || event.author;

    // Process tool results first (they update existing widgets)
    if (event.metadata?.tool_results) {
      for (const tr of event.metadata.tool_results) {
        this.processToolResult(tr);
      }
    }

    // Process thinking from metadata FIRST (native A2A format sends it here)
    // Thinking logically happens before the response text
    if (event.metadata?.thinking) {
      const t = event.metadata.thinking as {
        id: string;
        content?: string;
        status?: string;
        type?: string;
      };
      this.processThinking(
        t.id,
        t.content || "",
        t.status === "completed" || t.status === "complete",
        t.type,
        author
      );
    }

    // Process artifact parts (text content, tool calls, etc.)
    if (event.artifact?.parts) {
      for (const part of event.artifact.parts) {
        const partKind = part.kind || part.type;

        if (partKind === "text" && part.text) {
          this.processTextPart(part.text, isPartial, author, invocationId);
        } else if ((partKind === "data" || !partKind) && part.data) {
          const data = part.data as Record<string, unknown>;

          if (data.type === "thinking") {
            this.processThinking(
              data.id as string,
              (data.content as string) || "",
              (data.status as string) === "completed",
              data.type as string,
              author
            );
          } else if (data.type === "tool_use") {
            this.processToolCall(data, author);
          } else if (data.type === "tool_result") {
            this.processToolResult({
              tool_call_id: data.tool_call_id as string,
              content: data.content as string,
              is_error: data.is_error as boolean,
              status: data.status as string,
            });
          }
        }
      }
    }

    // Process progress metadata
    if (event.metadata?.progress) {
      this.processProgress(event.metadata.progress, author);
    }

    // Mark widgets as completed if event is not partial
    if (!isPartial) {
      this.widgetMap.forEach((widget, id) => {
        if (widget.type === "thinking" && widget.status === "active") {
          this.widgetMap.set(id, { ...widget, status: "completed" });
        }
        if (widget.type === "text" && widget.status === "active") {
          this.widgetMap.set(id, { ...widget, status: "completed" });
        }
      });
    }
  }

  /**
   * Process text content
   * 
   * For batch processing (fetched events), each event gets its own text widget
   * to ensure all content is displayed.
   */
  private processTextPart(
    text: string,
    isPartial: boolean,
    author?: string,
    _invocationId?: string // Kept for API compatibility but not used
  ): void {
    if (!text) return;

    // For batch processing (fetched events), use event ID to ensure uniqueness
    // This prevents different events from overwriting each other's content
    const stableWidgetId = `text_${this.lastEventId}_${this.textSegmentCounter}`;

    const existingWidget = this.widgetMap.get(stableWidgetId);

    if (existingWidget && existingWidget.type === "text") {
      // Update existing widget (for streaming case where same event updates)
      const currentContent = existingWidget.content || "";
      let newContent: string;

      if (isPartial) {
        // Streaming: append delta
        if (text.startsWith(currentContent)) {
          newContent = text;
        } else {
          newContent = currentContent + text;
        }
      } else {
        // Final: replace with full content
        newContent = text;
      }

      this.widgetMap.set(stableWidgetId, {
        ...existingWidget,
        content: newContent,
        status: isPartial ? "active" : "completed",
        data: { ...existingWidget.data, author: author || existingWidget.data.author },
      });

      this.accumulatedText = newContent;
    } else {
      // Create new text widget
      const widget: TextWidget = {
        id: stableWidgetId,
        type: "text",
        content: text,
        isExpanded: true,
        status: isPartial ? "active" : "completed",
        data: { author },
      };

      this.widgetMap.set(stableWidgetId, widget);
      if (!this.contentOrder.includes(stableWidgetId)) {
        this.contentOrder.push(stableWidgetId);
      }

      this.accumulatedText += text;
    }
  }

  /**
   * Process tool call
   */
  private processToolCall(data: Record<string, unknown>, author?: string): void {
    const id = data.id as string;
    const widgetId = `tool_${id}`;

    if (this.createdToolWidgets.has(id) || this.widgetMap.has(widgetId)) {
      return;
    }

    this.createdToolWidgets.add(id);

    const toolWidget: ToolWidget = {
      id: widgetId,
      type: "tool",
      status: "working",
      content: "",
      data: {
        name: (data.name as string) || "unknown",
        args: (data.arguments || data.input || data.args || {}) as Record<string, unknown>,
        author,
      },
      isExpanded: true,
    };

    this.widgetMap.set(widgetId, toolWidget);
    if (!this.contentOrder.includes(widgetId)) {
      this.contentOrder.push(widgetId);
    }

    // Increment segment counter for text/tool interleaving
    this.textSegmentCounter++;
  }

  /**
   * Process tool result
   */
  private processToolResult(tr: {
    tool_call_id: string;
    content: string | Record<string, unknown>;
    status?: string;
    is_error?: boolean;
  }): void {
    const widgetId = `tool_${tr.tool_call_id}`;
    const existing = this.widgetMap.get(widgetId);

    if (!existing || existing.type !== "tool") {
      // Tool widget not found - create it with result
      const toolWidget: ToolWidget = {
        id: widgetId,
        type: "tool",
        status: tr.is_error ? "failed" : "success",
        content: typeof tr.content === "string" ? tr.content : JSON.stringify(tr.content),
        data: {
          name: "unknown",
          args: {},
        },
        isExpanded: false,
      };
      this.widgetMap.set(widgetId, toolWidget);
      if (!this.contentOrder.includes(widgetId)) {
        this.contentOrder.push(widgetId);
      }
      return;
    }

    const newContent = typeof tr.content === "string" ? tr.content : JSON.stringify(tr.content);
    const existingContent = existing.content || "";

    // Determine if incremental update
    const isIncremental =
      existing.status === "working" &&
      existingContent.length > 0 &&
      newContent.length > 0 &&
      !newContent.includes(existingContent);

    const updatedContent = isIncremental
      ? existingContent + newContent
      : newContent || existingContent;

    let status: ToolWidgetStatus = "success";
    if (tr.is_error) {
      status = "failed";
    } else if (tr.status === "working") {
      status = "working";
    } else if (tr.status === "failed") {
      status = "failed";
    } else if (isIncremental) {
      status = "working";
    }

    this.widgetMap.set(widgetId, {
      ...existing,
      status,
      content: updatedContent,
      isExpanded: status === "working",
    });
  }

  /**
   * Process thinking block
   */
  private processThinking(
    id: string,
    content: string,
    isCompleted: boolean,
    type?: string,
    author?: string
  ): void {
    const widgetId = `thinking_${id}`;

    if (this.createdThinkingWidgets.has(id)) {
      // Update existing
      const existing = this.widgetMap.get(widgetId) as ThinkingWidget | undefined;
      if (existing) {
        const newContent = isCompleted ? content : (existing.content || "") + content;
        this.widgetMap.set(widgetId, {
          ...existing,
          content: newContent,
          status: isCompleted ? "completed" : existing.status,
          isExpanded: isCompleted ? false : existing.isExpanded,
        });
      }
      return;
    }

    this.createdThinkingWidgets.add(id);

    const thinkingWidget: ThinkingWidget = {
      id: widgetId,
      type: "thinking",
      status: isCompleted ? "completed" : "active",
      content,
      data: {
        type: (type || "default") as "todo" | "goal" | "reflection" | "default",
        author,
      },
      isExpanded: !isCompleted,
    };

    this.widgetMap.set(widgetId, thinkingWidget);
    if (!this.contentOrder.includes(widgetId)) {
      this.contentOrder.push(widgetId);
    }

    // Increment segment counter
    this.textSegmentCounter++;
  }

  /**
   * Process progress indicator
   */
  private processProgress(
    data: { id: string; message: string; status?: string },
    author?: string
  ): void {
    const id = data.id;
    if (!id) return;

    const isCompleted = data.status === "completed" || data.status === "success";

    if (!this.widgetMap.has(id)) {
      const widget: ProgressWidget = {
        id,
        type: "progress",
        status: isCompleted ? "completed" : "active",
        content: data.message || "Working...",
        isExpanded: true,
        data: {
          id,
          message: data.message || "",
          author,
        },
      };
      this.widgetMap.set(id, widget);
      if (!this.contentOrder.includes(id)) {
        this.contentOrder.push(id);
      }
    } else {
      const existing = this.widgetMap.get(id) as ProgressWidget;
      this.widgetMap.set(id, {
        ...existing,
        status: isCompleted ? "completed" : "active",
        content: data.message || existing.content,
        data: {
          ...existing.data,
          message: data.message || existing.data.message,
        },
      });
    }
  }

  /**
   * Get current widgets in content order
   */
  getWidgets(): Widget[] {
    const orderedWidgets: Widget[] = [];
    const seenIds = new Set<string>();

    // First, add widgets in content order
    for (const widgetId of this.contentOrder) {
      const widget = this.widgetMap.get(widgetId);
      if (widget) {
        orderedWidgets.push(widget);
        seenIds.add(widgetId);
      }
    }

    // Then add any widgets not in content order (shouldn't happen, but safety)
    this.widgetMap.forEach((widget, id) => {
      if (!seenIds.has(id)) {
        orderedWidgets.push(widget);
      }
    });

    return orderedWidgets;
  }

  /**
   * Get content order array
   */
  getContentOrder(): string[] {
    return [...this.contentOrder];
  }

  /**
   * Get accumulated text
   */
  getAccumulatedText(): string {
    return this.accumulatedText;
  }
}

/**
 * Process a batch of SessionEvents into Messages
 *
 * Groups consecutive events by author into messages, creating widgets
 * using the unified EventProcessor.
 *
 * @param events - Array of SessionEvents from backend
 * @returns Array of processed Messages ready for UI rendering
 */
export function processSessionEvents(events: SessionEvent[]): {
  messages: ProcessedMessage[];
  lastEventId: string | null;
} {
  if (events.length === 0) {
    return { messages: [], lastEventId: null };
  }

  const messages: ProcessedMessage[] = [];
  let currentMessage: ProcessedMessage | null = null;
  let currentProcessor = new EventProcessor();
  let lastAuthorRole: "user" | "agent" | null = null;

  for (const event of events) {
    const normalizedEvent = normalizeSessionEvent(event);
    // Determine role - "user" is user, everything else (agent, assistant, system) is agent
    const role: "user" | "agent" = normalizedEvent.author === "user" ? "user" : "agent";

    // Check if we need to start a new message (role change)
    if (lastAuthorRole !== null && lastAuthorRole !== role) {
      // Finalize current message
      if (currentMessage) {
        currentMessage.widgets = currentProcessor.getWidgets();
        currentMessage.metadata.contentOrder = currentProcessor.getContentOrder();
        
        // For agent messages, text comes from widgets
        if (currentMessage.role === "agent") {
          const textWidgets = currentMessage.widgets.filter(w => w.type === "text");
          currentMessage.text = textWidgets.map(w => w.content || "").join("\n");
        }
        
        messages.push(currentMessage);
      }

      // Start new message with fresh processor
      currentProcessor = new EventProcessor();
      currentMessage = {
        id: normalizedEvent.id,
        role,
        text: role === "user" ? extractTextFromEvent(normalizedEvent) : "",
        widgets: [],
        metadata: {
          author: normalizedEvent.author,
          invocationId: normalizedEvent.invocation_id,
        },
        time: new Date(normalizedEvent.timestamp).toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        }),
      };
    } else if (currentMessage === null) {
      // First message
      currentMessage = {
        id: normalizedEvent.id,
        role,
        text: role === "user" ? extractTextFromEvent(normalizedEvent) : "",
        widgets: [],
        metadata: {
          author: normalizedEvent.author,
          invocationId: normalizedEvent.invocation_id,
        },
        time: new Date(normalizedEvent.timestamp).toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        }),
      };
    } else {
      // Same role - append to current message
      if (role === "user") {
        const additionalText = extractTextFromEvent(normalizedEvent);
        if (additionalText) {
          currentMessage.text = currentMessage.text
            ? currentMessage.text + "\n" + additionalText
            : additionalText;
        }
      }
      // Update timestamp
      currentMessage.time = new Date(normalizedEvent.timestamp).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      });
    }

    // Process event through unified processor
    currentProcessor.processEvent(normalizedEvent);
    lastAuthorRole = role;
  }

  // Finalize last message
  if (currentMessage) {
    currentMessage.widgets = currentProcessor.getWidgets();
    currentMessage.metadata.contentOrder = currentProcessor.getContentOrder();
    
    // For agent messages, text comes from widgets
    if (currentMessage.role === "agent") {
      const textWidgets = currentMessage.widgets.filter(w => w.type === "text");
      currentMessage.text = textWidgets.map(w => w.content || "").join("\n");
    }
    
    messages.push(currentMessage);
  }

  // Get cursor from last processor state
  const cursor = currentProcessor.getCursor();

  return {
    messages,
    lastEventId: cursor.lastEventId,
  };
}

/**
 * Extract text content from an A2A event
 */
function extractTextFromEvent(event: A2AEvent): string {
  let text = "";
  if (event.artifact?.parts) {
    for (const part of event.artifact.parts) {
      if ((part.kind === "text" || part.type === "text") && part.text) {
        text += part.text;
      }
    }
  }
  return text;
}

/**
 * Process events starting after a cursor (for incremental updates)
 * 
 * @param events - Full array of events
 * @param afterEventId - Only process events after this ID
 * @returns Processed messages for events after the cursor
 */
export function processEventsAfterCursor(
  events: SessionEvent[],
  afterEventId: string | null
): {
  messages: ProcessedMessage[];
  lastEventId: string | null;
} {
  if (!afterEventId) {
    return processSessionEvents(events);
  }

  // Find cursor position
  const cursorIndex = events.findIndex(e => e.id === afterEventId);
  if (cursorIndex === -1) {
    // Cursor not found - process all events
    return processSessionEvents(events);
  }

  // Process only events after cursor
  const newEvents = events.slice(cursorIndex + 1);
  return processSessionEvents(newEvents);
}
