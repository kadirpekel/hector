import { useState, useEffect, useRef } from 'react';
import type { Widget } from '../../types';

interface UseWidgetExpansionOptions {
    widget: Widget;
    onExpansionChange?: (expanded: boolean) => void;
    autoExpandWhenActive?: boolean; // Only thinking widgets should auto-expand by default
    activeStatuses?: string[]; // Statuses that count as "active" for auto-expand
    completedStatuses?: string[]; // Statuses that trigger auto-collapse
    collapseDelay?: number; // Delay before auto-collapse (default 4000ms)
}

/**
 * Shared hook for widget expansion behavior
 * Handles auto-expand/collapse logic consistently across all widgets
 */
export function useWidgetExpansion({
    widget,
    onExpansionChange,
    autoExpandWhenActive = false, // Default: don't auto-expand (only thinking should)
    activeStatuses = ['working', 'pending', 'active'],
    completedStatuses = ['success', 'failed', 'completed', 'decided'],
    collapseDelay = 4000, // 4 seconds default
}: UseWidgetExpansionOptions) {
    const status = widget.status;
    const isActive = activeStatuses.includes(status || '');
    const isCompleted = completedStatuses.includes(status || '');
    
    // Determine initial expanded state
    // Also auto-expand if widget has content and is active (content is streaming)
    const hasContent = !!(widget.content && widget.content.length > 0);
    const shouldAutoExpand = autoExpandWhenActive && (isActive || (isActive && hasContent));
    const [localExpanded, setLocalExpanded] = useState(
        widget.isExpanded !== undefined 
            ? widget.isExpanded 
            : shouldAutoExpand
    );
    
    const isExpanded = widget.isExpanded !== undefined ? widget.isExpanded : localExpanded;
    const prevStatusRef = useRef<string | undefined>(status);
    const prevContentLengthRef = useRef(0);
    const collapseTimeoutRef = useRef<number | null>(null);
    const userToggledRef = useRef(false);

    // Auto-expand/collapse logic
    useEffect(() => {
        const prevStatus = prevStatusRef.current;
        const statusChanged = prevStatus !== status;
        prevStatusRef.current = status;

        // Track content changes for streaming detection
        const currentContentLength = widget.content?.length || 0;
        const contentAppeared = currentContentLength > prevContentLengthRef.current;
        prevContentLengthRef.current = currentContentLength;

        // Clear any pending collapse timeout
        if (collapseTimeoutRef.current !== null) {
            window.clearTimeout(collapseTimeoutRef.current);
            collapseTimeoutRef.current = null;
        }

        // Auto-expand when:
        // 1. Status becomes active (statusChanged && isActive)
        // 2. Content starts appearing while active (contentAppeared && isActive)
        const shouldExpand = (isActive && statusChanged) || (isActive && contentAppeared);
        
        if (shouldExpand && autoExpandWhenActive && !isExpanded && !userToggledRef.current) {
            requestAnimationFrame(() => {
                const newExpanded = true;
                if (widget.isExpanded === undefined) {
                    setLocalExpanded(newExpanded);
                }
                onExpansionChange?.(newExpanded);
            });
        }
        // Auto-collapse when completed (after delay)
        else if (isCompleted && statusChanged) {
            if (isExpanded && !userToggledRef.current) {
                collapseTimeoutRef.current = window.setTimeout(() => {
                    const newExpanded = false;
                    if (widget.isExpanded === undefined) {
                        setLocalExpanded(newExpanded);
                    }
                    onExpansionChange?.(newExpanded);
                }, collapseDelay);
            }
        }

        // Reset user toggle flag when status changes
        if (statusChanged) {
            userToggledRef.current = false;
        }

        // Cleanup timeout on unmount
        return () => {
            if (collapseTimeoutRef.current !== null) {
                window.clearTimeout(collapseTimeoutRef.current);
            }
        };
    }, [status, isExpanded, widget.isExpanded, widget.content, onExpansionChange, isActive, isCompleted, autoExpandWhenActive, collapseDelay]);

    // Sync local expansion state to store on unmount
    useEffect(() => {
        return () => {
            if (widget.isExpanded === undefined && localExpanded !== (widget.isExpanded ?? false)) {
                onExpansionChange?.(localExpanded);
            }
        };
    }, [localExpanded, widget.isExpanded, onExpansionChange]);

    const handleToggle = () => {
        // Clear any pending auto-collapse when user manually toggles
        if (collapseTimeoutRef.current !== null) {
            window.clearTimeout(collapseTimeoutRef.current);
            collapseTimeoutRef.current = null;
        }

        // Mark that user manually toggled (prevents auto-expand/collapse)
        userToggledRef.current = true;

        const newExpanded = !isExpanded;
        if (widget.isExpanded === undefined) {
            setLocalExpanded(newExpanded);
        }
        onExpansionChange?.(newExpanded);
    };

    return {
        isExpanded,
        isActive,
        isCompleted,
        handleToggle,
    };
}

