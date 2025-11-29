import React, { useState, useEffect } from "react";
import {
  Shield,
  Check,
  X,
  ChevronDown,
  ChevronRight,
  Loader2,
} from "lucide-react";
import type { Widget } from "../../types";
import { cn } from "../../lib/utils";
import { useStore } from "../../store/useStore";
import { StreamParser } from "../../lib/stream-parser";
import { handleError } from "../../lib/error-handler";
import { generateShortId } from "../../lib/id-generator";

interface ApprovalWidgetProps {
  widget: Widget;
  sessionId: string;
  onExpansionChange?: (expanded: boolean) => void;
  shouldAnimate?: boolean;
}

export const ApprovalWidget: React.FC<ApprovalWidgetProps> = ({
  widget,
  sessionId,
  onExpansionChange,
  shouldAnimate = false,
}) => {
  // Auto-expand when pending for better UX, collapse when decided
  const [isExpanded, setIsExpanded] = useState(
    widget.isExpanded !== undefined
      ? widget.isExpanded
      : widget.status === "pending",
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { toolName, toolInput } = widget.data;
  const { status, decision } = widget;
  const {
    updateMessage,
    sessions,
    selectedAgent,
    setActiveStreamParser,
    setIsGenerating,
  } = useStore();

  // Update expanded state when widget status changes
  useEffect(() => {
    if (widget.status === "decided" && isExpanded) {
      // Collapse immediately when decided
      setIsExpanded(false);
    } else if (widget.status === "pending" && !isExpanded) {
      setIsExpanded(true);
    }
  }, [widget.status]); // Remove isExpanded from dependencies to avoid loops

  // Sync local expansion state to store on unmount (handles edge case where local state
  // changes via auto-expand but user navigates away before toggling)
  useEffect(() => {
    return () => {
      if (widget.isExpanded !== isExpanded) {
        onExpansionChange?.(isExpanded);
      }
    };
  }, [isExpanded, widget.isExpanded, onExpansionChange]);

  const handleToggle = () => {
    const newExpanded = !isExpanded;
    setIsExpanded(newExpanded);
    onExpansionChange?.(newExpanded);
  };

  const handleDecision = async (decisionValue: "approve" | "deny") => {
    if (status !== "pending" || isSubmitting) return;

    setIsSubmitting(true);

    try {
      const session = sessions[sessionId];
      if (!session || !selectedAgent) {
        throw new Error("Session or agent not found");
      }

      const taskId = session.taskId;
      if (!taskId) {
        throw new Error("Task ID not found - cannot send approval decision");
      }

      // Update widget state locally first
      const approvalMessage = session.messages.find((m) =>
        m.widgets.some((w) => w.id === widget.id),
      );
      if (!approvalMessage) {
        throw new Error("Message not found");
      }

      const updatedWidgets = approvalMessage.widgets.map((w) =>
        w.id === widget.id
          ? { ...w, status: "decided", decision: decisionValue }
          : w,
      );
      updateMessage(sessionId, approvalMessage.id, { widgets: updatedWidgets });

      // Send decision to backend via message/stream endpoint with taskId
      const requestBody = {
        jsonrpc: "2.0",
        method: "message/stream",
        params: {
          request: {
            contextId: session.contextId,
            taskId: taskId,
            role: "user",
            parts: [
              {
                text: decisionValue, // Backend parses "approve" or "deny" from text
              },
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
        await parser.stream(`${selectedAgent.url}/stream`, requestBody);
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
          w.id === widget.id
            ? { ...w, status: "pending", decision: undefined }
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
    <div className="border border-white/10 rounded-lg bg-black/20 overflow-hidden text-sm">
      <div
        className="flex items-center gap-2 p-2 bg-white/5 cursor-pointer hover:bg-white/10 transition-colors"
        onClick={handleToggle}
      >
        <Shield
          size={14}
          className={cn(
            status === "pending"
              ? "text-yellow-400"
              : decision === "approve"
                ? "text-green-400"
                : "text-red-400",
            shouldAnimate && "animate-[badgeLifecycle_2s_ease-in-out_infinite]",
          )}
        />
        <span
          className={cn(
            "font-medium",
            status === "pending"
              ? "text-yellow-200"
              : decision === "approve"
                ? "text-green-200"
                : "text-red-200",
          )}
        >
          Approval Required: {toolName}
        </span>

        <div className="ml-auto flex items-center gap-2">
          {status === "pending" && isSubmitting && (
            <Loader2 size={14} className="animate-spin text-yellow-400" />
          )}
          {status === "decided" &&
            (decision === "approve" ? (
              <Check size={14} className="text-green-400" />
            ) : (
              <X size={14} className="text-red-400" />
            ))}

          {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </div>
      </div>

      {isExpanded && (
        <div className="p-3 space-y-3 border-t border-white/10 font-mono text-xs">
          {/* Input */}
          <div>
            <div className="text-gray-500 mb-1">Input:</div>
            <pre className="bg-black/40 p-2 rounded overflow-x-auto text-gray-300">
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
                  "flex-1 bg-white/5 hover:bg-green-500/20 border border-green-500/30 text-green-400 hover:text-green-300 py-1.5 px-3 rounded flex items-center justify-center gap-2 transition-colors text-xs disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isSubmitting ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <>
                    <Check size={12} />
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
                  "flex-1 bg-white/5 hover:bg-red-500/20 border border-red-500/30 text-red-400 hover:text-red-300 py-1.5 px-3 rounded flex items-center justify-center gap-2 transition-colors text-xs disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {isSubmitting ? (
                  <Loader2 size={12} className="animate-spin" />
                ) : (
                  <>
                    <X size={12} />
                    Deny
                  </>
                )}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
