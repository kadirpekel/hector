import React, { useMemo } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { User, Bot, AlertCircle } from "lucide-react";
import type { Message, Widget } from "../../types";
import { cn } from "../../lib/utils";
import { ToolWidget } from "../Widgets/ToolWidget";
import { ThinkingWidget } from "../Widgets/ThinkingWidget";
import { ApprovalWidget } from "../Widgets/ApprovalWidget";
import { ImageWidget } from "../Widgets/ImageWidget";
import { useStore } from "../../store/useStore";
import { isWidgetInLifecycle } from "../../lib/widget-animations";
import { getAgentColor, getAgentColorClasses } from "../../lib/colors";
import "highlight.js/styles/github-dark.css";

// Shared ReactMarkdown configuration
const markdownComponents = {
  a: ({ ...props }: React.ComponentProps<"a">) => (
    <a
      {...props}
      className="text-hector-green hover:underline"
      target="_blank"
      rel="noopener noreferrer"
    />
  ),
  code: ({
    inline,
    className,
    children,
    ...props
  }: React.ComponentProps<"code"> & { inline?: boolean }) => {
    const match = /language-(\w+)/.exec(className || "");
    return !inline && match ? (
      <code className={className} {...props}>
        {children}
      </code>
    ) : (
      <code
        className="bg-white/10 rounded px-1 py-0.5 text-xs font-mono"
        {...props}
      >
        {children}
      </code>
    );
  },
};

interface MessageItemWithContextProps {
  message: Message;
  messageIndex: number;
  isLastMessage: boolean;
}

interface BubbleGroup {
  id: string; // Unique ID for the group
  author?: string;
  isUser: boolean;
  widgetIds: string[];
}

const MessageItemComponent: React.FC<MessageItemWithContextProps> = ({
  message,
  messageIndex,
  isLastMessage,
}) => {
  // Use selector for better performance - only subscribe to currentSessionId
  const currentSessionId = useStore((state) => state.currentSessionId);
  const isUser = message.role === "user";
  const isSystem = message.role === "system";

  // Memoize widget map to prevent recreation on every render
  const widgetsMap = useMemo(
    () => new Map(message.widgets.map((w) => [w.id, w])),
    [message.widgets],
  );

  // Memoize contentOrder to prevent unnecessary comparisons
  const contentOrder = useMemo(
    () => message.metadata?.contentOrder || [],
    [message.metadata?.contentOrder],
  );

  // Group widgets into separate bubbles based on author continuity
  const bubbleGroups = useMemo(() => {
    // Collect all widget IDs from contentOrder first
    const orderedIds = [...contentOrder];

    // Find any widgets that are NOT in contentOrder (orphans) and append them
    // This is a safety net for synchronization issues
    message.widgets.forEach(w => {
      if (!orderedIds.includes(w.id)) {
        orderedIds.push(w.id);
      }
    });

    if (orderedIds.length === 0) return [];

    const groups: BubbleGroup[] = [];
    let currentAuthor: string | undefined = undefined;
    let currentWidgetIds: string[] = [];

    // Helper: Determine author of a widget
    const getWidgetAuthor = (widgetId: string): string | undefined => {
      const w = widgetsMap.get(widgetId);
      if (!w) return undefined;
      // Check data.author for widgets that support it
      if (w.type === 'text' && w.data.author) return w.data.author;
      if (w.type === 'tool' && w.data.author) return w.data.author;
      if (w.type === 'thinking' && w.data.author) return w.data.author;
      // For widgets without author metadata, return undefined
      // This allows proper grouping by treating them as continuation of current author
      return undefined;
    };

    orderedIds.forEach((widgetId) => {
      const author = getWidgetAuthor(widgetId);

      // Transition Logic:
      // If we have a current group, does the new widget belong to it?
      // - If author is defined and DIFFERENT from current -> Split
      // - If author is defined and SAME as current -> Keep
      // - If author is undefined -> Keep (assume continuation/system)

      const shouldSplit = author && author !== currentAuthor;

      if (shouldSplit && currentWidgetIds.length > 0) {
        groups.push({
          id: `group_${groups.length}`,
          author: currentAuthor,
          isUser: false,
          widgetIds: [...currentWidgetIds]
        });
        currentWidgetIds = [];
      }

      if (author) currentAuthor = author;
      currentWidgetIds.push(widgetId);
    });

    // Push final group
    if (currentWidgetIds.length > 0) {
      groups.push({
        id: `group_${groups.length}`,
        author: currentAuthor,
        isUser: false,
        widgetIds: currentWidgetIds
      });
    }

    return groups;
  }, [contentOrder, widgetsMap, message.widgets]);

  // Widget expansion state is managed via widget.isExpanded prop and onExpansionChange callback
  // Widgets read their initial state from the prop and sync changes back via the callback
  // No need for additional sync logic here - widgets handle their own state management

  if (isSystem) {
    return (
      <div className="flex items-center justify-center gap-2 text-yellow-500 text-sm py-2 opacity-80">
        <AlertCircle size={14} />
        <span>{message.text}</span>
      </div>
    );
  }

  // --- RENDER LOGIC ---

  // User Message (Standard single bubble)
  if (isUser) {
    return (
      <div className="flex flex-row-reverse gap-4 group">
        <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg bg-blue-600">
          <User size={16} className="text-white" />
        </div>
        <div className="flex flex-col min-w-0 max-w-[85%] md:max-w-[75%] items-end">
          <div className="flex items-center gap-2 mb-1 opacity-50 text-xs">
            <span className="font-medium">You</span>
            <span>{message.time}</span>
          </div>
          <div className="rounded-2xl px-4 py-3 shadow-md text-sm leading-relaxed overflow-hidden break-words w-full bg-blue-600/20 border border-blue-500/30 text-blue-50 rounded-tr-sm">
            {message.metadata?.images?.map((img, idx) => (
              <div key={idx} className="relative group/img overflow-hidden rounded-lg border border-white/10 mb-3">
                <img src={img.preview} alt="" className="h-32 w-auto object-cover" />
              </div>
            ))}
            {/* Fallback for user text if not in widgets */}
            {message.text && (
              <div className="prose prose-invert prose-sm max-w-none">
                <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={markdownComponents}>
                  {message.text}
                </ReactMarkdown>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // Agent Message (Decoupled Bubbles)
  return (
    <div className="flex flex-col gap-4"> {/* Stack of bubbles */}
      {bubbleGroups.map((group, groupIndex) => {
        const agentColor = getAgentColor(group.author || "Hector");
        const colors = getAgentColorClasses(agentColor);
        const displayName = group.author || "Hector";
        const isLastGroup = groupIndex === bubbleGroups.length - 1;

        return (
          <div key={group.id} className="flex flex-row gap-4 group">
            {/* Avatar (Colored per Agent) */}
            <div className={cn(
              "w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg transition-colors border border-white/10",
              colors.bg
            )}>
              <Bot size={16} className="text-white" />
            </div>

            {/* Bubble Content */}
            <div className="flex flex-col min-w-0 w-full items-start">
              <div className="flex items-center gap-2 mb-1 text-xs">
                <span className={cn("font-bold uppercase tracking-wider", colors.text)}>
                  {displayName}
                </span>
                <span className="opacity-50">{message.time}</span>
              </div>

              <div className={cn(
                "rounded-2xl px-4 py-3 shadow-md text-sm leading-relaxed overflow-hidden break-words w-full rounded-tl-sm transition-colors",
                "bg-white/5", // Base background
                colors.border, // Colored border
                "border"
              )}>
                {/* Render Widgets in this Group */}
                {group.widgetIds.map(itemId => {
                  const widget = widgetsMap.get(itemId);
                  if (!widget) return null;

                  return (
                    <div key={widget.id} className="mb-3 last:mb-0">
                      <WidgetRenderer
                        widget={widget}
                        sessionId={currentSessionId || undefined}
                        messageId={message.id}
                        message={message}
                        messageIndex={messageIndex}
                        isLastMessage={isLastMessage && isLastGroup}
                      />
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        );
      })}

      {/* Fallback: If no contentOrder/widgets but text exists (legacy/error) */}
      {bubbleGroups.length === 0 && message.text && (
        <div className="flex flex-row gap-4 group">
          <div className="w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg bg-hector-green">
            <Bot size={16} className="text-white" />
          </div>
          <div className="flex flex-col min-w-0 w-full items-start">
            <div className="bg-white/5 border border-white/10 text-gray-100 rounded-2xl px-4 py-3 rounded-tl-sm w-full">
              <div className="prose prose-invert prose-sm max-w-none">
                <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} components={markdownComponents}>
                  {message.text}
                </ReactMarkdown>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cancellation Token */}
      {message.cancelled && (
        <div className="ml-12 text-xs text-gray-500 italic">user cancelled</div>
      )}
    </div>
  );
};

// Widget Renderer
// Widget Renderer - Memoized to prevent re-rendering irrelevant widgets
const WidgetRenderer: React.FC<{
  widget: Widget;
  sessionId?: string;
  messageId: string;
  message: Message;
  messageIndex: number;
  isLastMessage: boolean;
}> = React.memo(({
  widget,
  sessionId,
  messageId,
  message,
  messageIndex,
  isLastMessage,
}) => {
  // Use selectors for better performance - only subscribe to specific state slices
  const setWidgetExpanded = useStore((state) => state.setWidgetExpanded);
  const isGeneratingState = useStore((state) => state.isGenerating);

  const handleExpansionChange = (expanded: boolean) => {
    if (sessionId) {
      setWidgetExpanded(sessionId, messageId, widget.id, expanded);
    }
  };

  const shouldAnimate = isWidgetInLifecycle(
    widget,
    message,
    messageIndex,
    isLastMessage,
    isGeneratingState,
  );

  switch (widget.type) {
    case "tool":
      return (
        <ToolWidget
          widget={widget}
          onExpansionChange={handleExpansionChange}
          shouldAnimate={shouldAnimate}
        />
      );
    case "thinking":
      return (
        <ThinkingWidget
          widget={widget}
          onExpansionChange={handleExpansionChange}
          shouldAnimate={shouldAnimate}
        />
      );
    case "approval":
      return sessionId ? (
        <ApprovalWidget
          widget={widget}
          sessionId={sessionId}
          onExpansionChange={handleExpansionChange}
          shouldAnimate={shouldAnimate}
        />
      ) : null;
    case "image":
      return (
        <ImageWidget
          widget={widget}
          onExpansionChange={handleExpansionChange}
        />
      );
    case "text":
      // Text widgets are rendered inline as markdown
      return widget.content ? (
        <div className="prose prose-invert prose-sm max-w-none">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeHighlight]}
            components={markdownComponents}
          >
            {widget.content}
          </ReactMarkdown>
        </div>
      ) : null;
    default: {
      // Exhaustive check - if we reach here, widget.type is 'never'
      return null;
    }
  }
}, (prevProps, nextProps) => {
  // Custom comparison for performance
  return (
    prevProps.widget === nextProps.widget &&
    prevProps.isLastMessage === nextProps.isLastMessage &&
    prevProps.messageIndex === nextProps.messageIndex &&
    prevProps.messageId === nextProps.messageId &&
    prevProps.sessionId === nextProps.sessionId
    // We exclude 'message' from deep comparison because it changes every stream update
    // But the widget object itself (prevProps.widget) is stable if it's a previous bubble
  );
});

export const MessageItem = React.memo(MessageItemComponent);