import type { Widget, Message } from '../types';

/**
 * Determines if a widget should animate based on its lifecycle state
 * Matches legacy Alpine.js implementation logic
 */
export function isWidgetInLifecycle(
    widget: Widget,
    message: Message,
    _messageIndex: number,
    isLastMessage: boolean,
    isGenerating: boolean
): boolean {
    if (!widget) return false;

    // If no status, assume it's pending/active (should animate)
    if (!widget.status) {
        return true;
    }

    // Define lifecycle states - GENERIC for all widget types
    const activeStates = ['pending', 'working', 'active'];
    const completedStates = ['success', 'completed', 'decided', 'failed'];

    // NEVER animate completed widgets - applies to ALL types
    if (completedStates.includes(widget.status)) {
        return false;
    }

    // ALWAYS animate active widgets - applies to ALL types
    if (activeStates.includes(widget.status)) {
        return true;
    }

    // Special case: thinking widgets during typing (when status might be undefined/null)
    // This is the ONLY special case needed, and it's for edge cases during streaming
    if (widget.type === 'thinking') {
        const isTyping = isGenerating && message && message.role === 'agent' && isLastMessage;
        // Only animate if typing AND widget is not completed
        if (isTyping && widget.status !== 'completed') {
            return true;
        }
    }

    // Default: don't animate unknown states
    return false;
}

