import { useEffect, useRef, useCallback } from "react";
import { SCROLL } from "../constants";

export function useMessageListAutoScroll(
  isGenerating: boolean
) {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const shouldAutoScrollRef = useRef(true);
  const scrollTimeoutRef = useRef<number | null>(null);

  // Check if user is near bottom - if so, auto-scroll
  const isNearBottom = useCallback(() => {
    if (!scrollContainerRef.current) return true;
    const { scrollTop, scrollHeight, clientHeight } =
      scrollContainerRef.current;

    // Increased threshold for better UX
    return scrollHeight - scrollTop - clientHeight < SCROLL.THRESHOLD_PX + 50;
  }, []);

  // Scroll to bottom with smooth behavior
  const scrollToBottom = useCallback(
    (force = false) => {
      if (!messagesEndRef.current || !scrollContainerRef.current) return;

      if (scrollTimeoutRef.current) {
        cancelAnimationFrame(scrollTimeoutRef.current);
      }

      scrollTimeoutRef.current = requestAnimationFrame(() => {
        // If force is true, we ignore user position (e.g. streaming start)
        // If force is false, we check if user is near bottom or stickiness is enabled
        if (!force && !isNearBottom() && !shouldAutoScrollRef.current) {
          return;
        }

        messagesEndRef.current?.scrollIntoView({
          behavior: isGenerating ? "auto" : "smooth",
          block: "end",
        });
        shouldAutoScrollRef.current = true;
      });
    },
    [isNearBottom, isGenerating],
  );

  // Track scroll position to detect manual scrolling
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      if (scrollTimeoutRef.current !== null) {
        window.clearTimeout(scrollTimeoutRef.current);
      }

      if (!isNearBottom()) {
        shouldAutoScrollRef.current = false;
      } else {
        shouldAutoScrollRef.current = true;
      }

      // Reset auto-scroll after delay
      scrollTimeoutRef.current = window.setTimeout(() => {
        if (isNearBottom()) {
          shouldAutoScrollRef.current = true;
        }
      }, SCROLL.RESET_DELAY_MS);
    };

    container.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      container.removeEventListener("scroll", handleScroll);
      if (scrollTimeoutRef.current !== null) {
        window.clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, [isNearBottom]);

  // Use MutationObserver to detect ANY DOM changes (new messages, text growth, widgets)
  useEffect(() => {
    if (!scrollContainerRef.current) return;

    let mutationTimeout: number | null = null;

    const observer = new MutationObserver(() => {
      if (mutationTimeout !== null) {
        window.clearTimeout(mutationTimeout);
      }

      // Debounce slightly to batch layout updates
      mutationTimeout = window.setTimeout(() => {
        // Scroll if sticky logic applies
        requestAnimationFrame(() => scrollToBottom(false));
      }, SCROLL.MUTATION_DEBOUNCE_MS);
    });

    observer.observe(scrollContainerRef.current, {
      childList: true, // New messages
      subtree: true,   // Text updates deep in tree
      attributes: true, // Widget expansions
      characterData: true, // Text node updates
    });

    // Initial scroll on mount/attach
    scrollToBottom(false);

    return () => {
      observer.disconnect();
      if (mutationTimeout !== null) {
        window.clearTimeout(mutationTimeout);
      }
    };
  }, [scrollToBottom]);

  // Force sticky mode when generation starts
  useEffect(() => {
    if (isGenerating) {
      shouldAutoScrollRef.current = true;
      scrollToBottom(true);
    }
  }, [isGenerating, scrollToBottom]);

  return {
    messagesEndRef,
    scrollContainerRef,
  };
}
