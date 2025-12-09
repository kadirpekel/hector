import type {
  Widget,
  ToolWidget,
  ThinkingWidget,
  ApprovalWidget,
  TextWidget,
  ToolWidgetStatus,
} from "../types";
import { handleError } from "./error-handler";
import type { StreamDispatcher } from "./stream-utils";

type Dispatcher = StreamDispatcher;

/**
 * StreamParser - V2 A2A-native stream parser following legacy patterns.
 *
 * V2 Backend Event Flow (aligned with legacy):
 *
 * STREAMING (partial=true):
 * - artifact.parts: [DataPart(thinking), TextPart(chunk), DataPart(tool_use)]
 *   ^ ALL content types stream as Parts in order!
 * - metadata.partial: true
 * - metadata.thinking: {id, content, status}   (also in metadata for backwards compat)
 * - metadata.tool_calls: [{id, name}]          (also in metadata for backwards compat)
 *
 * FINAL (partial=false):
 * - artifact.parts: [DataPart(thinking), TextPart(full), DataPart(tool_use)]
 * - metadata.partial: false
 * - metadata.thinking: completed
 *
 * Key Pattern (from legacy):
 * - ALL contextual content (thinking, text, tool calls) streams as Parts
 * - Parts arrive IN ORDER as the LLM generates them
 * - Widgets are created/updated in the order Parts arrive
 * - Metadata is supplementary, Parts are primary
 */

const TEXT_MARKER_PREFIX = "$$text_marker$$";

export class StreamParser {
  private sessionId: string;
  private messageId: string;
  private currentController: AbortController | null = null;
  private dispatch: Dispatcher;

  // Track created widgets to avoid duplicates
  private createdToolWidgets = new Set<string>();
  private createdThinkingWidgets = new Set<string>();

  constructor(
    sessionId: string,
    messageId: string,
    dispatch: Dispatcher
  ) {
    this.sessionId = sessionId;
    this.messageId = messageId;
    this.dispatch = dispatch;
  }

  abort() {
    if (this.currentController) {
      this.currentController.abort();
    }
  }

  cleanup() {
    this.createdToolWidgets.clear();
    this.createdThinkingWidgets.clear();
    this.abort();
  }

  public async stream(url: string, requestBody: unknown) {
    this.currentController = new AbortController();

    try {
      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
        signal: this.currentController.signal,
      });

      if (!response.ok) {
        const errorText = await response.text().catch(() => "Unknown error");
        throw new Error("HTTP " + response.status + ": " + errorText.substring(0, 200));
      }

      if (!response.body) {
        throw new Error("No response body");
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            try {
              const data = JSON.parse(line.substring(6));
              this.handleData(data);
            } catch {
              // Ignore parse errors
            }
          }
        }
      }

      this.finalizeStream();
      this.dispatch.setIsGenerating(false);
    } catch (error: unknown) {
      this.dispatch.setIsGenerating(false);
      if (error instanceof Error && error.name === "AbortError") {
        this.dispatch.updateMessage(this.sessionId, this.messageId, { cancelled: true });
      } else {
        handleError(error, "Stream error");
      }
    } finally {
      this.currentController = null;
      this.dispatch.setActiveAgentId(null);
    }
  }

  private handleData(data: unknown) {
    // Dispatch setSessionTaskId action if present (assumes added to Dispatcher if needed,
    // but for now we can skip it or add it to dispatcher types.
    // For now, let's assume we handle taskId via metadata update if crucial, or ignore if minor.
    // Actually, setSessionTaskId helps with context. Let's add it to Dispatcher type later if needed.
    // For crash fix, let's focus on message updates.

    const result = (data as { result?: A2AResult })?.result || (data as A2AResult);

    if (result.taskId) {
      // Reduced scope for crash fix: taskId update not critical path for rendering
      this.dispatch.setSessionTaskId(this.sessionId, result.taskId);
    }

    switch (result.kind) {
      case "status-update":
        this.handleStatusUpdate(result);
        break;
      case "artifact-update":
        this.processArtifactUpdate(result);
        break;
      case "task":
        if (result.artifacts) {
          for (const artifact of result.artifacts) {
            this.processArtifactUpdate({ ...result, artifact });
          }
        }
        break;
      default:
        if (result.artifact) {
          this.processArtifactUpdate(result);
        }
    }
  }

  /**
   * Process artifact update - V2 specific logic.
   *
   * Order of processing matters:
   * 1. Metadata (tool_calls, thinking, tool_results) - creates widgets during streaming
   * 2. Artifact parts (text, tool_use DataParts) - only text during streaming
   * 3. Dedupe: Don't create widget if already exists from metadata
   */
  private processArtifactUpdate(result: A2AResult) {
    // We can't access store state directly anymore.
    // We assume the caller provides the CURRENT message state or we operate on accumulation?
    // "Legacy Fluent Pattern" relies on reading current message state to accum.
    // We need 'message' passed in? Or we rely on 'accumulatedText' state in parser?
    // To support re-hydration, legacy parser read from store.
    // The safest way without store import is to have 'message' passed into 'stream' or 'handleData'?
    // OR: we fetch message from store via a getter in dispatcher?

    // Let's use a getter pattern in dispatcher to fetch fresh state without importing store.
    // This solves the read dependency.
    const message = this.dispatch.getMessage(this.sessionId, this.messageId);
    if (!message) return;

    const isPartial = result.metadata?.partial === true;

    // Initialize from existing state (batch update pattern)
    const widgetMap = new Map<string, Widget>();
    const contentOrder: string[] = message.metadata?.contentOrder
      ? [...message.metadata.contentOrder]
      : [];

    message.widgets.forEach((w) => {
      widgetMap.set(w.id, w);
    });

    let accumulatedText = message.text || "";

    // ==== STEP 1: Process METADATA (tool results only - they update existing widgets) ====
    // NOTE: Thinking and tool_calls from metadata are FALLBACK only.
    // Primary source is artifact.parts (Parts stream in order!)

    if (result.metadata?.tool_results) {
      for (const tr of result.metadata.tool_results) {
        this.processToolResult(tr, widgetMap);
      }
    }

    // ==== STEP 2: Process artifact.parts IN ORDER (thinking, text, tool_use) ====
    // Parts stream in the order they were generated by the LLM
    // This is critical for correct widget ordering

    if (result.artifact?.parts) {
      for (const part of result.artifact.parts) {
        // Flatten/extract author from result metadata
        const author = (result.metadata?.author as string) || (result.metadata?.["event_author"] as string);
        if (author) {
          this.dispatch.setActiveAgentId(author);
        }

        if (part.kind === "text" && part.text) {
          // Text content - pass isPartial to handle delta vs complete correctly
          accumulatedText = this.processTextPart(part.text, accumulatedText, widgetMap, contentOrder, isPartial, author);
        } else if (part.kind === "data" && part.data) {
          const data = part.data as Record<string, unknown>;

          if (data.type === "thinking") {
            // Thinking Part (legacy pattern: thinking content as Part)
            const id = data.id as string;
            const content = data.content as string;
            const status = data.status as string;
            const isCompleted = status === "completed";
            this.processThinking(id, content || "", isCompleted, "default", widgetMap, contentOrder, author);
          } else if (data.type === "tool_use") {
            // Tool call Part
            const toolId = data.id as string;
            if (!this.createdToolWidgets.has(toolId)) {
              this.processToolCallFromPart(data, widgetMap, contentOrder, author);
            }
          } else if (data.type === "tool_result") {
            // Tool result Part
            const toolCallId = data.tool_call_id as string;
            this.processToolResult({
              tool_call_id: toolCallId,
              content: data.content as string,
              is_error: data.is_error as boolean,
              status: data.status as string,
            }, widgetMap);
          }
        }
      }
    }

    // ==== STEP 3: Mark thinking completed when final event arrives ====
    if (!isPartial) {
      widgetMap.forEach((widget, id) => {
        if (widget.type === "thinking" && widget.status === "active") {
          widgetMap.set(id, { ...widget, status: "completed" });
        }
        // Also mark text widgets as completed
        if (widget.type === "text" && widget.status === "active") {
          widgetMap.set(id, { ...widget, status: "completed" });
        }
      });
    }

    // ==== STEP 4: Build ordered widgets and update ====
    const orderedWidgets: Widget[] = [];
    const seenWidgetIds = new Set<string>();

    contentOrder.forEach((widgetId) => {
      const widget = widgetMap.get(widgetId);
      if (widget) {
        orderedWidgets.push(widget);
        seenWidgetIds.add(widgetId);
      }
    });

    widgetMap.forEach((widget, id) => {
      if (!seenWidgetIds.has(id)) {
        orderedWidgets.push(widget);
      }
    });

    this.dispatch.updateMessage(this.sessionId, this.messageId, {
      text: accumulatedText,
      widgets: orderedWidgets,
      metadata: {
        ...message.metadata,
        contentOrder: contentOrder.length > 0 ? contentOrder : undefined,
      },
    });
  }

  /**
   * Process text part - compute position fresh from contentOrder.
   * @param isPartial - true for streaming deltas, false for final complete text
   */
  private processTextPart(
    text: string,
    accumulatedText: string,
    widgetMap: Map<string, Widget>,
    contentOrder: string[],
    isPartial: boolean,
    author?: string
  ): string {
    if (!text) return accumulatedText;

    // On final (non-partial) events, apply ADK-Go duplicate detection
    // This prevents duplication when final event sends complete text after streaming
    if (!isPartial) {
      // If accumulated already contains this text, it's a duplicate - skip
      if (accumulatedText === text || accumulatedText.endsWith(text)) {
        return accumulatedText;
      }
    }

    const newAccumulatedText = accumulatedText + text; // Always append (Dedup handled above)

    // Find last non-text widget
    const lastNonTextWidgetId = contentOrder
      .filter((id) => {
        const widget = widgetMap.get(id);
        return widget && widget.type !== "text";
      })
      .pop();

    // Determine text widget ID based on position
    const textMarkerId = lastNonTextWidgetId
      ? TEXT_MARKER_PREFIX + "_after_" + lastNonTextWidgetId
      : TEXT_MARKER_PREFIX + "_start";

    // Find existing text widget at this position
    let targetTextWidgetId = textMarkerId;
    if (lastNonTextWidgetId) {
      const lastNonTextIndex = contentOrder.indexOf(lastNonTextWidgetId);
      for (let i = contentOrder.length - 1; i > lastNonTextIndex; i--) {
        const widget = widgetMap.get(contentOrder[i]);
        if (widget?.type === "text") {
          targetTextWidgetId = widget.id;
          break;
        }
      }
    } else {
      const startTextWidget = contentOrder.find((id) => {
        const widget = widgetMap.get(id);
        return widget?.type === "text" && id === TEXT_MARKER_PREFIX + "_start";
      });
      if (startTextWidget) {
        targetTextWidgetId = startTextWidget;
      }
    }

    // Create or update text widget
    const existingWidget = widgetMap.get(targetTextWidgetId);

    // Author change detection for sequential agents:
    // Only create new widget if BOTH widgets have authors AND they differ
    // This correctly handles:
    // - undefined → "agent" (first text) → append (don't create new)
    // - "agent1" → "agent2" (sequential) → create new widget
    // - "agent" → undefined (backward compat) → append
    const authorChanged =
      existingWidget?.type === "text" &&
      existingWidget.data?.author &&
      author &&
      existingWidget.data.author !== author;

    if (!widgetMap.has(targetTextWidgetId) || authorChanged) {
      // Create NEW text widget if:
      // 1. No widget exists at this position, OR
      // 2. Author has changed (e.g., sequential agent switching from researcher to summarizer)

      if (authorChanged) {
        // Complete the previous author's widget before creating new one
        widgetMap.set(targetTextWidgetId, {
          ...existingWidget,
          status: "completed" as const,
        });
        // Generate new unique ID for the new author's text
        targetTextWidgetId = TEXT_MARKER_PREFIX + "_" + author + "_" + Date.now();
      }

      const textWidget: TextWidget = {
        id: targetTextWidgetId,
        type: "text",
        status: isPartial ? "active" : "completed",
        content: text,
        data: { author },
        isExpanded: true,
      };
      widgetMap.set(targetTextWidgetId, textWidget);
      if (!contentOrder.includes(targetTextWidgetId)) {
        contentOrder.push(targetTextWidgetId);
      }
    } else {
      const existing = widgetMap.get(targetTextWidgetId);
      if (existing && existing.type === "text") {
        // Append to existing widget (same author, continuing stream)
        // Add a newline for separation if this is a new block (not streaming delta)
        const separator = (isPartial || !existing.content) ? "" : "\n\n";
        const newContent = (existing.content || "") + separator + text;

        widgetMap.set(targetTextWidgetId, {
          ...existing,
          content: newContent,
          status: isPartial ? existing.status : ("completed" as const),
        });
      }
    }

    return newAccumulatedText;
  }

  /**
   * Process tool call from artifact.parts DataPart.
   */
  /**
   * Process tool call from artifact.parts DataPart.
   */
  private processToolCallFromPart(
    data: Record<string, unknown>,
    widgetMap: Map<string, Widget>,
    contentOrder: string[],
    author?: string
  ) {
    const id = data.id as string;
    const widgetId = "tool_" + id;

    if (this.createdToolWidgets.has(id) || widgetMap.has(widgetId)) {
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
        args: (data.arguments || data.input || {}) as Record<string, unknown>,
        author: author
      },
      isExpanded: true,
    };

    widgetMap.set(widgetId, toolWidget);
    if (!contentOrder.includes(widgetId)) {
      contentOrder.push(widgetId);
    }
  }

  /**
   * Process tool result - update existing tool widget.
   */
  private processToolResult(tr: ToolResultMeta, widgetMap: Map<string, Widget>) {
    const widgetId = "tool_" + tr.tool_call_id;
    const existing = widgetMap.get(widgetId);
    if (!existing || existing.type !== "tool") return;

    const newContent =
      typeof tr.content === "string" ? tr.content : JSON.stringify(tr.content);

    const existingContent = existing.content || "";
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

    widgetMap.set(widgetId, {
      ...existing,
      status,
      content: updatedContent,
      isExpanded: status === "working",
    });
  }

  /**
   * Process thinking block.
   */
  /**
   * Process thinking block.
   */
  private processThinking(
    id: string,
    content: string,
    isCompleted: boolean,
    type: string | undefined,
    widgetMap: Map<string, Widget>,
    contentOrder: string[],
    author?: string
  ) {
    const widgetId = "thinking_" + id;

    if (this.createdThinkingWidgets.has(id)) {
      // Update existing widget
      const existing = widgetMap.get(widgetId) as ThinkingWidget | undefined;
      if (existing) {
        // CRITICAL: On completed, REPLACE content (full content arrives)
        // On active (streaming), APPEND content (delta arrives)
        const newContent = isCompleted
          ? content  // Replace - this is the complete content
          : (existing.content || "") + content;  // Append - this is a delta

        widgetMap.set(widgetId, {
          ...existing,
          content: newContent,
          status: isCompleted ? "completed" : existing.status,
          // Collapse when completed
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
      content: content,
      data: {
        type: (type || "default") as "todo" | "goal" | "reflection" | "default",
        author: author
      },
      isExpanded: !isCompleted,  // Expand while active, collapse when done
    };

    widgetMap.set(widgetId, thinkingWidget);
    if (!contentOrder.includes(widgetId)) {
      contentOrder.push(widgetId);
    }
  }

  /**
   * Handle status update for HITL.
   */
  private handleStatusUpdate(result: A2AResult) {
    // A2A spec uses "input-required" (with hyphen) as the state value
    const state = result.status?.state;
    // Get fresh message state via dispatcher
    const message = this.dispatch.getMessage(this.sessionId, this.messageId);
    if (!message) return;

    // Handle failed state
    if (state === "failed") {
      const errorText =
        result.status?.message?.parts?.[0]?.text ||
        "Agent execution failed with unknown error.";

      const alertContent = "\n\n> [!CAUTION]\n> **Agent Run Failed**\n> " + errorText + "\n";

      this.dispatch.updateMessage(this.sessionId, this.messageId, {
        text: (message.text || "") + alertContent
      });
      return;
    }

    if (state !== "input-required" && state !== "input_required") return;

    const widgetMap = new Map<string, Widget>();
    const contentOrder: string[] = message.metadata?.contentOrder
      ? [...message.metadata.contentOrder]
      : [];

    message.widgets.forEach((w) => {
      widgetMap.set(w.id, w);
    });

    const taskId = result.taskId;
    const toolCallIDs = result.metadata?.long_running_tool_ids || [];
    const inputPrompt = result.metadata?.input_prompt || "Human input required.";
    const widgetId = "approval_" + (taskId || this.messageId);

    if (widgetMap.has(widgetId)) return;

    let toolName = "Unknown Tool";
    let toolInput: Record<string, unknown> = {};

    if (toolCallIDs.length > 0) {
      for (const toolCallID of toolCallIDs) {
        const toolWidgetId = "tool_" + toolCallID;
        const toolWidget = widgetMap.get(toolWidgetId);
        if (toolWidget && toolWidget.type === "tool") {
          toolName = toolWidget.data.name || "Unknown Tool";
          toolInput = toolWidget.data.args || {};
          break;
        }
      }
    }

    const approvalWidget: ApprovalWidget = {
      id: widgetId,
      type: "approval",
      status: "pending",
      content: inputPrompt,
      data: {
        toolName: toolName,
        toolInput: toolInput,
        task_id: taskId || undefined,
        tool_call_ids: toolCallIDs,
        prompt: inputPrompt,
      },
      isExpanded: true,
    };

    widgetMap.set(widgetId, approvalWidget);
    if (!contentOrder.includes(widgetId)) {
      contentOrder.push(widgetId);
    }

    const orderedWidgets: Widget[] = [];
    contentOrder.forEach((id) => {
      const w = widgetMap.get(id);
      if (w) orderedWidgets.push(w);
    });

    this.dispatch.updateMessage(this.sessionId, this.messageId, {
      widgets: orderedWidgets,
      metadata: {
        ...message.metadata,
        contentOrder: contentOrder.length > 0 ? contentOrder : undefined
      }
    });

  }

  /**
   * Finalize stream.
   */
  private finalizeStream() {
    const message = this.dispatch.getMessage(this.sessionId, this.messageId);
    if (!message) return;

    const hasActiveWidgets = message.widgets.some(
      (w) => (w.type === "thinking" || w.type === "text") && w.status === "active"
    );

    if (hasActiveWidgets) {
      const updatedWidgets = message.widgets.map((w) =>
        (w.type === "thinking" || w.type === "text") && w.status === "active"
          ? { ...w, status: "completed" as const }
          : w
      );
      this.dispatch.updateMessage(this.sessionId, this.messageId, { widgets: updatedWidgets });
    }
  }
}

// ============================================================================
// A2A Types
// ============================================================================

interface A2APart {
  kind: string;
  text?: string;
  data?: unknown;
}

interface A2AArtifact {
  artifactId: string;
  parts: A2APart[];
}

interface A2AResult {
  kind?: "status-update" | "artifact-update" | "task";
  taskId?: string;
  status?: {
    state: string;
    message?: {
      parts?: { text: string }[];
    };
  };
  artifact?: A2AArtifact;
  artifacts?: A2AArtifact[];
  metadata?: {
    partial?: boolean;
    thinking?: ThinkingMeta;
    tool_calls?: ToolCallMeta[];
    tool_results?: ToolResultMeta[];
    long_running_tool_ids?: string[];
    input_prompt?: string;
    [key: string]: unknown;
  };
}

interface ThinkingMeta {
  id: string;
  status: "active" | "completed";
  content: string;
  type?: string;
}

interface ToolCallMeta {
  id: string;
  name: string;
  args?: Record<string, unknown>;
  status?: string;
}

interface ToolResultMeta {
  tool_call_id: string;
  content: string | Record<string, unknown>;
  status?: string;
  is_error?: boolean;
}
