import { useStore } from "../store/useStore";
import type {
  Message,
  Widget,
  AGUIStreamData,
  AGUIPart,
  AGUIPartMetadata,
} from "../types";
import { getBaseUrl } from "./api-utils";
import { handleError } from "./error-handler";
import { STREAM } from "./constants";

export class StreamParser {
  private sessionId: string;
  private messageId: string;
  private abortController: AbortController;
  private parseErrors: Error[] = [];

  constructor(sessionId: string, messageId: string) {
    this.sessionId = sessionId;
    this.messageId = messageId;
    this.abortController = new AbortController();
    this.parseErrors = [];
  }

  public abort() {
    this.abortController.abort();
  }

  /**
   * Get accumulated parse errors (useful for debugging/telemetry)
   */
  public getParseErrors(): Error[] {
    return [...this.parseErrors];
  }

  public async stream(url: string, requestBody: unknown) {
    const { updateMessage, setIsGenerating } = useStore.getState();

    // Update base URL if needed (handle relative URLs)
    const baseUrl = getBaseUrl();
    const fullUrl = url.startsWith("http") ? url : `${baseUrl}${url}`;

    try {
      const response = await fetch(fullUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody),
        signal: this.abortController.signal,
      });

      if (!response.ok) {
        const errorText = await response.text().catch(() => "Unknown error");
        throw new Error(
          `HTTP ${response.status}: ${errorText.substring(0, 100)}`,
        );
      }
      if (!response.body) throw new Error("No response body");

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
            } catch (parseError) {
              // Track parse errors for debugging/telemetry
              const error =
                parseError instanceof Error
                  ? parseError
                  : new Error(String(parseError));
              this.parseErrors.push(error);

              // Log in development
              if (import.meta.env.DEV) {
                console.error("Error parsing SSE data:", parseError);
              }

              // Surface to user if too many errors (indicates stream corruption)
              if (this.parseErrors.length >= STREAM.MAX_PARSE_ERRORS) {
                handleError(
                  new Error(
                    `Stream parsing failed ${this.parseErrors.length} times. Data may be incomplete.`,
                  ),
                  "Stream error",
                );
                // Reset counter to avoid spamming
                this.parseErrors = [];
              }
            }
          }
        }
      }

      // Stream completed - finalize any active thinking widgets
      // This handles cases where backend doesn't send explicit thinking_complete events
      const { sessions } = useStore.getState();
      const session = sessions[this.sessionId];
      const message = session?.messages.find((m) => m.id === this.messageId);

      if (message) {
        const hasActiveThinking = message.widgets.some(
          (w) => w.type === "thinking" && w.status === "active",
        );

        if (hasActiveThinking) {
          // Mark all active thinking widgets as completed when stream ends
          const updatedWidgets = message.widgets.map((w) =>
            w.type === "thinking" && w.status === "active"
              ? { ...w, status: "completed" as const }
              : w,
          );
          updateMessage(this.sessionId, this.messageId, {
            widgets: updatedWidgets,
          });
        }
      }

      // Check if we're waiting for user input (HITL)
      // Don't set isGenerating=false if there's a pending approval widget
      const hasPendingApproval = message?.widgets.some(
        (w) => w.type === "approval" && w.status === "pending",
      );

      if (!hasPendingApproval) {
        setIsGenerating(false);
      }
    } catch (error: unknown) {
      // On error, always stop generating
      setIsGenerating(false);
      if (error instanceof Error && error.name === "AbortError") {
        // Stream was cancelled by user - update message state
        updateMessage(this.sessionId, this.messageId, { cancelled: true });
      } else {
        // Real error occurred - use error handler (will display via ErrorDisplay)
        handleError(error, "Stream error");
      }
    }
  }

  private handleData(data: unknown) {
    const { setSessionTaskId, sessions } = useStore.getState();
    const session = sessions[this.sessionId];
    if (!session) return;

    const message = session.messages.find((m) => m.id === this.messageId);
    if (!message) return;

    // Parse the stream data using proper protocol types
    const streamData = data as AGUIStreamData;
    const resultObj = streamData.result || streamData;

    // Handle Task Status Updates
    if (resultObj.statusUpdate) {
      const statusUpdate = resultObj.statusUpdate;

      // Update Task ID if present
      if (statusUpdate.taskId) {
        setSessionTaskId(this.sessionId, statusUpdate.taskId);
      }

      // Check for INPUT_REQUIRED state
      if (statusUpdate.status?.state === "TASK_STATE_INPUT_REQUIRED") {
        // We might want to trigger some UI state here, but the approval widget
        // usually comes in the message update, so we just ensure taskId is set for resuming
      }

      // Check if statusUpdate contains a message update
      if (statusUpdate.status?.update) {
        this.processMessageUpdate(message, statusUpdate.status.update);
      }
    }
    // Handle Direct Message Updates
    else if (resultObj.message) {
      if (resultObj.message.taskId) {
        setSessionTaskId(this.sessionId, resultObj.message.taskId);
      }
      this.processMessageUpdate(message, resultObj.message);
    }
    // Handle Parts directly
    else if (resultObj.parts) {
      this.processMessageUpdate(message, resultObj);
    }
  }

  private processMessageUpdate(
    currentMessage: Message,
    update: { parts?: AGUIPart[]; [key: string]: unknown },
  ) {
    const { updateMessage } = useStore.getState();
    const parts: AGUIPart[] = update.parts || [];

    // Process parts sequentially to maintain correct order
    // This is the proper way to handle content ordering
    const widgetMap = new Map<string, Widget>();
    const contentOrder: string[] = currentMessage.metadata?.contentOrder
      ? [...currentMessage.metadata.contentOrder]
      : [];

    // Initialize with existing widgets
    currentMessage.widgets.forEach((w) => {
      widgetMap.set(w.id, w);
    });

    // Track accumulated text for completion detection
    let accumulatedText = currentMessage.text;

    // Track completed thinking blocks explicitly to prevent race conditions
    // This prevents marking thinking as completed while chunks are still arriving out-of-order
    const completedThinkingBlocks = new Set<string>();

    // Process parts in order - this is critical for correct ordering
    parts.forEach((part: AGUIPart, partIndex: number) => {
      // Check if this is a widget part
      const metadata: AGUIPartMetadata = part.metadata || {};
      const isThinking =
        metadata.event_type === "thinking" ||
        metadata.block_type === "thinking";
      const isToolCall =
        metadata.event_type === "tool_call" && !("is_error" in metadata);
      const isApproval = part.data?.data?.interaction_type === "tool_approval";
      const isToolResult =
        metadata.event_type === "tool_call" && "is_error" in metadata;
      const isThinkingComplete =
        metadata.event_type === "thinking_complete" ||
        metadata.block_type === "thinking_complete";

      // Handle explicit thinking_complete events first (prevents race conditions)
      if (isThinkingComplete) {
        const thinkingId = metadata.block_id || part.data?.data?.thinking_id;
        if (thinkingId) {
          completedThinkingBlocks.add(thinkingId);
          const existing = widgetMap.get(thinkingId);
          if (existing && existing.type === "thinking") {
            widgetMap.set(thinkingId, {
              ...existing,
              status: "completed",
            });
          }
        }
      }

      if (isThinking || isToolCall || isApproval) {
        // Widget appears - track its order
        if (isThinking) {
          const thinkingId = metadata.block_id || `thinking-${partIndex}`;
          const isCompleted = completedThinkingBlocks.has(thinkingId);

          if (!widgetMap.has(thinkingId)) {
            const thinkingType = (metadata.thinking_type || "default") as
              | "todo"
              | "goal"
              | "reflection"
              | "default";
            widgetMap.set(thinkingId, {
              id: thinkingId,
              type: "thinking" as const,
              data: { type: thinkingType },
              status: isCompleted
                ? ("completed" as const)
                : ("active" as const),
              content: part.text || part.data?.data?.text || "",
              isExpanded: false,
            });
            // Add to content order if not already there
            if (!contentOrder.includes(thinkingId)) {
              contentOrder.push(thinkingId);
            }
          } else {
            // Update existing thinking block
            const existing = widgetMap.get(thinkingId);
            if (existing && existing.type === "thinking") {
              // Only mark as completed if explicitly completed, otherwise preserve state
              const newStatus: "completed" | "active" = isCompleted
                ? "completed"
                : existing.status === "completed"
                  ? "completed"
                  : "active";
              widgetMap.set(thinkingId, {
                ...existing,
                content:
                  (existing.content || "") +
                  (part.text || part.data?.data?.text || ""),
                status: newStatus,
              });
            }
          }
        } else if (isToolCall) {
          // Use tool_call_id as primary identifier (most reliable)
          // IMPORTANT: Different tool_call_id = different tool call (e.g., retry after failure)
          // Only prevent duplicates if the SAME tool_call_id appears multiple times
          const toolCallId = metadata.tool_call_id || part.data?.data?.id;
          const toolName =
            metadata.tool_name || part.data?.data?.name || "unknown";
          const toolArgs = part.data?.data?.arguments || {};

          // Generate stable toolId - prefer tool_call_id, otherwise create deterministic ID
          const toolId =
            toolCallId ||
            `tool-${toolName}-${JSON.stringify(toolArgs).slice(0, 50)}-${partIndex}`;

          // Only prevent duplicates if the SAME tool_call_id already exists
          // This allows legitimate retries (different tool_call_id, same tool name) to show up
          const isDuplicate = toolCallId && widgetMap.has(toolCallId);

          if (!isDuplicate && !widgetMap.has(toolId)) {
            widgetMap.set(toolId, {
              id: toolId,
              type: "tool" as const,
              data: {
                name: toolName,
                args: toolArgs,
              },
              status: "working" as const,
              content: "",
              isExpanded: false,
            });
            if (!contentOrder.includes(toolId)) {
              contentOrder.push(toolId);
            }
          }
        } else if (isApproval) {
          const approvalId =
            part.data?.data?.approval_id || `approval-${partIndex}`;
          if (!widgetMap.has(approvalId)) {
            widgetMap.set(approvalId, {
              id: approvalId,
              type: "approval" as const,
              data: {
                toolName: part.data?.data?.tool_name || "unknown",
                toolInput: part.data?.data?.tool_input || {},
                options: part.data?.data?.options,
              },
              status: "pending" as const,
              isExpanded: true,
            });
            if (!contentOrder.includes(approvalId)) {
              contentOrder.push(approvalId);
            }
          }
        }
      } else if (isToolResult) {
        // Tool result - update existing tool widget
        // Support incremental content updates for streaming tools (e.g., agent_call, execute_command)
        const toolId = metadata.tool_call_id || part.data?.data?.tool_call_id;
        if (toolId && widgetMap.has(toolId)) {
          const existing = widgetMap.get(toolId);
          if (!existing) return;

          const newContent = part.data?.data?.content || "";
          const isDenied =
            metadata.is_error &&
            (newContent.includes("TOOL_EXECUTION_DENIED") ||
              newContent.includes("user denied"));

          // Determine if this is incremental content (append) or final result (replace)
          // For streaming tools (e.g., agent_call, execute_command), content may arrive incrementally
          // We append if: tool is working AND we have existing content AND new content doesn't contain the existing content
          // (meaning it's a continuation, not a replacement)
          const existingContent = existing.content || "";
          const isIncremental =
            existing.status === "working" &&
            existingContent.length > 0 &&
            newContent.length > 0 &&
            !newContent.includes(existingContent) && // New content doesn't contain old (it's incremental)
            newContent.length < existingContent.length * 2; // New chunk is reasonable size (not a full replacement)

          const updatedContent = isIncremental
            ? existingContent + newContent
            : newContent || existingContent;

          // Tool result events indicate completion - determine final status
          // IMPORTANT: tool_result events mean the tool has completed (or errored)
          // Only keep as 'working' if we're certain this is incremental streaming (content being appended)
          // Otherwise, tool_result without error means success
          const toolStatus: "working" | "success" | "failed" = isDenied
            ? "failed"
            : metadata.is_error
              ? "failed"
              : isIncremental
                ? "working" // Keep working only for confirmed incremental streaming updates
                : "success"; // Tool result without error means completion -> success

          if (existing.type === "tool") {
            widgetMap.set(toolId, {
              ...existing,
              status: toolStatus,
              content: updatedContent,
            });
          }

          // Update related approval widget when tool completes (approved or denied)
          if (toolStatus === "success" || isDenied) {
            // Get tool name from the existing tool widget for matching approval widgets
            const existingToolName =
              existing.type === "tool" ? existing.data.name : undefined;
            if (existingToolName) {
              widgetMap.forEach((widget, widgetId) => {
                if (
                  widget.type === "approval" &&
                  widget.data?.toolName === existingToolName &&
                  widget.status === "pending"
                ) {
                  widgetMap.set(widgetId, {
                    ...widget,
                    status: "decided",
                    decision: isDenied ? "deny" : "approve",
                  });
                }
              });
            }
          }
        }
      } else if (part.text && !isThinking && !isToolCall && !isApproval) {
        // Regular text part - accumulate it
        let textContent = part.text || "";

        // Sanitize control characters and escape sequences
        // Remove control characters (except newlines and tabs)
        textContent = textContent.replace(
          /[\x00-\x08\x0B-\x0C\x0E-\x1F\x7F]/g,
          "",
        );
        // Remove common escape sequences that leak through
        textContent = textContent.replace(/<ctrl\d+>/gi, "");
        textContent = textContent.replace(/&lt;ctrl\d+&gt;/gi, "");
        // Remove ANSI escape sequences
        textContent = textContent.replace(/\x1B\[[0-9;]*[a-zA-Z]/g, "");

        // Filter out tool call syntax that leaks into text
        const toolCallPatterns = [
          /call:\w+\{/i, // call:TOOL_NAME{
          /tool:\s*\w+\s*\{/i, // tool: NAME {
          /^\s*call:\s*\w+/i, // call:TOOL_NAME at start
        ];

        // Filter out verbose backend messages that are redundant with widgets
        const approvalPatterns = [
          /^✅\s*Approved:/,
          /^🚫\s*Denied:/,
          /SUCCESS:\s*Approved:/i,
          /DENIED:\s*Denied:/i,
          /Command Execution Request/i,
          /This will execute on the server/i,
          /🔐 Tool Approval Required/i,
          /Please respond with:\s*approve or deny/i,
          /Tool:\s*\w+\s*Input:/i,
        ];

        // Remove tool call syntax from text content (but preserve spacing)
        toolCallPatterns.forEach((pattern) => {
          textContent = textContent.replace(pattern, "");
        });

        // Check if entire text should be filtered
        const shouldFilter =
          approvalPatterns.some((pattern) => textContent.match(pattern)) ||
          toolCallPatterns.some(
            (pattern) => part.text?.match(pattern), // Check original text for tool call patterns
          );

        // Only skip truly empty content (not whitespace - whitespace is important for word separation!)
        if (textContent.length === 0 || shouldFilter) {
          return;
        }

        if (textContent) {
          // Trust provider's spacing - append text as-is
          // All LLM providers (OpenAI, Anthropic, Gemini, Ollama) include proper spacing in their streaming chunks
          accumulatedText += textContent;

          // Find the appropriate text widget to append to
          // Strategy: Find the last non-text widget, then look for a text widget immediately after it
          // If no text widget exists at that position, create one
          const lastNonTextWidgetId = contentOrder
            .filter((id) => {
              const widget = widgetMap.get(id);
              return widget && widget.type !== "text";
            })
            .pop();

          // Text widgets use synthetic IDs to track position relative to other widgets:
          // - $$text_marker$$_start: Text before any widgets (initial text content)
          // - $$text_marker$$_after_{widgetId}: Text after a specific widget (interleaved content)
          // Using $$ delimiters prevents collision with widget IDs that might contain __
          // This marker-based approach allows proper text accumulation across streaming updates
          // and handles interleaved text/widget content correctly
          const TEXT_MARKER_PREFIX = "$$text_marker$$";
          const textMarkerId = lastNonTextWidgetId
            ? `${TEXT_MARKER_PREFIX}_after_${lastNonTextWidgetId}`
            : `${TEXT_MARKER_PREFIX}_start`;

          // Find existing text widget at this position (handles text chunks arriving in separate updates)
          let targetTextWidgetId = textMarkerId;
          if (lastNonTextWidgetId) {
            // Look backwards from end of contentOrder to find the most recent text widget after lastNonTextWidgetId
            const lastNonTextIndex = contentOrder.indexOf(lastNonTextWidgetId);
            for (let i = contentOrder.length - 1; i > lastNonTextIndex; i--) {
              const widget = widgetMap.get(contentOrder[i]);
              if (widget?.type === "text") {
                targetTextWidgetId = widget.id;
                break;
              }
            }
          } else {
            // Look for $$text_marker$$_start widget
            const TEXT_MARKER_PREFIX = "$$text_marker$$";
            const startTextWidget = contentOrder.find((id) => {
              const widget = widgetMap.get(id);
              return (
                widget?.type === "text" && id === `${TEXT_MARKER_PREFIX}_start`
              );
            });
            if (startTextWidget) {
              targetTextWidgetId = startTextWidget;
            }
          }

          if (!widgetMap.has(targetTextWidgetId)) {
            // Create a new text widget at this position
            widgetMap.set(targetTextWidgetId, {
              id: targetTextWidgetId,
              type: "text" as const,
              data: {},
              status: "active",
              content: textContent,
              isExpanded: true,
            });
            if (!contentOrder.includes(targetTextWidgetId)) {
              contentOrder.push(targetTextWidgetId);
            }
          } else {
            // Append to existing text widget
            // Trust provider's spacing - append text as-is
            const existing = widgetMap.get(targetTextWidgetId);
            if (existing) {
              widgetMap.set(targetTextWidgetId, {
                ...existing,
                content: (existing.content || "") + textContent,
              });
            }
          }
        }
      }
    });

    // Mark completed thinking blocks when text appears or tool calls start (fallback for implicit completion)
    // Only mark blocks that haven't been explicitly completed via thinking_complete event
    const hasNewText = accumulatedText.length > currentMessage.text.length;
    const hasNewToolCalls = parts.some(
      (p) => p.metadata?.event_type === "tool_call" && !p.metadata?.is_error,
    );

    if (hasNewText || hasNewToolCalls) {
      widgetMap.forEach((widget, id) => {
        if (
          widget.type === "thinking" &&
          widget.status === "active" &&
          !completedThinkingBlocks.has(id)
        ) {
          completedThinkingBlocks.add(id);
          widgetMap.set(id, { ...widget, status: "completed" });
        }
      });
    }

    // Build final widget array in contentOrder sequence
    const orderedWidgets: Widget[] = [];
    const seenWidgetIds = new Set<string>();

    // First, add widgets in contentOrder (maintains stream order)
    contentOrder.forEach((widgetId) => {
      const widget = widgetMap.get(widgetId);
      if (widget) {
        orderedWidgets.push(widget);
        seenWidgetIds.add(widgetId);
      }
    });

    // Then add any widgets not in contentOrder (shouldn't happen, but safety fallback)
    widgetMap.forEach((widget, id) => {
      if (!seenWidgetIds.has(id)) {
        orderedWidgets.push(widget);
      }
    });

    const newWidgets = orderedWidgets;

    updateMessage(this.sessionId, this.messageId, {
      text: accumulatedText,
      widgets: newWidgets,
      metadata: {
        ...currentMessage.metadata,
        contentOrder: contentOrder.length > 0 ? contentOrder : undefined,
      },
    });
  }
}
