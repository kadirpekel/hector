import React, { useEffect, useRef, useCallback } from 'react';
import { MessageList } from './MessageList';
import { InputArea } from './InputArea';
import { useStore } from '../../store/useStore';

export const ChatArea: React.FC = () => {
    const { currentSessionId, sessions, isGenerating } = useStore();
    const session = currentSessionId ? sessions[currentSessionId] : null;
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const shouldAutoScrollRef = useRef(true);
    const scrollTimeoutRef = useRef<number | null>(null);

    // Check if user is near bottom (within 100px) - if so, auto-scroll
    const isNearBottom = useCallback(() => {
        if (!scrollContainerRef.current) return true;
        const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
        const threshold = 100; // pixels from bottom
        return scrollHeight - scrollTop - clientHeight < threshold;
    }, []);

    // Scroll to bottom with smooth behavior
    const scrollToBottom = useCallback((force = false) => {
        if (!messagesEndRef.current || !scrollContainerRef.current) return;
        
        // Only auto-scroll if user is near bottom or forced
        if (!force && !isNearBottom() && !shouldAutoScrollRef.current) {
            return;
        }

        // Use requestAnimationFrame for smooth scrolling
        requestAnimationFrame(() => {
            messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
            shouldAutoScrollRef.current = true;
        });
    }, [isNearBottom]);

    // Track scroll position to detect manual scrolling
    useEffect(() => {
        const container = scrollContainerRef.current;
        if (!container) return;

        const handleScroll = () => {
            // Clear any pending scroll timeout
            if (scrollTimeoutRef.current !== null) {
                window.clearTimeout(scrollTimeoutRef.current);
            }

            // If user scrolls up, disable auto-scroll temporarily
            if (!isNearBottom()) {
                shouldAutoScrollRef.current = false;
            } else {
                // Re-enable auto-scroll if user scrolls back to bottom
                shouldAutoScrollRef.current = true;
            }

            // Reset auto-scroll after a delay of no scrolling
            scrollTimeoutRef.current = window.setTimeout(() => {
                if (isNearBottom()) {
                    shouldAutoScrollRef.current = true;
                }
            }, 1000);
        };

        container.addEventListener('scroll', handleScroll, { passive: true });
        return () => {
            container.removeEventListener('scroll', handleScroll);
            if (scrollTimeoutRef.current !== null) {
                window.clearTimeout(scrollTimeoutRef.current);
            }
        };
    }, [isNearBottom]);

    // Comprehensive scroll trigger - tracks all content changes
    useEffect(() => {
        if (!session) return;

        // Scroll when content changes
        scrollToBottom(false);

        // Also scroll when generating starts (new content incoming)
        if (isGenerating) {
            scrollToBottom(true);
        }
    }, [
        session?.messages.length,
        session?.messages.map(m => m.id).join(','),
        session?.messages.map(m => m.text).join('||'),
        session?.messages.map(m => 
            m.widgets?.map(w => `${w.id}:${w.status}:${w.content}`).join('|') || ''
        ).join('||'),
        session?.messages.map(m => 
            m.metadata?.contentOrder?.join(',') || ''
        ).join('||'),
        session?.messages.map(m => 
            m.widgets?.map(w => w.isExpanded).join(',') || ''
        ).join('||'),
        isGenerating,
        scrollToBottom
    ]);

    // Use MutationObserver to detect DOM changes (widget expansions, content updates, etc.)
    useEffect(() => {
        if (!scrollContainerRef.current) return;

        let mutationTimeout: number | null = null;

        const observer = new MutationObserver(() => {
            // Debounce mutations to avoid excessive scrolling
            if (mutationTimeout !== null) {
                window.clearTimeout(mutationTimeout);
            }

            mutationTimeout = window.setTimeout(() => {
                // Only scroll if user is near bottom or actively generating
                if (shouldAutoScrollRef.current || isGenerating) {
                    requestAnimationFrame(() => {
                        scrollToBottom(false);
                    });
                }
            }, 50); // 50ms debounce - smooth but responsive
        });

        observer.observe(scrollContainerRef.current, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['class', 'style'], // Track class/style changes (for expand/collapse)
            characterData: true, // Track text content changes
        });

        return () => {
            observer.disconnect();
            if (mutationTimeout !== null) {
                window.clearTimeout(mutationTimeout);
            }
        };
    }, [isGenerating, scrollToBottom]);

    // Force scroll when generation starts
    useEffect(() => {
        if (isGenerating) {
            shouldAutoScrollRef.current = true;
            scrollToBottom(true);
        }
    }, [isGenerating, scrollToBottom]);

    if (!session) {
        return (
            <div className="flex-1 flex items-center justify-center text-gray-500">
                Select or create a chat to begin
            </div>
        );
    }

    return (
        <div className="flex flex-col h-full w-full relative">
            <div 
                ref={scrollContainerRef}
                className="flex-1 overflow-y-auto custom-scrollbar p-4 md:p-6 pb-32"
            >
                <MessageList messages={session.messages} />
                <div ref={messagesEndRef} />
            </div>

            <div className="sticky bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black via-black/90 to-transparent pt-10 z-10">
                <div className="max-w-[760px] mx-auto w-full">
                    <InputArea />
                </div>
            </div>
        </div>
    );
};
