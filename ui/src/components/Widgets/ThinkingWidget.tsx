import React, { useState, useEffect } from 'react';
import { Brain, ChevronDown, ChevronRight, CheckCircle2, Loader2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Widget } from '../../types';
import { cn } from '../../lib/utils';

interface ThinkingWidgetProps {
    widget: Widget;
    onExpansionChange?: (expanded: boolean) => void;
    shouldAnimate?: boolean;
}

export const ThinkingWidget: React.FC<ThinkingWidgetProps> = ({ widget, onExpansionChange, shouldAnimate = false }) => {
    // Widget expansion state: read from prop, sync changes via callback
    const [isExpanded, setIsExpanded] = useState(widget.isExpanded ?? false);
    const { type } = widget.data;
    const status = widget.status;

    // Sync local state when widget prop changes (e.g., from store updates)
    useEffect(() => {
        if (widget.isExpanded !== undefined && widget.isExpanded !== isExpanded) {
            setIsExpanded(widget.isExpanded);
        }
    }, [widget.isExpanded, isExpanded]);

    // Sync local expansion state to store on unmount (handles edge case where local state
    // changes but user navigates away before toggling, e.g., auto-expand scenarios)
    useEffect(() => {
        return () => {
            if (widget.isExpanded !== isExpanded) {
                onExpansionChange?.(isExpanded);
            }
        };
    }, [isExpanded, widget.isExpanded, onExpansionChange]);

    const getLabel = (type: string) => {
        switch (type) {
            case 'todo': return 'Planning';
            case 'goal': return 'Goal Decomposition';
            case 'reflection': return 'Reflection';
            default: return 'Thinking';
        }
    };

    const handleToggle = () => {
        const newExpanded = !isExpanded;
        setIsExpanded(newExpanded);
        onExpansionChange?.(newExpanded);
    };

    return (
        <div className="border border-white/10 rounded-lg bg-black/20 overflow-hidden text-sm">
            <div
                className="flex items-center gap-2 p-2 bg-white/5 cursor-pointer hover:bg-white/10 transition-colors"
                onClick={handleToggle}
            >
                <Brain 
                    size={14} 
                    className={cn(
                        "text-blue-400",
                        shouldAnimate && "animate-[badgeLifecycle_2s_ease-in-out_infinite]"
                    )}
                />
                <span className="font-medium text-blue-200">{getLabel(type)}</span>

                <div className="ml-auto flex items-center gap-2">
                    {status === 'active' ? (
                        <Loader2 size={14} className="animate-spin text-blue-400" />
                    ) : (
                        <CheckCircle2 size={14} className="text-green-500" />
                    )}

                    {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                </div>
            </div>

            {isExpanded && (
                <div className="p-3 border-t border-white/10 text-gray-300 bg-black/40">
                    <div className="prose prose-invert prose-sm max-w-none prose-p:leading-relaxed prose-pre:bg-black/50">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>
                            {widget.content || ''}
                        </ReactMarkdown>
                    </div>
                    {status === 'active' && (
                        <span className="inline-block w-2 h-4 ml-1 bg-blue-400 animate-pulse align-middle" />
                    )}
                </div>
            )}
        </div>
    );
};
