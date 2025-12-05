import { useState, memo } from "react";
import { Shield, Check, X, ChevronDown, Loader2, Sparkles } from "lucide-react";
import type { ApprovalWidget as ApprovalWidgetType } from "../../types";
import { cn } from "../../lib/utils";
import { useStore } from "../../store/useStore";
import { StreamParser } from "../../lib/stream-parser";
import { handleError } from "../../lib/error-handler";
import { generateShortId } from "../../lib/id-generator";
import { useWidgetExpansion } from "./useWidgetExpansion";
import {
  getWidgetStatusStyles,
  getWidgetContainerClasses,
  getWidgetHeaderClasses,
} from "./widgetStyles";

interface ApprovalWidgetProps {
  widget: ApprovalWidgetType;
  sessionId: string;
  onExpansionChange?: (expanded: boolean) => void;
  shouldAnimate?: boolean;
}

/**
 * ApprovalWidget handles HITL tool approval requests.
 * Memoized to prevent re-renders when parent updates but widget props unchanged.
 */
export const ApprovalWidget = memo<ApprovalWidgetProps>(function ApprovalWidget({
  widget,
  sessionId,
  onExpansionChange,
  shouldAnimate = false,
}) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { toolName, toolInput } = widget.data;
  const { status, decision } = widget;
  
  // Use selectors for better performance - only subscribe to what we need
  const updateMessage = useStore((state) => state.updateMessage);
  const sessions = useStore((state) => state.sessions);
  const selectedAgent = useStore((state) => state.selectedAgent);
  const setActiveStreamParser = useStore((state) => state.setActiveStreamParser);
  const setIsGenerating = useStore((state) => state.setIsGenerating);

  // Use shared expansion hook - approval widgets auto-expand when pending
  const { isExpanded, isActive, isCompleted, handleToggle } =
    useWidgetExpansion({
      widget,
      onExpansionChange,
      autoExpandWhenActive: true, // Auto-expand when pending
      activeStatuses: ["pending"],
      completedStatuses: ["decided"],
      collapseDelay: 4000, // 4 seconds
    });

  // Custom status styles for approval (handles approve/deny decision)
  const getApprovalStatusStyles = () => {
    if (status === "pending") {
      return getWidgetStatusStyles("pending", false);
    } else if (decision === "approve") {
      return getWidgetStatusStyles("success", true);
    } else {
      return getWidgetStatusStyles("failed", true);
    }
  };

  const statusStyles = getApprovalStatusStyles();

  const handleDecision = async (decisionValue: "approve" | "deny") => {
    if (status !== "pending" || isSubmitting) return;

    setIsSubmitting(true);

    try {
      const session = sessions[sessionId];
      if (!session || !selectedAgent) {
        throw new Error("Session or agent not found");
      }

      // Update widget state locally first
      const approvalMessage = session.messages.find((m) =>
        m.widgets.some((w) => w.id === widget.id),
      );
      if (!approvalMessage) {
        throw new Error("Message not found");
      }

      const updatedWidgets = approvalMessage.widgets.map((w) =>
        w.id === widget.id && w.type === "approval"
          ? { ...w, status: "decided" as const, decision: decisionValue }
          : w,
      );
      updateMessage(sessionId, approvalMessage.id, { widgets: updatedWidgets });

      // Send decision to backend via message/stream
      // A2A spec: use same contextId for multi-turn, no taskId in message
      // Use structured DataPart format for better tool_call_id matching
      const toolCallIDs = widget.data.tool_call_ids || [];
      const taskId = widget.data.task_id;
      
      const requestBody = {
        jsonrpc: "2.0",
        method: "message/stream",
        params: {
          message: {
            contextId: session.contextId,
            role: "user",
            parts: [
              // Structured approval (preferred - includes tool_call_id and tool_name)
              ...(toolCallIDs.length > 0
                ? toolCallIDs.map((toolCallID: string) => ({
                    kind: "data",
                    data: {
                      type: "tool_approval",
                      decision: decisionValue,
                      tool_call_id: toolCallID,
                      ...(widget.data.toolName ? { tool_name: widget.data.toolName } : {}),
                      ...(taskId ? { task_id: taskId } : {}),
                    },
                  }))
                : // Fallback: simple text approval (server also supports this)
                  [
                    {
                      kind: "text",
                      text: decisionValue,
                    },
                  ]),
            ],
          },
        },
        id: generateShortId(),
      };

      // Use StreamParser to handle the response stream
      const parser = new StreamParser(sessionId, approvalMessage.id);
      setActiveStreamParser(parser);
      setIsGenerating(true);

      try {
        // A2A spec: POST to agent's URL - streaming is determined by method name
        await parser.stream(selectedAgent.url, requestBody);
      } catch (streamError: unknown) {
        if (streamError instanceof Error && streamError.name !== "AbortError") {
          throw streamError;
        }
      } finally {
        // Note: Don't set isGenerating(false) here - StreamParser.stream() handles it
        // This prevents prematurely showing the send button while the agent continues
        setActiveStreamParser(null);
      }
    } catch (error: unknown) {
      // Revert widget state on error
      const errorSession = sessions[sessionId];
      const errorMessage = errorSession?.messages.find((m) =>
        m.widgets.some((w) => w.id === widget.id),
      );
      if (errorMessage) {
        const revertedWidgets = errorMessage.widgets.map((w) =>
          w.id === widget.id && w.type === "approval"
            ? { ...w, status: "pending" as const, decision: undefined }
            : w,
        );
        updateMessage(sessionId, errorMessage.id, { widgets: revertedWidgets });
      }
      handleError(error, "Failed to send approval decision");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      className={getWidgetContainerClasses(
        statusStyles,
        isExpanded,
        isCompleted,
      )}
      role="region"
      aria-label={`Approval request for ${toolName}`}
    >
      <div
        className={getWidgetHeaderClasses(statusStyles, isActive)}
        onClick={handleToggle}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            handleToggle();
          }
        }}
        role="button"
        tabIndex={0}
        aria-expanded={isExpanded}
        aria-label={`Toggle ${toolName} approval details. Status: ${status === "pending" ? "awaiting decision" : `${decision}`}`}
      >
        <div className={cn("relative", statusStyles.iconColor)}>
          {isActive && (
            <Sparkles
              size={12}
              className="absolute -top-1 -right-1 animate-pulse opacity-70"
            />
          )}
          <Shield
            size={isCompleted ? 14 : 16}
            className={cn(
              "transition-transform duration-200",
              shouldAnimate &&
                "animate-[badgeLifecycle_2s_ease-in-out_infinite]",
              isExpanded && !isCompleted && "rotate-12",
            )}
          />
        </div>

        <span
          className={cn("font-medium flex-1 text-sm", statusStyles.textColor)}
        >
          Approval Required: {toolName}
        </span>

        <div className="ml-auto flex items-center gap-2">
          {status === "pending" && isSubmitting && (
            <Loader2 size={14} className="animate-spin text-yellow-400" />
          )}
          {status === "decided" &&
            (decision === "approve" ? (
              <Check
                size={14}
                className="text-green-500 transition-all duration-300"
              />
            ) : (
              <X
                size={14}
                className="text-red-500 transition-all duration-300"
              />
            ))}

          <ChevronDown
            size={14}
            className={cn(
              "transition-transform duration-300 text-gray-400",
              isExpanded ? "rotate-0" : "-rotate-90",
            )}
          />
        </div>
      </div>

      <div
        className={cn(
          "overflow-hidden transition-all duration-300 ease-in-out",
          isExpanded ? "max-h-[300px] opacity-100" : "max-h-0 opacity-0",
        )}
      >
        <div
          className={cn(
            "p-3 space-y-2 border-t border-white/10 overflow-y-auto max-h-[300px] scrollbar-thin scrollbar-thumb-white/20 scrollbar-track-transparent",
            isCompleted ? "bg-black/10" : "bg-black/30",
          )}
        >
          {/* Input */}
          <div className="space-y-2">
            <div className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
              Input
            </div>
            <pre className="bg-black/60 p-3 rounded-lg overflow-x-auto text-xs max-h-[120px] overflow-y-auto border border-white/10 text-gray-300 font-mono leading-relaxed scrollbar-thin scrollbar-thumb-white/20 scrollbar-track-transparent">
              {JSON.stringify(toolInput, null, 2)}
            </pre>
          </div>

          {/* Action buttons - only show when pending */}
          {status === "pending" && (
            <div className="flex gap-2 pt-2">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleDecision("approve");
                }}
                disabled={isSubmitting}
                className={cn(
                  "flex-1 bg-white/5 hover:bg-green-500/20 border border-green-500/30 text-green-400 hover:text-green-300 py-2 px-3 rounded-lg flex items-center justify-center gap-2 transition-colors text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isSubmitting ? (
                  <Loader2 size={14} className="animate-spin" />
                ) : (
                  <>
                    <Check size={14} />
                    Approve
                  </>
                )}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleDecision("deny");
                }}
                disabled={isSubmitting}
                className={cn(
                  "flex-1 bg-white/5 hover:bg-red-500/20 border border-red-500/30 text-red-400 hover:text-red-300 py-2 px-3 rounded-lg flex items-center justify-center gap-2 transition-colors text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isSubmitting ? (
                  <Loader2 size={14} className="animate-spin" />
                ) : (
                  <>
                    <X size={14} />
                    Deny
                  </>
                )}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
