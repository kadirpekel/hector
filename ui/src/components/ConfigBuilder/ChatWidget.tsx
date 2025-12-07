import React, {
  useEffect,
  useState,
  useRef,
  useCallback,
  useMemo,
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
import { api } from "../../services/api";
import { MessageList } from "../Chat/MessageList";
import { InputArea } from "../Chat/InputArea";
import { handleError } from "../../lib/error-handler";
import { DEFAULT_SUPPORTED_FILE_TYPES } from "../../lib/constants";
import { SCROLL } from "../../lib/constants";

type ChatWidgetState = "closed" | "popup" | "expanded" | "maximized";

interface ChatWidgetProps {
  state: ChatWidgetState;
  onStateChange: (state: ChatWidgetState) => void;
  isPinned: boolean;
  onPinChange: (pinned: boolean) => void;
}

export const ChatWidget: React.FC<ChatWidgetProps> = ({
  state,
  onStateChange,
  isPinned,
  onPinChange,
}) => {
  const [agentsLoaded, setAgentsLoaded] = useState(false);

  // Store
  const currentSessionId = useStore((state) => state.currentSessionId);
  const sessions = useStore((state) => state.sessions);
  const availableAgents = useStore((state) => state.availableAgents);
  const selectedAgent = useStore((state) => state.selectedAgent);
  const setAvailableAgents = useStore((state) => state.setAvailableAgents);
  const setSelectedAgent = useStore((state) => state.setSelectedAgent);
  const setAgentCard = useStore((state) => state.setAgentCard);
  const createSession = useStore((state) => state.createSession);

  const session = currentSessionId ? sessions[currentSessionId] : null;
  const messages = session?.messages || [];
  const isGenerating = useStore((state) => state.isGenerating);
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

  // Memoized content hash for scroll trigger dependencies
  const contentHash = useMemo(() => {
    if (!session) return null;
    const messages = session.messages;
    const lastMsg = messages[messages.length - 1];

    return {
      messageCount: messages.length,
      lastMessageId: lastMsg?.id,
      lastMessageText: lastMsg?.text?.length ?? 0,
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
    };

    container.addEventListener("scroll", handleScroll);
    return () => container.removeEventListener("scroll", handleScroll);
  }, [isNearBottom]);

  // Auto-scroll when content changes
  useEffect(() => {
    if (contentHash === null) return;

    // Small delay to ensure DOM is updated
    const timeoutId = window.setTimeout(() => {
      scrollToBottom();
    }, 100);

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [contentHash, scrollToBottom]);

  // Load agents once
  useEffect(() => {
    if (agentsLoaded) return;

    const loadAgents = async () => {
      try {
        const data = await api.fetchAgents();
        setAvailableAgents(data.agents || []);

        if (!selectedAgent && data.agents && data.agents.length > 0) {
          const firstAgent = data.agents[0];
          setSelectedAgent(firstAgent);
          try {
            const card = await api.fetchAgentCard(firstAgent.url);
            setAgentCard(card);
          } catch (error) {
            handleError(error, "Failed to fetch agent card");
            useStore
              .getState()
              .setSupportedFileTypes([...DEFAULT_SUPPORTED_FILE_TYPES]);
          }
        }
        setAgentsLoaded(true);
      } catch (error) {
        handleError(error, "Failed to load agents");
      }
    };

    loadAgents();
  }, [
    agentsLoaded,
    setAvailableAgents,
    setSelectedAgent,
    setAgentCard,
    selectedAgent,
  ]);

  const handleAgentChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const index = parseInt(e.target.value);
    const agent = availableAgents[index];
    if (agent) {
      setSelectedAgent(agent);
      try {
        const card = await api.fetchAgentCard(agent.url);
        setAgentCard(card);
      } catch (error) {
        handleError(error, "Failed to fetch agent card");
        setAgentCard(null);
        useStore
          .getState()
          .setSupportedFileTypes([...DEFAULT_SUPPORTED_FILE_TYPES]);
      }
    }
  };

  const handleClearChat = () => {
    if (
      window.confirm(
        "Start a new chat session? Current conversation will be cleared.",
      )
    ) {
      // Create new session (persisted in localStorage automatically by store)
      createSession();
    }
  };

  // Size and position based on state
  const getWidgetStyles = () => {
    switch (state) {
      case "popup":
        return "w-96 h-[600px] bottom-6 right-6";
      case "expanded":
        return "w-[50vw] h-[80vh] bottom-6 right-6";
      case "maximized":
        return "w-full h-full bottom-0 right-0 rounded-none";
      default:
        return "";
    }
  };

  return (
    <div
      className={`chat-widget fixed bg-gradient-to-br from-hector-darker to-black border border-white/20 rounded-2xl shadow-2xl flex flex-col overflow-hidden backdrop-blur-xl z-50 animate-in slide-in-from-bottom-4 duration-300 ${getWidgetStyles()}`}
      style={{
        boxShadow: "0 20px 60px rgba(0, 0, 0, 0.5)",
      }}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-black/40 border-b border-white/10 backdrop-blur-md flex-shrink-0">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="w-2 h-2 bg-hector-green rounded-full animate-pulse flex-shrink-0"></div>
          <div className="flex-1 min-w-0">
            <select
              className="w-full bg-transparent border-none text-sm font-semibold appearance-none focus:outline-none pr-6 truncate cursor-pointer hover:text-hector-green transition-colors"
              onChange={handleAgentChange}
              value={
                selectedAgent ? availableAgents.indexOf(selectedAgent) : ""
              }
            >
              {availableAgents.length === 0 ? (
                <option>Loading...</option>
              ) : (
                availableAgents.map((agent, idx) => (
                  <option key={idx} value={idx}>
                    {agent.name}
                  </option>
                ))
              )}
            </select>
            <ChevronDown
              size={14}
              className="absolute right-24 top-4 text-gray-400 pointer-events-none"
            />
          </div>
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          <button
            onClick={handleClearChat}
            className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
            title="New chat"
          >
            <Trash2 size={16} />
          </button>
          <button
            onClick={() => onPinChange(!isPinned)}
            className={`p-1.5 rounded hover:bg-white/10 transition-colors ${
              isPinned ? "text-hector-green" : "text-gray-400"
            }`}
            title={isPinned ? "Unpin" : "Pin"}
          >
            <Pin size={16} />
          </button>

          {state === "popup" && (
            <button
              onClick={() => onStateChange("expanded")}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
              title="Expand to half screen"
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
                title="Maximize to full screen"
              >
                <Square size={16} />
              </button>
            </>
          )}

          {state === "maximized" && (
            <button
              onClick={() => onStateChange("expanded")}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
              title="Restore to half screen"
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
        </div>
      </div>

      {/* Chat Area with proper scrolling */}
      <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
        {messages.length === 0 ? (
          <div className="flex-1 flex flex-col items-center justify-center p-6 text-center overflow-y-auto">
            <div className="w-16 h-16 bg-gradient-to-br from-hector-green to-blue-600 rounded-2xl flex items-center justify-center mb-4">
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
            <h3 className="text-lg font-semibold mb-2">Test Your Agent</h3>
            <p className="text-sm text-gray-400 max-w-xs mb-6">
              Deploy your config and start chatting to test your agent's
              behavior
            </p>
          </div>
        ) : (
          <div className="flex-1 min-h-0 flex flex-col overflow-hidden">
            <div
              ref={scrollContainerRef}
              className="flex-1 overflow-y-auto px-4 py-2"
            >
              <MessageList messages={messages} />
              <div ref={messagesEndRef} />
            </div>
          </div>
        )}

        {/* Input Area - always at bottom */}
        <div className="flex-shrink-0 border-t border-white/10 p-4">
          <InputArea />
        </div>
      </div>
    </div>
  );
};
