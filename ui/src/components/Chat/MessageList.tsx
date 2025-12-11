import React from "react";
import { useStore } from "../../store/useStore";
import { useShallow } from "zustand/react/shallow";
import { MessageItem } from "./MessageItem";
import { StreamingIndicator } from "../Widgets/StreamingIndicator";
import { ErrorBoundary } from "../ErrorBoundary";
import { useMessageListAutoScroll } from "../../lib/hooks/useMessageListAutoScroll";

interface MessageListProps {
  sessionId: string;
}

export const MessageList: React.FC<MessageListProps> = ({ sessionId }) => {
  // 1. Subscribe to Message IDs only (Atomic Structure)
  // useShallow ensures we only re-render if the IDs array content changes
  const messageIds = useStore(useShallow((state) =>
    state.sessions[sessionId]?.messages.map((m) => m.id) || []
  ));

  const isGenerating = useStore((state) => state.isGenerating);
  const currentSessionId = useStore((state) => state.currentSessionId);

  // Optimizing hook usage: we removed the session dependency from the hook
  // It now relies on MutationObserver, which is safer and decoupled.
  const { messagesEndRef, scrollContainerRef } = useMessageListAutoScroll(
    isGenerating
  );

  return (
    <div
      ref={scrollContainerRef}
      className="flex-1 overflow-y-auto px-4 py-4 space-y-4 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent custom-scrollbar"
    >
      <div className="flex flex-col gap-6 max-w-[760px] mx-auto w-full">
        {(messageIds as string[]).map((id, index) => (
          <ErrorBoundary key={id}>
            <MessageItem
              messageId={id}
              messageIndex={index}
              isLastMessage={index === messageIds.length - 1}
              isGenerating={isGenerating}
              currentSessionId={currentSessionId}
            />
          </ErrorBoundary>
        ))}

        {isGenerating && <StreamingIndicator />}

        <div ref={messagesEndRef} className="h-4" />
      </div>
    </div>
  );
};
