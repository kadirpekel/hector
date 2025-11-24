import React, { useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import { User, Bot, AlertCircle } from 'lucide-react';
import type { Message, Widget } from '../../types';
import { cn } from '../../lib/utils';
import { ToolWidget } from '../Widgets/ToolWidget';
import { ThinkingWidget } from '../Widgets/ThinkingWidget';
import { ApprovalWidget } from '../Widgets/ApprovalWidget';
import { ImageWidget } from '../Widgets/ImageWidget';
import { useStore } from '../../store/useStore';
import { isWidgetInLifecycle } from '../../lib/widget-animations';
import 'highlight.js/styles/github-dark.css';

// Shared ReactMarkdown configuration
const markdownComponents = {
    a: ({ node, ...props }: any) => <a {...props} className="text-hector-green hover:underline" target="_blank" rel="noopener noreferrer" />,
    code: ({ node, inline, className, children, ...props }: any) => {
        const match = /language-(\w+)/.exec(className || '')
        return !inline && match ? (
            <code className={className} {...props}>
                {children}
            </code>
        ) : (
            <code className="bg-white/10 rounded px-1 py-0.5 text-xs font-mono" {...props}>
                {children}
            </code>
        )
    }
};

interface MessageItemProps {
    message: Message;
}

interface MessageItemWithContextProps extends MessageItemProps {
    messageIndex: number;
    isLastMessage: boolean;
}

export const MessageItem: React.FC<MessageItemWithContextProps> = ({ message, messageIndex, isLastMessage }) => {
    const { currentSessionId, getWidgetExpanded } = useStore();
    const isUser = message.role === 'user';
    const isSystem = message.role === 'system';

    // Restore widget expansion state from store
    useEffect(() => {
        if (currentSessionId && message.widgets) {
            message.widgets.forEach((widget) => {
                const savedExpanded = getWidgetExpanded(currentSessionId, message.id, widget.id);
                if (savedExpanded !== undefined && widget.isExpanded !== savedExpanded) {
                    widget.isExpanded = savedExpanded;
                }
            });
        }
    }, [currentSessionId, message.id, message.widgets, getWidgetExpanded]);

    if (isSystem) {
        return (
            <div className="flex items-center justify-center gap-2 text-yellow-500 text-sm py-2 opacity-80">
                <AlertCircle size={14} />
                <span>{message.text}</span>
            </div>
        );
    }

    return (
        <div className={cn(
            "flex gap-4 group",
            isUser ? "flex-row-reverse" : "flex-row"
        )}>
            {/* Avatar */}
            <div className={cn(
                "w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 shadow-lg",
                isUser ? "bg-blue-600" : "bg-hector-green"
            )}>
                {isUser ? <User size={16} className="text-white" /> : <Bot size={16} className="text-white" />}
            </div>

            {/* Content */}
            <div className={cn(
                "flex flex-col min-w-0",
                isUser 
                    ? "max-w-[85%] md:max-w-[75%] lg:max-w-[65%] xl:max-w-[55%] items-end"
                    : "w-full items-start"
            )}>
                {/* Header */}
                <div className="flex items-center gap-2 mb-1 opacity-50 text-xs">
                    <span className="font-medium">{isUser ? 'You' : 'Hector'}</span>
                    <span>{message.time}</span>
                </div>

                {/* Message Bubble */}
                <div className={cn(
                    "rounded-2xl px-4 py-3 shadow-md text-sm leading-relaxed overflow-hidden break-words w-full",
                    isUser
                        ? "bg-blue-600/20 border border-blue-500/30 text-blue-50 rounded-tr-sm"
                        : "bg-white/5 border border-white/10 text-gray-100 rounded-tl-sm"
                )}>
                    {/* Attached Images */}
                    {message.metadata?.images && message.metadata.images.length > 0 && (
                        <div className="flex flex-wrap gap-2 mb-3">
                            {message.metadata.images.map((img, idx) => (
                                <div key={idx} className="relative group/img overflow-hidden rounded-lg border border-white/10">
                                    <img
                                        src={img.preview}
                                        alt={img.file.name}
                                        className="h-32 w-auto object-cover transition-transform hover:scale-105"
                                    />
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Render content in order: widgets and text based on contentOrder */}
                    {(() => {
                        const contentOrder = message.metadata?.contentOrder || [];
                        const widgetsMap = new Map(message.widgets.map(w => [w.id, w]));
                        
                        // If we have contentOrder, render widgets in that order, then text
                        // Otherwise, render text first, then widgets (backward compatibility)
                        if (contentOrder.length > 0) {
                            return (
                                <>
                                    {/* Render widgets in order */
                                    contentOrder.map((widgetId) => {
                                        const widget = widgetsMap.get(widgetId);
                                        if (!widget) return null;
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
                                    
                                    {/* Render any widgets not in contentOrder */}
                                    {message.widgets
                                        .filter(w => !contentOrder.includes(w.id))
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
                                    
                                    {/* Text content after widgets */}
                                    {message.text && (
                                        <div className="prose prose-invert prose-sm max-w-none mt-3">
                                            <ReactMarkdown
                                                remarkPlugins={[remarkGfm]}
                                                rehypePlugins={[rehypeHighlight]}
                                                components={markdownComponents}
                                            >
                                                {message.text}
                                            </ReactMarkdown>
                                        </div>
                                    )}
                                </>
                            );
                        }
                        
                        // Fallback: render text first, then widgets (backward compatibility)
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
                                
                                {message.widgets && message.widgets.length > 0 && (
                                    <div className="mt-3 space-y-2">
                                        {message.widgets.map((widget) => (
                                            <WidgetRenderer
                                                key={widget.id}
                                                widget={widget}
                                                sessionId={currentSessionId || undefined}
                                                messageId={message.id}
                                                message={message}
                                                messageIndex={messageIndex}
                                                isLastMessage={isLastMessage}
                                            />
                                        ))}
                                    </div>
                                )}
                            </>
                        );
                    })()}
                </div>
                
                {/* Cancellation indicator */}
                {message.cancelled && message.role === 'agent' && (
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
}> = ({ widget, sessionId, messageId, message, messageIndex, isLastMessage }) => {
    const { setWidgetExpanded, isGenerating: isGeneratingState } = useStore();

    const handleExpansionChange = (expanded: boolean) => {
        if (sessionId) {
            setWidgetExpanded(sessionId, messageId, widget.id, expanded);
        }
    };

    const shouldAnimate = isWidgetInLifecycle(widget, message, messageIndex, isLastMessage, isGeneratingState);

    switch (widget.type) {
        case 'tool':
            return <ToolWidget widget={widget} onExpansionChange={handleExpansionChange} shouldAnimate={shouldAnimate} />;
        case 'thinking':
            return <ThinkingWidget widget={widget} onExpansionChange={handleExpansionChange} shouldAnimate={shouldAnimate} />;
        case 'approval':
            return sessionId ? <ApprovalWidget widget={widget} sessionId={sessionId} onExpansionChange={handleExpansionChange} shouldAnimate={shouldAnimate} /> : null;
        case 'image':
            return <ImageWidget widget={widget} onExpansionChange={handleExpansionChange} />;
        default:
            return (
                <div className="bg-black/30 border border-white/10 rounded p-2 text-xs font-mono text-gray-500">
                    Unknown widget type: {widget.type}
                </div>
            );
    }
};
