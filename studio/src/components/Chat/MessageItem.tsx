import React, { useMemo } from "react";
import { User, Bot, AlertCircle, Sparkles } from "lucide-react";
import type {
  Message,
  Widget,
  TextWidget as TextWidgetType,
} from "../../types";
import { cn } from "../../lib/utils";
import { ToolWidget } from "../Widgets/ToolWidget";
import { ThinkingWidget } from "../Widgets/ThinkingWidget";
import { ApprovalWidget } from "../Widgets/ApprovalWidget";
import { ImageWidget } from "../Widgets/ImageWidget";
import { TodoWidget } from "../Widgets/TodoWidget";
import { MemoryWidget } from "../Widgets/MemoryWidget";
import { useStore } from "../../store/useStore";

// ... (skipping unchanged lines)



import { isWidgetInLifecycle } from "../../lib/widget-animations";
import { getAgentBranding } from "../../lib/colors";
import { ThrottledMarkdown } from "../Markdown/ThrottledMarkdown";

// Streaming text widget that reads from the optimized buffer
const StreamingTextWidget: React.FC<{
  widget: TextWidgetType;
  components?: any;
}> = React.memo(
  ({ widget, components }) => {
    // Subscribe to streaming buffer for this widget
    // This selector only triggers re-renders when THIS widget's content changes
    const streamingContent = useStore(
      (state) => state.streamingTextContent[widget.id],
    );

    // Use streaming content if available (actively streaming), otherwise use widget content
    const content = streamingContent || widget.content || "";

    if (!content) return null;

    return (
      <div className="prose prose-invert prose-sm max-w-none">
        <ThrottledMarkdown content={content} components={components} />
      </div>
    );
  },
  (prevProps, nextProps) => {
    // Must re-render when:
    // 1. Widget ID changes (different widget)
    // 2. Widget content changes (finalization commits content)
    // Streaming updates are handled by Zustand subscription, not props
    return (
      prevProps.widget.id === nextProps.widget.id &&
      prevProps.widget.content === nextProps.widget.content
    );
  },
);

// Component-specific markdown configuration
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
  messageId: string;
  messageIndex: number;
  isLastMessage: boolean;
  isGenerating: boolean;
  currentSessionId: string | null;
}

interface BubbleGroup {
  id: string; // Unique ID for the group
  author?: string;
  isUser: boolean;
  widgetIds: string[];
}

const MessageItemComponent: React.FC<MessageItemWithContextProps> = ({
  messageId,
  messageIndex,
  isLastMessage,
  isGenerating,
  currentSessionId,
}) => {
  // PERFORMANCE OPTIMIZATION: Subscribe to specific message
  // Zustand's structural sharing ensures we only re-render when message reference changes
  const message = useStore((state) => {
    if (!currentSessionId) return undefined;
    const session = state.sessions[currentSessionId];
    return session?.messages.find((m) => m.id === messageId);
  });

  // Subscribe to available agents for consistent branding
  const activeAgents = useStore((state) => state.availableAgents);

  // Derived state to check if we should render anything
  const shouldRender = useMemo(() => {
    if (!message) return false;
    if (
      message.role === "agent" &&
      !message.text &&
      (!message.widgets || message.widgets.length === 0)
    ) {
      return false;
    }
    return true;
  }, [message]);

  const isUser = message?.role === "user";
  const isSystem = message?.role === "system";

  // Memoize widget map
  const widgetsMap = useMemo(() => {
    if (!message || !message.widgets) return new Map();
    return new Map(message.widgets.map((w) => [w.id, w]));
  }, [message?.widgets]);

  // Memoize contentOrder
  const contentOrder = useMemo(
    () => message?.metadata?.contentOrder || [],
    [message?.metadata?.contentOrder],
  );

  // Group widgets
  const bubbleGroups = useMemo(() => {
    if (!message || !message.widgets) return [];

    const orderedIds = [...contentOrder];

    message.widgets.forEach((w) => {
      if (!orderedIds.includes(w.id)) {
        orderedIds.push(w.id);
      }
    });

    if (orderedIds.length === 0) return [];

    const groups: BubbleGroup[] = [];
    let currentAuthor: string | undefined = undefined;
    let currentWidgetIds: string[] = [];

    const getWidgetAuthor = (widgetId: string): string | undefined => {
      const w = widgetsMap.get(widgetId);
      if (!w) return undefined;
      if (w.type === "text" && w.data.author) return w.data.author;
      if (w.type === "tool" && w.data.author) return w.data.author;
      if (w.type === "thinking" && w.data.author) return w.data.author;
      return undefined;
    };

    orderedIds.forEach((widgetId) => {
      const author = getWidgetAuthor(widgetId);
      // Case-insensitive comparison for consistent grouping
      const authorLower = author?.toLowerCase();
      const currentAuthorLower = currentAuthor?.toLowerCase();
      const shouldSplit = author && authorLower !== currentAuthorLower;

      if (shouldSplit && currentWidgetIds.length > 0) {
        groups.push({
          id: `group_${groups.length}`,
          author: currentAuthor,
          isUser: false,
          widgetIds: [...currentWidgetIds],
        });
        currentWidgetIds = [];
      }

      if (author) currentAuthor = author;
      currentWidgetIds.push(widgetId);
    });

    if (currentWidgetIds.length > 0) {
      groups.push({
        id: `group_${groups.length}`,
        author: currentAuthor,
        isUser: false,
        widgetIds: currentWidgetIds,
      });
    }

    return groups;
  }, [contentOrder, widgetsMap, message?.widgets]);

  // FINAL RENDER CHECK (Moved after all hooks)
  if (!message || !shouldRender) return null;

  if (isSystem) {
    return (
      <div className="flex items-center justify-center gap-2 text-yellow-500 text-sm py-2 opacity-80">
        <AlertCircle size={14} />
        <span>{message.text}</span>
      </div>
    );
  }

  // User Message
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
              img.mediaType.startsWith("audio/") ? (
                <div key={idx} className="mb-3">
                  <audio controls src={img.preview} className="w-full" />
                </div>
              ) : (
                <div
                  key={idx}
                  className="relative group/img overflow-hidden rounded-lg border border-white/10 mb-3"
                >
                  <img
                    src={img.preview}
                    alt=""
                    className="h-32 w-auto object-cover"
                  />
                </div>
              )
            ))}
            {message.text && (
              <div className="prose prose-invert prose-sm max-w-none">
                <ThrottledMarkdown
                  content={message.text}
                  components={markdownComponents}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // Agent Message
  return (
    <div className="flex flex-col gap-4">
      {bubbleGroups.map((group, groupIndex) => {
        // Fallback to message author if group author is missing (e.g. from partial event)
        const authorName = group.author || (message.metadata?.author as string) || "Hector";

        // Use getAgentBranding for consistent styling
        // Resolved agent via reactive store hook (safe for re-renders)
        // ROBUST MATCHING: The backend sends 'author' as the slug (e.g. 'pr-reviewer'), 
        // but the agent list has Display Names (e.g. 'PR Reviewer'). 
        // We use the agent's URL (e.g. '/agents/pr-reviewer') to extract the true slug for matching.
        let knownAgentId: string | undefined = undefined;
        const knownAgent = activeAgents?.find(a => {
          // 1. Try exact name match (legacy/fallback)
          if (a.name === authorName) return true;

          // 2. Try matching slug from URL (robust)
          // URL format: http://.../agents/{slug} or /agents/{slug}
          if (a.url) {
            const parts = a.url.replace(/\/$/, '').split('/'); // Remove trailing slash if any
            const slug = parts[parts.length - 1];
            if (slug === authorName) {
              knownAgentId = slug;
              return true;
            }
            // Also partial match if authorName might be the ID but wasn't found by name
            if (parts[parts.length - 1] === authorName) {
              knownAgentId = parts[parts.length - 1];
              return true;
            }
          }

          return false;
        });

        // If we found variables but didn't set ID (e.g. exact name match), try to extract it now
        if (knownAgent && !knownAgentId && knownAgent.url) {
          try {
            const parts = knownAgent.url.replace(/\/$/, '').split('/');
            knownAgentId = parts[parts.length - 1];
          } catch (e) { }
        }

        // If still no ID found (e.g. agent not in list), try to use authorName as ID if it looks like a slug
        // But for now, just fallback to undefined which defaults to name-hash in colors.ts
        if (!knownAgentId && authorName) {
          // Optimistic assumption: authorName IS the ID/slug coming from backend events
          knownAgentId = authorName;
        }

        const branding = getAgentBranding({
          name: authorName,
          id: knownAgentId,
          type: knownAgent?.type // Pass resolved type (e.g. 'conditional') -> Purple
        });
        const displayName = knownAgent?.name || authorName;
        const isLastGroup = groupIndex === bubbleGroups.length - 1;

        return (
          <div key={group.id} className="flex flex-row gap-4 group">
            <div
              className={cn(
                "w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg transition-colors border border-white/10",
                branding.classes.bg,
              )}
            >
              <Bot size={16} className="text-white" />
            </div>

            <div className="flex flex-col min-w-0 w-full items-start">
              <div className="flex items-center gap-2 mb-1 text-xs">
                <span
                  className={cn(
                    "font-bold uppercase tracking-wider",
                    branding.classes.text,
                  )}
                >
                  {displayName}
                </span>
                <span className="opacity-50">{message.time}</span>
              </div>

              <div
                className={cn(
                  "rounded-2xl px-4 py-3 shadow-md text-sm leading-relaxed overflow-hidden break-words w-full rounded-tl-sm transition-colors",
                  "bg-white/5",
                  branding.classes.border,
                  "border",
                )}
              >
                {group.widgetIds.map((itemId) => {
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
                        isGenerating={isGenerating}
                      />
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        );
      })}

      {bubbleGroups.length === 0 && message.text && (
        <div className="flex flex-row gap-4 group">
          <div
            className={cn(
              "w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg border border-white/10",
              getAgentBranding({ name: (message.metadata?.author as string) || "Hector" }).classes.bg
            )}
          >
            <Bot size={16} className="text-white" />
          </div>
          <div className="flex flex-col min-w-0 w-full items-start">
            <div className="bg-white/5 border border-white/10 text-gray-100 rounded-2xl px-4 py-3 rounded-tl-sm w-full">
              <div className="prose prose-invert prose-sm max-w-none">
                <ThrottledMarkdown
                  content={message.text}
                  components={markdownComponents}
                />
              </div>
            </div>
          </div>
        </div>
      )}

      {message.cancelled && (
        <div className="ml-12 text-xs text-gray-500 italic">user cancelled</div>
      )}
    </div>
  );
};

// Widget Renderer
const WidgetRenderer: React.FC<{
  widget: Widget;
  sessionId?: string;
  messageId: string;
  message: Message;
  messageIndex: number;
  isLastMessage: boolean;
  isGenerating?: boolean;
}> = React.memo(
  ({
    widget,
    sessionId,
    messageId,
    message,
    messageIndex,
    isLastMessage,
    isGenerating = false,
  }) => {
    const handleExpansionChange = (expanded: boolean) => {
      if (sessionId) {
        useStore
          .getState()
          .setWidgetExpanded(sessionId, messageId, widget.id, expanded);
      }
    };

    const shouldAnimate = isWidgetInLifecycle(
      widget,
      message,
      messageIndex,
      isLastMessage,
      isGenerating,
    );

    switch (widget.type) {
      case "tool":
        // Get taskId from session for scoped cancellation
        const taskId = sessionId
          ? useStore.getState().sessions[sessionId]?.taskId
          : null;
        if (widget.data.name === "todo_write") {
          return (
            <TodoWidget
              widget={widget}
              onExpansionChange={handleExpansionChange}
              shouldAnimate={shouldAnimate}
            />
          );
        }
        if (widget.data.name === "memory_search") {
          return (
            <MemoryWidget
              widget={widget}
              onExpansionChange={handleExpansionChange}
              shouldAnimate={shouldAnimate}
            />
          );
        }

        return (
          <ToolWidget
            widget={widget}
            taskId={taskId}
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
      case "progress":
        return (
          <div className="flex items-center gap-3 p-3 bg-hector-green/5 border border-hector-green/20 rounded-md text-sm text-gray-200 animate-in fade-in slide-in-from-left-2 my-2">
            <div className="relative flex items-center justify-center w-5 h-5">
              {widget.status === "completed" ? (
                <div className="w-2 h-2 rounded-full bg-hector-green" />
              ) : (
                <>
                  <div className="absolute inset-0 bg-hector-green/20 rounded-full animate-ping" />
                  <Sparkles className="w-4 h-4 text-hector-green relative z-10 animate-pulse" />
                </>
              )}
            </div>
            <div className="flex-1 flex flex-col">
              <span className="font-medium text-hector-green">{widget.content}</span>
            </div>
          </div>
        );
      case "text":
        return (
          <StreamingTextWidget
            widget={widget}
            components={markdownComponents}
          />
        );
      default: {
        return null;
      }
    }
  },
  (prevProps, nextProps) => {
    return (
      prevProps.widget === nextProps.widget &&
      prevProps.isLastMessage === nextProps.isLastMessage &&
      prevProps.messageIndex === nextProps.messageIndex &&
      prevProps.messageId === nextProps.messageId &&
      prevProps.sessionId === nextProps.sessionId &&
      prevProps.isGenerating === nextProps.isGenerating
    );
  },
);

export const MessageItem = React.memo(
  MessageItemComponent,
  (prevProps, nextProps) => {
    if (prevProps.messageId !== nextProps.messageId) return false;
    if (prevProps.isLastMessage !== nextProps.isLastMessage) return false;
    if (prevProps.messageIndex !== nextProps.messageIndex) return false;
    if (prevProps.isGenerating !== nextProps.isGenerating) return false;
    if (prevProps.currentSessionId !== nextProps.currentSessionId) return false;
    return true;
  },
);
