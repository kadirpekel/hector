import React, { useState } from 'react';
import { Image as ImageIcon, ChevronDown, ChevronRight, Download, Maximize2 } from 'lucide-react';
import type { Widget } from '../../types';

interface ImageWidgetProps {
    widget: Widget;
    onExpansionChange?: (expanded: boolean) => void;
    shouldAnimate?: boolean;
}

export const ImageWidget: React.FC<ImageWidgetProps> = ({ widget, onExpansionChange }) => {
    const [isExpanded, setIsExpanded] = useState(widget.isExpanded);
    const { url, revised_prompt } = widget.data;

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
                <ImageIcon size={14} className="text-pink-400" />
                <span className="font-medium text-pink-200">Generated Image</span>

                <div className="ml-auto flex items-center gap-2">
                    {isExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                </div>
            </div>

            {isExpanded && (
                <div className="p-3 border-t border-white/10 space-y-3">
                    <div className="relative group rounded-lg overflow-hidden border border-white/10 bg-black/50">
                        <img
                            src={url}
                            alt={revised_prompt || "Generated image"}
                            className="w-full h-auto max-h-[400px] object-contain"
                        />

                        <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-4">
                            <a
                                href={url}
                                download={`generated-image.png`}
                                className="p-2 bg-white/10 hover:bg-white/20 rounded-full text-white transition-colors"
                                title="Download"
                                onClick={(e) => e.stopPropagation()}
                            >
                                <Download size={20} />
                            </a>
                            <button
                                className="p-2 bg-white/10 hover:bg-white/20 rounded-full text-white transition-colors"
                                title="View Fullscreen"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    window.open(url, '_blank');
                                }}
                            >
                                <Maximize2 size={20} />
                            </button>
                        </div>
                    </div>

                    {revised_prompt && (
                        <div className="text-xs text-gray-400 italic">
                            Prompt: {revised_prompt}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};
