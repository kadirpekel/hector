import React from "react";
import { MessageList } from "./MessageList";
import { InputArea } from "./InputArea";
import { useStore } from "../../store/useStore";
import { useMessageListAutoScroll } from "../../lib/hooks/useMessageListAutoScroll";

interface ChatAreaProps {
  onNavigateToBuilder?: () => void; // Kept for backward compatibility but not used in Studio Mode
}

export const ChatArea: React.FC<ChatAreaProps> = ({ onNavigateToBuilder }) => {
  // Use selectors for better performance - only subscribe to specific state slices
  const currentSessionId = useStore((state) => state.currentSessionId);
  const sessions = useStore((state) => state.sessions);
  const isGenerating = useStore((state) => state.isGenerating);
  const session = currentSessionId ? sessions[currentSessionId] : null;

  // Use shared auto-scroll hook
  const { messagesEndRef, scrollContainerRef } = useMessageListAutoScroll(
    session,
    isGenerating,
  );

  if (!session) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-gray-500">
        {onNavigateToBuilder && (
          <div className="text-center max-w-md mb-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-3">
              Welcome to Hector! 👋
            </h2>
            <p className="text-gray-600 mb-6">
              Start chatting with your agent or build a custom configuration
            </p>

            <button
              onClick={onNavigateToBuilder}
              className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              🛠️ Build Your Agent Configuration
            </button>

            <p className="text-sm text-gray-500 mt-4">
              Or select a chat to begin
            </p>
          </div>
        )}
        {!onNavigateToBuilder && <span>Select or create a chat to begin</span>}
      </div>
    );
  }

  const hasMessages = session.messages.length > 0;

  return (
    <div className="flex flex-col h-full w-full relative">
      {hasMessages ? (
        <>
          <div
            ref={scrollContainerRef}
            className="flex-1 overflow-y-auto custom-scrollbar p-4 md:p-6 pb-32"
          >
            <MessageList messages={session.messages} />
            <div ref={messagesEndRef} />
          </div>

          <div className="sticky bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black via-black/90 to-transparent pt-10 z-50">
            <div className="max-w-[760px] mx-auto w-full">
              <InputArea />
            </div>
          </div>
        </>
      ) : (
        <>
          {onNavigateToBuilder && (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center max-w-md mb-8">
                <h2 className="text-3xl font-bold text-gray-900 mb-3">
                  Welcome to Hector! 👋
                </h2>
                <p className="text-gray-600 mb-6">
                  Start chatting with your agent or build a custom configuration
                </p>

                <button
                  onClick={onNavigateToBuilder}
                  className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
                >
                  🛠️ Build Your Agent Configuration
                </button>

                <p className="text-sm text-gray-500 mt-4">
                  Or just start typing below to chat with the default agent
                </p>
              </div>
            </div>
          )}

          <div className="sticky bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black via-black/90 to-transparent pt-10 z-50">
            <div className="max-w-[760px] mx-auto w-full">
              <InputArea />
            </div>
          </div>
        </>
      )}
    </div>
  );
};
