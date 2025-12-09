import { useEffect, useRef, useCallback, useMemo } from "react";
import type { Session } from "../../types";
import { SCROLL } from "../constants";

/**
 * Shared hook for message list auto-scrolling behavior.
 *
 * Implements terminal-style scroll stickiness:
 * - Auto-scrolls when user is near bottom
 * - Pauses auto-scroll when user scrolls up
 * - Resumes auto-scroll when user returns to bottom
 * - Handles streaming content with MutationObserver
 *
 * @param session - Current session with messages
 * @param isGenerating - Whether agent is currently generating
 * @returns Refs for scroll container and messages end marker
 */
export function useMessageListAutoScroll(
  session: Session | null,
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
    return scrollHeight - scrollTop - clientHeight < SCROLL.THRESHOLD_PX;
  }, []);

  /**
   * Memoized content hash for scroll trigger dependencies.
   * This prevents creating new strings on every render which would cause
   * the useEffect to run on every state change (performance disaster).
   *
   * We track: message count, last message changes, widget count, and generation state.
   * This covers all meaningful content changes without expensive O(n²) computations.
   */
  const contentHash = useMemo(() => {
    if (!session) return null;
    const messages = session.messages;
    const lastMsg = messages[messages.length - 1];

    return {
      messageCount: messages.length,
      lastMessageId: lastMsg?.id,
      lastMessageText: lastMsg?.text?.length ?? 0, // Track length, not content
      lastWidgetCount: lastMsg?.widgets?.length ?? 0,
      lastWidgetStatuses:
        lastMsg?.widgets?.map((w) => w.status).join(",") ?? "",
      isGenerating,
    };
  }, [
    session?.messages.length,
    session?.messages[session?.messages.length - 1]?.id,
    session?.messages[session?.messages.length - 1]?.text?.length,
    session?.messages[session?.messages.length - 1]?.widgets?.length,
    session?.messages[session?.messages.length - 1]?.widgets
      ?.map((w) => w.status)
      .join(","),
    isGenerating,
  ]);

  // Scroll to bottom with smooth behavior
  const scrollToBottom = useCallback(
    (force = false) => {
      if (!messagesEndRef.current || !scrollContainerRef.current) return;

      // Only auto-scroll if user is near bottom or forced
      if (!force && !isNearBottom() && !shouldAutoScrollRef.current) {
        return;
      }

      // Use requestAnimationFrame for smooth scrolling
      requestAnimationFrame(() => {
        messagesEndRef.current?.scrollIntoView({
          behavior: "smooth",
          block: "end",
        });
        shouldAutoScrollRef.current = true;
      });
    },
    [isNearBottom],
  );

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

  // Scroll trigger - uses memoized contentHash to avoid expensive computations
  useEffect(() => {
    if (!contentHash) return;

    // Scroll when content changes
    scrollToBottom(false);

    // Force scroll when generating starts (new content incoming)
    if (contentHash.isGenerating) {
      scrollToBottom(true);
    }
  }, [contentHash, scrollToBottom]);

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
      }, SCROLL.MUTATION_DEBOUNCE_MS);
    });

    observer.observe(scrollContainerRef.current, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["class", "style"], // Track class/style changes (for expand/collapse)
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

  return {
    messagesEndRef,
    scrollContainerRef,
  };
}
