import { useStore } from "../store/useStore";
import type {
  Widget,
  ToolWidget,
  ThinkingWidget,
  ApprovalWidget,
  TextWidget,
  ToolWidgetStatus,
} from "../types";
import { handleError } from "./error-handler";

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
  private abortController: AbortController;

  // Track created widgets to avoid duplicates
  private createdToolWidgets = new Set<string>();
  private createdThinkingWidgets = new Set<string>();

  constructor(sessionId: string, messageId: string) {
    this.sessionId = sessionId;
    this.messageId = messageId;
    this.abortController = new AbortController();
  }

  public abort() {
    this.abortController.abort();
  }

  public async stream(url: string, requestBody: unknown) {
    const { updateMessage, setIsGenerating } = useStore.getState();

    try {
      const response = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
        signal: this.abortController.signal,
      });

      if (!response.ok) {
        const errorText = await response.text().catch(() => "Unknown error");
        throw new Error(`HTTP ${response.status}: ${errorText.substring(0, 200)}`);
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
      setIsGenerating(false);
    } catch (error: unknown) {
      setIsGenerating(false);
      if (error instanceof Error && error.name === "AbortError") {
        updateMessage(this.sessionId, this.messageId, { cancelled: true });
      } else {
        handleError(error, "Stream error");
      }
    }
  }

  private handleData(data: unknown) {
    const { setSessionTaskId } = useStore.getState();
    const result = (data as { result?: A2AResult })?.result || (data as A2AResult);

    if (result.taskId) {
      setSessionTaskId(this.sessionId, result.taskId);
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
    const { updateMessage, sessions } = useStore.getState();
    const session = sessions[this.sessionId];
    const message = session?.messages.find((m) => m.id === this.messageId);
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
        if (part.kind === "text" && part.text) {
          // Text content - pass isPartial to handle delta vs complete correctly
          accumulatedText = this.processTextPart(part.text, accumulatedText, widgetMap, contentOrder, isPartial);
        } else if (part.kind === "data" && part.data) {
          const data = part.data as Record<string, unknown>;

          if (data.type === "thinking") {
            // Thinking Part (legacy pattern: thinking content as Part)
            const id = data.id as string;
            const content = data.content as string;
            const status = data.status as string;
            const isCompleted = status === "completed";
            this.processThinking(id, content || "", isCompleted, "default", widgetMap, contentOrder);
          } else if (data.type === "tool_use") {
            // Tool call Part
            const toolId = data.id as string;
            if (!this.createdToolWidgets.has(toolId)) {
              this.processToolCallFromPart(data, widgetMap, contentOrder);
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

    updateMessage(this.sessionId, this.messageId, {
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
    isPartial: boolean
  ): string {
    if (!text) return accumulatedText;

    // On final (non-partial) events, apply ADK-Go duplicate detection
    // This prevents duplication when final event sends complete text after streaming
    if (!isPartial) {
      // If accumulated already contains this text, it's a duplicate - skip
      if (accumulatedText === text || accumulatedText.endsWith(text)) {
        return accumulatedText;
      }
      // If the new text contains the accumulated text, it's the complete version
      // Skip processing since we already have partial content displayed
      if (text.includes(accumulatedText) && accumulatedText.length > 0) {
        return accumulatedText;
      }
    }

    const newAccumulatedText = isPartial
      ? accumulatedText + text  // Streaming: append delta
      : text;  // Final: use as-is (shouldn't reach here if duplicate detection worked)

    // Find last non-text widget
    const lastNonTextWidgetId = contentOrder
      .filter((id) => {
        const widget = widgetMap.get(id);
        return widget && widget.type !== "text";
      })
      .pop();

    // Determine text widget ID based on position
    const textMarkerId = lastNonTextWidgetId
      ? `${TEXT_MARKER_PREFIX}_after_${lastNonTextWidgetId}`
      : `${TEXT_MARKER_PREFIX}_start`;

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
        return widget?.type === "text" && id === `${TEXT_MARKER_PREFIX}_start`;
      });
      if (startTextWidget) {
        targetTextWidgetId = startTextWidget;
      }
    }

    // Create or update text widget
    if (!widgetMap.has(targetTextWidgetId)) {
      const textWidget: TextWidget = {
        id: targetTextWidgetId,
        type: "text",
        status: isPartial ? "active" : "completed",
        content: text,
        data: {},
        isExpanded: true,
      };
      widgetMap.set(targetTextWidgetId, textWidget);
      if (!contentOrder.includes(targetTextWidgetId)) {
        contentOrder.push(targetTextWidgetId);
      }
    } else {
      const existing = widgetMap.get(targetTextWidgetId);
      if (existing && existing.type === "text") {
        // On partial: append delta
        // On final: this shouldn't happen (duplicate detection returns early)
        const newContent = isPartial
          ? (existing.content || "") + text
          : text;
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
  private processToolCallFromPart(
    data: Record<string, unknown>,
    widgetMap: Map<string, Widget>,
    contentOrder: string[]
  ) {
    const id = data.id as string;
    const widgetId = `tool_${id}`;

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
    const widgetId = `tool_${tr.tool_call_id}`;
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
  private processThinking(
    id: string,
    content: string,
    isCompleted: boolean,
    type: string | undefined,
    widgetMap: Map<string, Widget>,
    contentOrder: string[]
  ) {
    const widgetId = `thinking_${id}`;

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
      data: { type: (type || "default") as "todo" | "goal" | "reflection" | "default" },
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
    if (state !== "input-required" && state !== "input_required") return;

    const { updateMessage, sessions } = useStore.getState();
    const session = sessions[this.sessionId];
    const message = session?.messages.find((m) => m.id === this.messageId);
    if (!message) return;

    const widgetMap = new Map<string, Widget>();
    const contentOrder: string[] = message.metadata?.contentOrder
      ? [...message.metadata.contentOrder]
      : [];

    message.widgets.forEach((w) => {
      widgetMap.set(w.id, w);
    });

    const taskId = result.taskId || session?.taskId;
    const toolCallIDs = result.metadata?.long_running_tool_ids || [];
    const inputPrompt = result.metadata?.input_prompt || "Human input required.";
    const widgetId = `approval_${taskId || this.messageId}`;

    if (widgetMap.has(widgetId)) return;

    // Extract tool name and input from existing tool widgets
    // Find the first tool widget that matches one of the long-running tool IDs
    let toolName = "Unknown Tool";
    let toolInput: Record<string, unknown> = {};
    
    if (toolCallIDs.length > 0) {
      // Find tool widget by matching tool_call_id
      for (const toolCallID of toolCallIDs) {
        const toolWidgetId = `tool_${toolCallID}`;
        const toolWidget = widgetMap.get(toolWidgetId);
        if (toolWidget && toolWidget.type === "tool") {
          toolName = toolWidget.data.name || "Unknown Tool";
          toolInput = toolWidget.data.args || {};
          break; // Use first matching tool
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
      const widget = widgetMap.get(id);
      if (widget) orderedWidgets.push(widget);
    });
    widgetMap.forEach((widget, id) => {
      if (!contentOrder.includes(id)) orderedWidgets.push(widget);
    });

    updateMessage(this.sessionId, this.messageId, {
      widgets: orderedWidgets,
      metadata: {
        ...message.metadata,
        contentOrder: contentOrder.length > 0 ? contentOrder : undefined,
      },
    });
  }

  /**
   * Finalize stream.
   */
  private finalizeStream() {
    const { updateMessage, sessions } = useStore.getState();
    const session = sessions[this.sessionId];
    const message = session?.messages.find((m) => m.id === this.messageId);
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
      updateMessage(this.sessionId, this.messageId, { widgets: updatedWidgets });
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
  status?: { state: string };
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
