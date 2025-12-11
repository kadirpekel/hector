import React from "react";
import { MessageList } from "./MessageList";
import { InputArea } from "./InputArea";
import { useStore } from "../../store/useStore";
import { AlertCircle, X } from "lucide-react";

interface ChatAreaProps {
  onNavigateToBuilder?: () => void; // Kept for backward compatibility but not used in Studio Mode
}

const ErrorBanner = () => {
  const error = useStore((state) => state.error);
  const setError = useStore((state) => state.setError);

  if (!error) return null;

  return (
    <div className="absolute top-4 left-4 right-4 z-[60] animate-in fade-in slide-in-from-top-2">
      <div className="mx-auto max-w-[760px] bg-red-500/10 border border-red-500/50 text-red-200 px-4 py-3 rounded-lg shadow-lg flex items-start gap-3 backdrop-blur-md">
        <AlertCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
        <div className="flex-1 text-sm">{error}</div>
        <button
          onClick={() => setError(null)}
          className="text-red-400 hover:text-red-300 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>
      </div>
    </div>
  );
};

export const ChatArea: React.FC<ChatAreaProps> = ({ onNavigateToBuilder }) => {
  console.log("RENDER: ChatArea");

  const currentSessionId = useStore((state) => state.currentSessionId);
  // PERFORMANCE: Check session existence without accessing full session object
  const sessionExists = useStore((state) =>
    !!(state.currentSessionId && state.currentSessionId in state.sessions)
  );
  // PERFORMANCE: Select only message count to avoid re-rendering on text updates
  const msgCount = useStore((state) => {
    if (!state.currentSessionId) return 0;
    const session = state.sessions[state.currentSessionId];
    return session?.messages.length || 0;
  });

  const hasMessages = msgCount > 0;

  if (!sessionExists || !currentSessionId) {
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

  return (
    <div className="flex flex-col h-full w-full relative">
      <ErrorBanner />
      {hasMessages ? (
        <>
          {/* Scroll container is handled inside MessageList now */}
          <MessageList sessionId={currentSessionId} />

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
