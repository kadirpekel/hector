import React, {
  useEffect,
  useState,
  useRef,
} from "react";
import {
  X,
  Pin,
  Maximize2,
  Minimize2,
  Square,
  ChevronDown,
  Trash2,
} from "lucide-react";
import { useStore } from "../../store/useStore";
import { MessageList } from "../Chat/MessageList";
import { InputArea } from "../Chat/InputArea";
import { useAgentSelection } from "../../lib/hooks/useAgentSelection";

type ChatWidgetState = "closed" | "popup" | "expanded" | "maximized" | "pane";

interface ChatWidgetProps {
  state: ChatWidgetState;
  onStateChange: (state: ChatWidgetState) => void;
  isPinned: boolean;
  onPinChange: (pinned: boolean) => void;
  onMessageSent?: () => void; // New callback for auto-layout switching
}

export const ChatWidget: React.FC<ChatWidgetProps> = React.memo(({
  state,
  onStateChange,
  isPinned,
  onPinChange,
  onMessageSent,
}) => {

  // Store
  // Removed direct session subscription to prevent entire widget re-render on every token
  const currentSessionId = useStore((state) => state.currentSessionId);
  // PERFORMANCE: Check session existence without accessing full session object
  // This prevents re-renders during streaming
  const sessionExists = useStore((state) =>
    !!(state.currentSessionId && state.currentSessionId in state.sessions)
  );

  const msgCount = useStore((state) => {
    if (!state.currentSessionId) return 0;
    const session = state.sessions[state.currentSessionId];
    return session?.messages.length || 0;
  });

  const availableAgents = useStore((state) => state.availableAgents);
  const agentsLoaded = useStore((state) => state.agentsLoaded);
  const selectedAgent = useStore((state) => state.selectedAgent);
  const setSelectedAgent = useStore((state) => state.setSelectedAgent);
  const createSession = useStore((state) => state.createSession);

  const { handleAgentChange, fetchAgentCardSafe } = useAgentSelection();

  // No message tracking needed here anymore


  /**
   * Agent Selection State Machine
   * ==============================
   *
   * This effect handles initial agent selection with the following priority:
   * 1. If an agent is already selected in store → Keep it (prevents unwanted switching)
   * 2. If persisted agent exists and is available → Restore it
   * 3. Otherwise → Select first available agent
   *
   * Runs once when agents finish loading.
   * After initial selection, only user actions can change the selected agent.
   */
  const initialSelectionDoneRef = useRef(false);

  useEffect(() => {
    // Wait for agents to load
    if (!agentsLoaded || availableAgents.length === 0) {
      return;
    }

    // Special case: If initialized but selectedAgent is null, allow re-selection
    // This handles the case where agent was removed during config reload
    const needsReselection = initialSelectionDoneRef.current && !selectedAgent;

    // Skip if already initialized AND agent is still selected
    // This ensures we don't override manual selection on config reload
    if (initialSelectionDoneRef.current && !needsReselection) {
      return;
    }

    // Mark as initialized to prevent double-execution
    initialSelectionDoneRef.current = true;

    // Priority 1: Trust existing selection (prevents implicit switching)
    // Note: loadAgents() in useStore handles restoration from localStorage
    // So if selectedAgent exists here, it's either freshly selected or restored from persistence
    if (selectedAgent) {
      console.log("ChatWidget: Using existing agent selection", selectedAgent.name);
      fetchAgentCardSafe(selectedAgent);
      return;
    }

    // Priority 2: Fallback to first agent (new session, persisted agent removed, or no selection)
    const firstAgent = availableAgents[0];
    if (firstAgent) {
      console.log(
        needsReselection
          ? "ChatWidget: Re-selecting first agent after previous agent removed"
          : "ChatWidget: Auto-selecting first agent",
        firstAgent.name
      );
      setSelectedAgent(firstAgent);
      fetchAgentCardSafe(firstAgent);
    } else {
      console.error("No agents available for selection");
      useStore.getState().setError("No agents available. Please check your configuration.");
    }
  }, [agentsLoaded, availableAgents, selectedAgent, setSelectedAgent, fetchAgentCardSafe]);

  // Note: Agent loading is now centralized in App.tsx
  // Removed duplicate loadAgents() call to prevent wasteful network requests
  // Agent selection logic is now in useAgentSelection hook

  const [showClearConfirm, setShowClearConfirm] = useState(false);

  const handleClearChat = () => {
    if (showClearConfirm) {
      createSession();
      setShowClearConfirm(false);
    } else {
      setShowClearConfirm(true);
      // Auto-hide confirmation after 3 seconds
      setTimeout(() => setShowClearConfirm(false), 3000);
    }
  };

  // Determine if the widget is in 'pane' mode
  const isPane = state === "pane";

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header/Controls */}
      <div className="flex items-center justify-between p-4 border-b border-white/10 flex-shrink-0 bg-black/20 backdrop-blur-sm">
        {/* Agent Selector */}
        <div className="relative">
          <select
            className="appearance-none bg-white/5 border border-white/10 text-white text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 pr-8"
            value={selectedAgent?.name || ""}
            onChange={handleAgentChange}
          >
            {availableAgents.map((agent) => (
              <option key={agent.name} value={agent.name}>
                {agent.name}
              </option>
            ))}
          </select>
          <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-gray-400">
            <ChevronDown size={16} />
          </div>
        </div>

        {/* Controls */}
        <div className="flex items-center gap-1 flex-shrink-0 ml-2">
          {showClearConfirm ? (
            <button
              onClick={handleClearChat}
              className="flex items-center gap-1 px-2 py-1.5 bg-red-500/20 text-red-400 hover:bg-red-500/30 hover:text-red-300 rounded transition-colors text-xs font-medium animate-in slide-in-from-right-2 fade-in"
              title="Confirm Clear"
            >
              <span>Sure?</span>
            </button>
          ) : (
            <button
              onClick={handleClearChat}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
              title="New Chat"
            >
              <Trash2 size={16} />
            </button>
          )}

          {/* Only show window controls if NOT in pane mode */}
          {!isPane && (
            <>
              <div className="w-px h-3 bg-white/10 mx-1" />
              <button
                onClick={() => onPinChange(!isPinned)}
                className={`p-1.5 rounded hover:bg-white/10 transition-colors ${isPinned ? "text-hector-green" : "text-gray-400"
                  }`}
                title={isPinned ? "Unpin" : "Pin"}
              >
                <Pin size={16} />
              </button>

              {state === "popup" && (
                <button
                  onClick={() => onStateChange("expanded")}
                  className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                  title="Expand"
                >
                  <Maximize2 size={16} />
                </button>
              )}

              {state === "expanded" && (
                <>
                  <button
                    onClick={() => onStateChange("popup")}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                    title="Minimize"
                  >
                    <Minimize2 size={16} />
                  </button>
                  <button
                    onClick={() => onStateChange("maximized")}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                    title="Maximize"
                  >
                    <Square size={16} />
                  </button>
                </>
              )}

              {state === "maximized" && (
                <button
                  onClick={() => onStateChange("expanded")}
                  className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                  title="Restore"
                >
                  <Minimize2 size={16} />
                </button>
              )}

              <button
                onClick={() => onStateChange("closed")}
                className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                title="Close"
              >
                <X size={16} />
              </button>
            </>
          )}
        </div>
      </div>

      {/* Chat Area */}
      <div className="flex-1 min-h-0 flex flex-col relative overflow-hidden">
        {(!sessionExists || !currentSessionId || msgCount === 0) ? (
          <div className="flex-1 flex flex-col items-center justify-center p-6 text-center overflow-y-auto">
            <div className="w-16 h-16 bg-gradient-to-br from-hector-green to-blue-600 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-hector-green/20">
              <svg
                className="w-8 h-8 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                />
              </svg>
            </div>
            <h3 className="text-lg font-semibold mb-2 text-white">Test Your Agent</h3>
            <p className="text-sm text-gray-400 max-w-xs mb-6">
              Deploy your config and start chatting to test your agent's behavior
            </p>
          </div>
        ) : (
          <div className="flex-1 min-h-0 flex flex-col">
            <MessageList sessionId={currentSessionId} />
          </div>
        )}

        {/* Input Area */}
        <div className="flex-shrink-0 border-t border-white/10 p-4 bg-black/40 backdrop-blur-md z-10">
          <InputArea onSend={onMessageSent} />
        </div>
      </div>
    </div>
  );
});
