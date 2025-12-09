import React from "react";
import { useStore } from "../../store/useStore";
import type { Message } from "../../types";
import { MessageItem } from "./MessageItem";
import { StreamingIndicator } from "../Widgets/StreamingIndicator";
import { ErrorBoundary } from "../ErrorBoundary";

interface MessageListProps {
  messages: Message[];
}

// Helper to check if a message has any visible content
const hasVisibleContent = (message: Message): boolean => {
  if (message.text) return true;
  if (message.widgets && message.widgets.length > 0) return true;
  return false;
};

export const MessageList: React.FC<MessageListProps> = ({ messages }) => {
  // Use selector for better performance - only subscribe to isGenerating
  const isGenerating = useStore((state) => state.isGenerating);

  // Filter out empty agent messages (placeholders that haven't received content yet)
  const visibleMessages = messages.filter(
    (msg) => msg.role !== "agent" || hasVisibleContent(msg),
  );

  return (
    <div className="flex flex-col gap-6 max-w-[760px] mx-auto w-full">
      {visibleMessages.map((message, index) => (
        <ErrorBoundary key={message.id}>
          <MessageItem
            message={message}
            messageIndex={index}
            isLastMessage={index === visibleMessages.length - 1}
          />
        </ErrorBoundary>
      ))}

      {/* Show streaming indicator while generating */}
      {isGenerating && <StreamingIndicator />}
    </div>
  );
};
