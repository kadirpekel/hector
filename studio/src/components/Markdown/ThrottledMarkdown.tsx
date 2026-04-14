import React, { useState, useEffect, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import "highlight.js/styles/github-dark.css";

interface ThrottledMarkdownProps {
    content: string;
    components?: any;
}

/**
 * ThrottledMarkdown Component
 * 
 * Renders Markdown with syntax highlighting.
 * Uses a heuristic to apply throttling only when expensive rendering (Highlight.js) is required.
 * 
 * Logic:
 * - Has Code Blocks (```): Aggressive throttling to prevent Main Thread blocking.
 * - Plain Text: Fast updates (20fps) for fluid experience.
 */
export const ThrottledMarkdown = React.memo(({ content, components }: ThrottledMarkdownProps) => {
    const [renderedContent, setRenderedContent] = useState(content);
    const lastUpdate = useRef(0);
    const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    useEffect(() => {
        const now = Date.now();
        const timeSinceLast = now - lastUpdate.current;

        // Heuristic: Check for code blocks
        // We only need aggressive throttling if there is complex syntax highlighting to do
        const hasCodeBlock = content.includes('```');

        let interval = 50; // Default: Fast updates (20fps)

        if (hasCodeBlock) {
            // Aggressive throttling for code blocks (High CPU cost due to highlight.js)
            if (content.length > 5000) interval = 1000;      // 1s for >5k chars
            else if (content.length > 1000) interval = 500;  // 500ms for >1k chars
            else if (content.length > 500) interval = 250;   // 250ms for >500 chars
            else interval = 100;                             // 100ms for small code
        } else {
            // Mild throttling for large plain text (Virtual DOM cost)
            if (content.length > 10000) interval = 200;
            else if (content.length > 2000) interval = 100;
            // < 2000 chars: 50ms (Fluid)
        }

        // Determine if we should update immediately or schedule a delayed update
        if (timeSinceLast >= interval) {
            // Use setTimeout to avoid synchronous state update warning and ensure effect cleanup runs first
            setTimeout(() => {
                setRenderedContent(content);
                lastUpdate.current = Date.now();
            }, 0);
        } else {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            timeoutRef.current = setTimeout(() => {
                setRenderedContent(content);
                lastUpdate.current = Date.now();
            }, interval - timeSinceLast);
        }

        return () => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        };
    }, [content]);

    return (
        <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeHighlight]}
            components={components}
        >
            {renderedContent}
        </ReactMarkdown>
    );
}, (prev, next) => {
    // Custom comparison to avoid unnecessary renders if content hasn't changed
    return prev.content === next.content && prev.components === next.components;
});
