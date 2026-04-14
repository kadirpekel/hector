import { memo } from "react";
import {
    BrainCircuit,
    ChevronDown,
    CheckCircle2,
    XCircle,
    Loader2,
    Sparkles,
} from "lucide-react";
import type { ToolWidget as ToolWidgetType } from "../../types";
import { cn } from "../../lib/utils";
import { useWidgetExpansion } from "./useWidgetExpansion";
import {
    getWidgetStatusStyles,
    getWidgetContainerClasses,
    getWidgetHeaderClasses,
} from "./widgetStyles";
import { useAutoScroll } from "./useAutoScroll";

interface MemoryWidgetProps {
    widget: ToolWidgetType;
    onExpansionChange?: (expanded: boolean) => void;
    shouldAnimate?: boolean;
}

/**
 * MemoryWidget displays a specialized UI for the memory_search tool.
 * It shows "Remembering..." when working, and a summary of what was found.
 */
export const MemoryWidget = memo<MemoryWidgetProps>(function MemoryWidget({
    widget,
    onExpansionChange,
    shouldAnimate = false,
}) {
    const { args } = widget.data;
    const status = widget.status;
    const query = (args as any)?.query || "memories";

    // Use shared expansion hook
    // Default to collapsed unless manually expanded
    const { isExpanded, isActive, isCompleted, handleToggle } =
        useWidgetExpansion({
            widget,
            onExpansionChange,
            autoExpandWhenActive: false, // Don't auto-expand memory searches, they are background tasks mostly
            activeStatuses: ["working"],
            completedStatuses: ["success", "failed"],
            collapseDelay: 0,
        });

    const statusStyles = getWidgetStatusStyles(status, isCompleted);

    // Auto-scroll result content
    const resultContentRef = useAutoScroll<HTMLPreElement>(
        widget.content,
        isActive && !!widget.content,
        isExpanded && isActive,
    );

    // Parse results to show a summary count
    let resultCount = 0;
    if (status === "success" && widget.content) {
        try {
            const result = JSON.parse(widget.content);
            if (result.results && Array.isArray(result.results)) {
                resultCount = result.results.length;
            }
        } catch (e) {
            // Ignore parse errors
        }
    }

    return (
        <div
            className={getWidgetContainerClasses(
                statusStyles,
                isExpanded,
                isCompleted,
            )}
            role="region"
            aria-label={`Memory Search: ${query}`}
        >
            <div
                className={getWidgetHeaderClasses(statusStyles, isActive)}
                onClick={handleToggle}
                onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        handleToggle();
                    }
                }}
                role="button"
                tabIndex={0}
                aria-expanded={isExpanded}
                aria-label={`Toggle memory search details`}
            >
                <div className={cn("relative", statusStyles.iconColor)}>
                    {isActive && (
                        <Sparkles
                            size={12}
                            className="absolute -top-1 -right-1 animate-pulse opacity-70"
                        />
                    )}
                    <BrainCircuit
                        size={isCompleted ? 14 : 16}
                        className={cn(
                            "transition-transform duration-200",
                            shouldAnimate &&
                            "animate-[badgeLifecycle_2s_ease-in-out_infinite]",
                        )}
                    />
                </div>

                <span
                    className={cn("font-medium flex-1 text-sm flex items-center gap-2", statusStyles.textColor)}
                >
                    {status === "working" ? (
                        <span className="animate-pulse">Remembering...</span>
                    ) : (
                        <span>Memory Search</span>
                    )}

                    <span className="opacity-60 text-xs font-normal truncate max-w-[200px]">
                        "{query}"
                    </span>

                    {status === "success" && resultCount > 0 && (
                        <span className="text-[10px] bg-white/10 px-1.5 py-0.5 rounded-full opacity-80">
                            {resultCount} found
                        </span>
                    )}
                </span>

                <div className="ml-auto flex items-center gap-2">
                    {status === "working" && (
                        <Loader2 size={14} className="animate-spin text-yellow-400" />
                    )}
                    {status === "success" && (
                        <CheckCircle2
                            size={14}
                            className="text-green-500 transition-all duration-300"
                        />
                    )}
                    {status === "failed" && (
                        <XCircle
                            size={14}
                            className="text-red-500 transition-all duration-300"
                        />
                    )}

                    <ChevronDown
                        size={14}
                        className={cn(
                            "transition-transform duration-300 text-gray-400",
                            isExpanded ? "rotate-0" : "-rotate-90",
                        )}
                    />
                </div>
            </div>

            <div
                className={cn(
                    "overflow-hidden transition-all duration-300 ease-in-out",
                    isExpanded ? "max-h-[400px] opacity-100" : "max-h-0 opacity-0",
                )}
            >
                <div
                    className={cn(
                        "p-3 space-y-2 border-t border-white/10",
                        isCompleted ? "bg-black/10" : "bg-black/30",
                    )}
                >
                    {/* Input */}
                    <div className="space-y-2">
                        <div className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
                            Query
                        </div>
                        <pre
                            className={cn(
                                "bg-black/60 p-3 rounded-lg overflow-x-auto text-xs max-h-[80px] overflow-y-auto",
                                "border border-white/10",
                                "text-gray-300 font-mono leading-relaxed",
                                "scrollbar-thin scrollbar-thumb-white/20 scrollbar-track-transparent",
                            )}
                        >
                            {JSON.stringify(args, null, 2)}
                        </pre>
                    </div>

                    {/* Output */}
                    {(widget.content || (isActive && status === "working")) && (
                        <div className="space-y-2">
                            <div className="flex items-center gap-2">
                                <div className="text-xs font-semibold text-gray-400 uppercase tracking-wider">
                                    Result
                                </div>
                            </div>
                            <pre
                                ref={resultContentRef}
                                className={cn(
                                    "bg-black/60 p-3 rounded-lg overflow-x-auto text-xs max-h-[200px] overflow-y-auto",
                                    "border border-white/10",
                                    "font-mono leading-relaxed",
                                    "scrollbar-thin scrollbar-thumb-white/20 scrollbar-track-transparent",
                                    status === "failed"
                                        ? "text-red-300 border-red-500/20"
                                        : status === "working"
                                            ? "text-yellow-300 border-yellow-500/20"
                                            : "text-green-300 border-green-500/20",
                                )}
                            >
                                {widget.content ||
                                    (status === "working" ? "Searching memory..." : "")}
                            </pre>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
});
