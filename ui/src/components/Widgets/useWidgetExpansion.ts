import { useState, useEffect, useRef, useCallback } from "react";
import type { Widget } from "../../types";
import { WIDGET } from "../../lib/constants";

interface UseWidgetExpansionOptions {
  widget: Widget;
  onExpansionChange?: (expanded: boolean) => void;
  autoExpandWhenActive?: boolean;
  activeStatuses?: string[];
  completedStatuses?: string[];
  collapseDelay?: number;
}

/**
 * Widget Expansion State Machine
 * ==============================
 *
 * This hook manages widget expansion with the following states:
 *
 * STATES:
 *   - collapsed: Widget is minimized
 *   - expanded: Widget is showing full content
 *   - pending_collapse: Widget will collapse after delay (completed status)
 *
 * TRANSITIONS:
 *   - collapsed → expanded: When status becomes active OR content starts streaming
 *   - expanded → pending_collapse: When status becomes completed
 *   - pending_collapse → collapsed: After collapseDelay timeout
 *   - any → (manual): User toggle overrides all automatic behavior
 *
 * OVERRIDE BEHAVIOR:
 *   - User manual toggle sets userToggled=true
 *   - While userToggled=true, no automatic transitions occur
 *   - userToggled resets to false when status changes
 */
export function useWidgetExpansion({
  widget,
  onExpansionChange,
  autoExpandWhenActive = false,
  activeStatuses = ["working", "pending", "active"],
  completedStatuses = ["success", "failed", "completed", "decided"],
  collapseDelay = WIDGET.COLLAPSE_DELAY_MS,
}: UseWidgetExpansionOptions) {
  // Derived state
  const status = widget.status;
  const isActive = activeStatuses.includes(status || "");
  const isCompleted = completedStatuses.includes(status || "");

  // Expansion state
  const [localExpanded, setLocalExpanded] = useState(() => {
    if (widget.isExpanded !== undefined) return widget.isExpanded;
    return autoExpandWhenActive && isActive;
  });

  // Refs for tracking changes between renders
  const prevStatusRef = useRef(status);
  const prevContentLengthRef = useRef(widget.content?.length || 0);
  const collapseTimeoutRef = useRef<number | null>(null);
  const userToggledRef = useRef(false);

  // Computed: actual expanded state (prop takes precedence over local)
  const isExpanded = widget.isExpanded ?? localExpanded;

  // Helper: update expansion state
  const setExpanded = useCallback(
    (expanded: boolean) => {
      if (widget.isExpanded === undefined) {
        setLocalExpanded(expanded);
      }
      onExpansionChange?.(expanded);
    },
    [widget.isExpanded, onExpansionChange],
  );

  // Helper: clear any pending collapse timeout
  const clearCollapseTimeout = useCallback(() => {
    if (collapseTimeoutRef.current !== null) {
      window.clearTimeout(collapseTimeoutRef.current);
      collapseTimeoutRef.current = null;
    }
  }, []);

  // Main effect: handle automatic expansion/collapse
  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    const statusChanged = prevStatus !== status;
    prevStatusRef.current = status;

    const currentContentLength = widget.content?.length || 0;
    const contentAppeared = currentContentLength > prevContentLengthRef.current;
    prevContentLengthRef.current = currentContentLength;

    // Reset user override when status changes
    if (statusChanged) {
      userToggledRef.current = false;
    }

    // Skip automatic behavior if user manually toggled
    if (userToggledRef.current) {
      return;
    }

    clearCollapseTimeout();

    // Auto-expand: status became active OR content started streaming while active
    if (autoExpandWhenActive && !isExpanded) {
      const shouldExpand =
        (isActive && statusChanged) || (isActive && contentAppeared);
      if (shouldExpand) {
        requestAnimationFrame(() => setExpanded(true));
        return;
      }
    }

    // Auto-collapse: status became completed (with delay)
    if (isCompleted && statusChanged && isExpanded) {
      collapseTimeoutRef.current = window.setTimeout(() => {
        setExpanded(false);
      }, collapseDelay);
    }

    return clearCollapseTimeout;
  }, [
    status,
    isExpanded,
    widget.content,
    autoExpandWhenActive,
    isActive,
    isCompleted,
    collapseDelay,
    setExpanded,
    clearCollapseTimeout,
  ]);

  // Cleanup on unmount only - sync final state
  useEffect(() => {
    return () => {
      // Clear any pending timeout
      if (collapseTimeoutRef.current !== null) {
        window.clearTimeout(collapseTimeoutRef.current);
        collapseTimeoutRef.current = null;
      }
    };
  }, []); // Empty deps - cleanup only on unmount

  // Manual toggle handler
  const handleToggle = useCallback(() => {
    clearCollapseTimeout();
    userToggledRef.current = true;
    setExpanded(!isExpanded);
  }, [isExpanded, setExpanded, clearCollapseTimeout]);

  return {
    isExpanded,
    isActive,
    isCompleted,
    handleToggle,
  };
}
