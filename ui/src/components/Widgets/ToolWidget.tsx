import React, { useState, useEffect } from 'react';
import { Wrench, ChevronDown, ChevronRight, CheckCircle2, XCircle, Loader2 } from 'lucide-react';
import type { Widget } from '../../types';
import { cn } from '../../lib/utils';

interface ToolWidgetProps {
    widget: Widget;
    onExpansionChange?: (expanded: boolean) => void;
    shouldAnimate?: boolean;
}

export const ToolWidget: React.FC<ToolWidgetProps> = ({ widget, onExpansionChange, shouldAnimate = false }) => {
    // Widget expansion state: read from prop, sync changes via callback
    const [isExpanded, setIsExpanded] = useState(widget.isExpanded ?? false);
    const { name, args } = widget.data;
    const status = widget.status;

    // Sync local state when widget prop changes (e.g., from store updates)
    useEffect(() => {
        if (widget.isExpanded !== undefined && widget.isExpanded !== isExpanded) {
            setIsExpanded(widget.isExpanded);
        }
    }, [widget.isExpanded, isExpanded]);

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
                <Wrench 
                    size={14} 
                    className={cn(
                        "text-purple-400",
                        shouldAnimate && "animate-[badgeLifecycle_2s_ease-in-out_infinite]"
                    )}
                />
                <span className="font-medium text-purple-200">Tool: {name}</span>

                <div className="ml-auto flex items-center gap-2">
                    {status === 'working' && <Loader2 size={14} className="animate-spin text-yellow-500" />}
                    {status === 'success' && <CheckCircle2 size={14} className="text-green-500" />}
                    {status === 'failed' && <XCircle size={14} className="text-red-500" />}

                    {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                </div>
            </div>

            {isExpanded && (
                <div className="p-3 space-y-2 border-t border-white/10 font-mono text-xs">
                    {/* Input */}
                    <div>
                        <div className="text-gray-500 mb-1">Input:</div>
                        <pre className="bg-black/40 p-2 rounded overflow-x-auto text-gray-300">
                            {JSON.stringify(args, null, 2)}
                        </pre>
                    </div>

                    {/* Output */}
                    {widget.content && (
                        <div>
                            <div className="text-gray-500 mb-1">Result:</div>
                            <pre className={cn(
                                "bg-black/40 p-2 rounded overflow-x-auto",
                                status === 'failed' ? "text-red-300" : "text-green-300"
                            )}>
                                {widget.content}
                            </pre>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};
