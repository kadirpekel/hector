import { useEffect, useRef } from 'react';

/**
 * Hook to auto-scroll a content area to bottom when content changes
 * Useful for streaming content in widgets
 */
export function useAutoScroll<T extends HTMLElement>(
    content: string | undefined,
    isActive: boolean,
    enabled: boolean = true
) {
    const contentRef = useRef<T>(null);
    const prevContentLengthRef = useRef(0);
    const hasScrolledRef = useRef(false);

    useEffect(() => {
        if (!enabled || !contentRef.current) return;

        const currentLength = content?.length || 0;
        const isNewContent = currentLength > prevContentLengthRef.current;
        const isFirstContent = currentLength > 0 && prevContentLengthRef.current === 0;
        prevContentLengthRef.current = currentLength;

        // Scroll if:
        // 1. New content was added (streaming)
        // 2. First content appeared (initial render)
        // 3. Widget is active (content is being streamed)
        if ((isNewContent || isFirstContent) && isActive) {
            // Use requestAnimationFrame for smooth scrolling
            requestAnimationFrame(() => {
                if (contentRef.current) {
                    contentRef.current.scrollTop = contentRef.current.scrollHeight;
                    hasScrolledRef.current = true;
                }
            });
        }

        // Reset scroll tracking when content becomes inactive
        if (!isActive) {
            hasScrolledRef.current = false;
        }
    }, [content, isActive, enabled]);

    return contentRef;
}

