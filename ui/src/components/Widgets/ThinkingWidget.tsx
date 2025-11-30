import React from 'react';
import { Brain, ChevronDown, CheckCircle2, Loader2, Sparkles } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Widget } from '../../types';
import { cn } from '../../lib/utils';
import { useWidgetExpansion } from './useWidgetExpansion';
import { getWidgetStatusStyles, getWidgetContainerClasses, getWidgetHeaderClasses } from './widgetStyles';
import { useAutoScroll } from './useAutoScroll';

interface ThinkingWidgetProps {
    widget: Widget;
    onExpansionChange?: (expanded: boolean) => void;
    shouldAnimate?: boolean;
}

export const ThinkingWidget: React.FC<ThinkingWidgetProps> = ({ widget, onExpansionChange, shouldAnimate = false }) => {
    const { type } = widget.data;
    const status = widget.status;
    
    // Use shared expansion hook - thinking widgets auto-expand when active
    const { isExpanded, isActive, isCompleted, handleToggle } = useWidgetExpansion({
        widget,
        onExpansionChange,
        autoExpandWhenActive: true, // Thinking widgets always auto-expand
        activeStatuses: ['active'],
        completedStatuses: ['completed'],
        collapseDelay: 4000, // 4 seconds
    });

    const getLabel = (type: string) => {
        switch (type) {
            case 'todo': return 'Planning';
            case 'goal': return 'Goal Decomposition';
            case 'reflection': return 'Reflection';
            default: return 'Thinking';
        }
    };

    const statusStyles = getWidgetStatusStyles(status, isCompleted);
    
    // Auto-scroll thinking content when streaming
    const thinkingContentRef = useAutoScroll<HTMLDivElement>(
        widget.content,
        isActive,
        isExpanded && isActive
    );

    return (
        <div className={getWidgetContainerClasses(statusStyles, isExpanded, isCompleted)}>
            <div
                className={getWidgetHeaderClasses(statusStyles, isActive)}
                onClick={handleToggle}
            >
                <div className={cn(
                    "relative",
                    statusStyles.iconColor
                )}>
                    {isActive && (
                        <Sparkles 
                            size={12} 
                            className="absolute -top-1 -right-1 animate-pulse opacity-70"
                        />
                    )}
                    <Brain 
                        size={isCompleted ? 14 : 16} 
                        className={cn(
                            "transition-transform duration-200",
                            shouldAnimate && "animate-[badgeLifecycle_2s_ease-in-out_infinite]",
                            isExpanded && !isCompleted && "rotate-12"
                        )}
                    />
                </div>
                
                <span className={cn(
                    "font-medium flex-1 text-sm",
                    statusStyles.textColor
                )}>
                    {getLabel(type)}
                </span>

                <div className="ml-auto flex items-center gap-2">
                    {status === 'active' ? (
                        <Loader2 size={14} className="animate-spin text-blue-400" />
                    ) : (
                        <CheckCircle2 size={14} className="text-green-500 transition-all duration-300" />
                    )}

                    <ChevronDown 
                        size={14} 
                        className={cn(
                            "transition-transform duration-300 text-gray-400",
                            isExpanded ? "rotate-0" : "-rotate-90"
                        )} 
                    />
                </div>
            </div>

            <div className={cn(
                "overflow-hidden transition-all duration-300 ease-in-out",
                isExpanded ? "max-h-[300px] opacity-100" : "max-h-0 opacity-0"
            )}>
                <div 
                    ref={thinkingContentRef}
                    className={cn(
                        "p-3 border-t border-white/10 text-gray-300 overflow-y-auto max-h-[300px] scrollbar-thin scrollbar-thumb-white/20 scrollbar-track-transparent",
                        isCompleted ? "bg-black/10" : "bg-black/30"
                    )}
                >
                    <div className="prose prose-invert prose-sm max-w-none prose-p:leading-relaxed prose-pre:bg-black/50">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                            {widget.content || (isActive ? 'Thinking...' : '')}
                        </ReactMarkdown>
                    </div>
                    {status === 'active' && (
                        <span className="inline-block w-2 h-4 ml-1 bg-blue-400 animate-pulse align-middle" />
                    )}
                </div>
            </div>
        </div>
    );
};
