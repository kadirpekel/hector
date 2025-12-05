import { useMemo } from "react";
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
import { TodoList } from "./TodoList";
import { useStore } from "../../store/useStore";
import { isWidgetInLifecycle } from "../../lib/widget-animations";
import type { TodoItem } from "../../types";
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

export const MessageItem: React.FC<MessageItemWithContextProps> = ({
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
    [message.widgets]
  );

  // Memoize contentOrder to prevent unnecessary comparisons
  const contentOrder = useMemo(
    () => message.metadata?.contentOrder || [],
    [message.metadata?.contentOrder]
  );

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

  return (
    <div
      className={cn(
        "flex gap-4 group",
        isUser ? "flex-row-reverse" : "flex-row",
      )}
    >
      {/* Avatar */}
      <div
        className={cn(
          "w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg",
          isUser ? "bg-blue-600" : "bg-hector-green",
        )}
      >
        {isUser ? (
          <User size={16} className="text-white" />
        ) : (
          <Bot size={16} className="text-white" />
        )}
      </div>

      {/* Content */}
      <div
        className={cn(
          "flex flex-col min-w-0",
          isUser
            ? "max-w-[85%] md:max-w-[75%] lg:max-w-[65%] xl:max-w-[55%] items-end"
            : "w-full items-start",
        )}
      >
        {/* Header */}
        <div className="flex items-center gap-2 mb-1 opacity-50 text-xs">
          <span className="font-medium">{isUser ? "You" : "Hector"}</span>
          <span>{message.time}</span>
        </div>

        {/* Message Bubble */}
        <div
          className={cn(
            "rounded-2xl px-4 py-3 shadow-md text-sm leading-relaxed overflow-hidden break-words w-full",
            isUser
              ? "bg-blue-600/20 border border-blue-500/30 text-blue-50 rounded-tr-sm"
              : "bg-white/5 border border-white/10 text-gray-100 rounded-tl-sm",
          )}
        >
          {/* Attached Images */}
          {message.metadata?.images && message.metadata.images.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {message.metadata.images.map((img, idx) => (
                <div
                  key={idx}
                  className="relative group/img overflow-hidden rounded-lg border border-white/10"
                >
                  <img
                    src={img.preview}
                    alt={img.file.name}
                    className="h-32 w-auto object-cover transition-transform hover:scale-105"
                  />
                </div>
              ))}
            </div>
          )}

          {/* Render content in order based on contentOrder */}
          {(() => {
            // Helper functions
            const extractTodos = (widget: Widget): TodoItem[] | null => {
              if (
                widget.type === "tool" &&
                widget.data?.name === "todo_write" &&
                widget.data?.args?.todos &&
                Array.isArray(widget.data.args.todos)
              ) {
                return widget.data.args.todos.map(
                  (todo: Record<string, unknown>) => ({
                    id: (typeof todo.id === "string" ? todo.id : "") || "",
                    content:
                      (typeof todo.content === "string" ? todo.content : "") ||
                      "",
                    status: (typeof todo.status === "string"
                      ? todo.status
                      : "pending") as TodoItem["status"],
                  }),
                );
              }
              return null;
            };

            const isTodoWidget = (widget: Widget): boolean =>
              widget.type === "tool" && widget.data?.name === "todo_write";

            // Use memoized widgetsMap and contentOrder from component level

            // Track accumulated todos as we iterate through contentOrder
            const accumulatedTodosMap = new Map<string, TodoItem>();

            // If we have contentOrder, render in that exact order
            // This preserves the position of tool calls, thinking, and text relative to each other
            // Text is now rendered as TextWidgets with position markers (text_start, text_after_*)
            if (contentOrder.length > 0) {
              return (
                <>
                  {contentOrder.map((itemId) => {
                    const widget = widgetsMap.get(itemId);
                    if (!widget) return null;

                    // If this is a todo_write widget, render the accumulated todos at this point
                    if (isTodoWidget(widget)) {
                      const todos = extractTodos(widget);
                      if (todos) {
                        // Update accumulated todos (merge=true behavior - latest status wins)
                        todos.forEach((todo: TodoItem) => {
                          accumulatedTodosMap.set(todo.id, todo);
                        });
                        // Render the current state of todos at this point in the flow
                        const currentTodos = Array.from(
                          accumulatedTodosMap.values(),
                        );
                        return (
                          <div key={widget.id} className="mb-2">
                            <TodoList todos={currentTodos} />
                          </div>
                        );
                      }
                      return null;
                    }

                    // Render all widgets (including TextWidget) via WidgetRenderer
                    return (
                      <div key={widget.id} className="mb-3">
                        <WidgetRenderer
                          widget={widget}
                          sessionId={currentSessionId || undefined}
                          messageId={message.id}
                          message={message}
                          messageIndex={messageIndex}
                          isLastMessage={isLastMessage}
                        />
                      </div>
                    );
                  })}

                  {/* Render any widgets not in contentOrder (excluding todo widgets) */}
                  {message.widgets
                    .filter(
                      (w) => !contentOrder.includes(w.id) && !isTodoWidget(w),
                    )
                    .map((widget) => (
                      <div key={widget.id} className="mb-3">
                        <WidgetRenderer
                          widget={widget}
                          sessionId={currentSessionId || undefined}
                          messageId={message.id}
                          message={message}
                          messageIndex={messageIndex}
                          isLastMessage={isLastMessage}
                        />
                      </div>
                    ))}
                </>
              );
            }

            // Fallback: render text first, then widgets (with todos inline)
            const fallbackTodosMap = new Map<string, TodoItem>();

            return (
              <>
                {message.text && (
                  <div className="prose prose-invert prose-sm max-w-none">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      rehypePlugins={[rehypeHighlight]}
                      components={markdownComponents}
                    >
                      {message.text}
                    </ReactMarkdown>
                  </div>
                )}

                {/* Render widgets - todos inline */}
                {message.widgets && message.widgets.length > 0 && (
                  <div className="mt-3 space-y-2">
                    {message.widgets.map((widget) => {
                      if (isTodoWidget(widget)) {
                        const todos = extractTodos(widget);
                        if (todos) {
                          // Update accumulated todos
                          todos.forEach((todo: TodoItem) => {
                            fallbackTodosMap.set(todo.id, todo);
                          });
                          // Render current state
                          const currentTodos = Array.from(
                            fallbackTodosMap.values(),
                          );
                          return (
                            <div key={widget.id} className="mb-2">
                              <TodoList todos={currentTodos} />
                            </div>
                          );
                        }
                        return null;
                      }

                      return (
                        <WidgetRenderer
                          key={widget.id}
                          widget={widget}
                          sessionId={currentSessionId || undefined}
                          messageId={message.id}
                          message={message}
                          messageIndex={messageIndex}
                          isLastMessage={isLastMessage}
                        />
                      );
                    })}
                  </div>
                )}
              </>
            );
          })()}
        </div>

        {/* Cancellation indicator */}
        {message.cancelled && message.role === "agent" && (
          <div className="mt-2 px-4 text-xs text-gray-500 italic">
            user cancelled
          </div>
        )}
      </div>
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
}> = ({
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
      const _exhaustiveCheck: never = widget;
      return (
        <div className="bg-black/30 border border-white/10 rounded p-2 text-xs font-mono text-gray-500">
          Unknown widget type: {(_exhaustiveCheck as { type: string }).type}
        </div>
      );
    }
  }
};
