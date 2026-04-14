import React from "react";
import { MessageList } from "./MessageList";
import { InputArea } from "./InputArea";
import { useStore } from "../../store/useStore";
import { AlertCircle, X, MessageSquare, Sparkles, Zap } from "lucide-react";
import { SessionHistoryPanel } from "./SessionHistoryPanel";
import { useSessionPolling } from "../../lib/hooks/useSessionPolling";

interface ChatAreaProps {
  isHistoryOpen?: boolean;
  isHistoryPinned?: boolean;
  onTogglePin?: () => void;
  onCloseHistory?: () => void;
}

const ErrorBanner = () => {
  const error = useStore((state) => state.error);
  const setError = useStore((state) => state.setError);

  if (!error) return null;

  return (
    <div className="absolute top-16 left-4 right-4 z-[60] animate-in fade-in slide-in-from-top-2">
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

/**
 * Welcome pane shown when a chat session has no messages yet.
 */
const WelcomePane: React.FC = () => {
  const selectedAgent = useStore((state) => state.selectedAgent);
  const agentName = selectedAgent?.name || "Agent";

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="text-center max-w-md">
        {/* Animated gradient orb */}
        <div className="relative w-20 h-20 mx-auto mb-6">
          <div className="absolute inset-0 rounded-full bg-gradient-to-br from-hector-green/30 to-blue-500/20 blur-xl animate-pulse" />
          <div className="relative w-full h-full rounded-full bg-gradient-to-br from-hector-green/20 to-blue-500/10 border border-white/10 flex items-center justify-center backdrop-blur-sm">
            <MessageSquare className="w-8 h-8 text-hector-green" />
          </div>
        </div>

        {/* Greeting */}
        <h2 className="text-xl font-medium text-gray-100 mb-2">
          Chat with {agentName}
        </h2>
        <p className="text-sm text-gray-500 mb-8">
          Start a conversation by typing a message below.
        </p>

        {/* Quick suggestions */}
        <div className="flex flex-wrap justify-center gap-2">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10 text-xs text-gray-400">
            <Sparkles size={12} className="text-amber-400" />
            <span>Ask a question</span>
          </div>
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10 text-xs text-gray-400">
            <Zap size={12} className="text-blue-400" />
            <span>Get help with a task</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export const ChatArea: React.FC<ChatAreaProps> = ({
  isHistoryOpen,
  isHistoryPinned,
  onTogglePin,
  onCloseHistory
}) => {
  // Global state
  const currentSessionId = useStore((state) => state.currentSessionId);
  
  // Poll for session updates when a session is selected (enables sync with headless updates)
  useSessionPolling();

  // PERFORMANCE: Check session existence without accessing full session object
  const sessionExists = useStore((state) =>
    !!(state.currentSessionId && state.currentSessionId in state.sessions)
  );

  // Session lifecycle is now managed by App.tsx on server switch
  // ChatArea only renders the current session

  // PERFORMANCE: Select only message count to avoid re-rendering on text updates
  const msgCount = useStore((state) => {
    if (!state.currentSessionId) return 0;
    const session = state.sessions[state.currentSessionId];
    return session?.messages.length || 0;
  });

  // If session is missing, show empty placeholder (App.tsx manages session creation)
  if (!sessionExists) {
    return <div className="flex-1 bg-gray-900/20" />;
  }

  const hasMessages = msgCount > 0;

  return (
    <div className="flex flex-row h-full w-full relative bg-gray-900/20">
      {/* Main Chat Column */}
      <div className="flex flex-col flex-1 h-full min-w-0 relative">
        <ErrorBanner />

        {/* Floating History Popover (Right Side) */}
        {!isHistoryPinned && isHistoryOpen && (
          <div className="absolute top-4 right-4 z-50 animate-in fade-in zoom-in-95 duration-200">
            <SessionHistoryPanel
              isPinned={false}
              onTogglePin={onTogglePin || (() => { })}
              onClose={onCloseHistory || (() => { })}
            />
            {/* Backdrop to close on click outside */}
            <div
              className="fixed inset-0 z-[-1]"
              onClick={onCloseHistory}
            />
          </div>
        )}

        {/* Message List or Welcome Pane */}
        {hasMessages ? (
          <MessageList sessionId={currentSessionId!} />
        ) : (
          <WelcomePane />
        )}

        {/* Input Area (Always Visible) */}
        <div className="sticky bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black via-black/90 to-transparent pt-10 z-30 pointer-events-none">
          <div className="max-w-[760px] mx-auto w-full pointer-events-auto">
            <InputArea />
          </div>
        </div>
      </div>

      {/* Pinned History Panel (Right Side) */}
      {isHistoryPinned && (
        <SessionHistoryPanel
          isPinned={true}
          onTogglePin={onTogglePin || (() => { })}
          onClose={onCloseHistory || (() => { })}
        />
      )}
    </div>
  );
};
