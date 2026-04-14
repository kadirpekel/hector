import React from "react";
import {
  X,
  Pin,
  Maximize2,
  Minimize2,
  History,
  Square,
  ScanEye,
} from "lucide-react";
import { cn } from "../../lib/utils";
// Import ChatArea instead of individual components
import { ChatArea } from "../Chat/ChatArea";
import { useStore } from "../../store/useStore";
import { useAgentSelection } from "../../lib/hooks/useAgentSelection";
import { AgentSelect } from "../AgentSelect";

type ChatWidgetState = "closed" | "popup" | "expanded" | "maximized" | "pane";

interface ChatWidgetProps {
  state: ChatWidgetState;
  onStateChange: (state: ChatWidgetState) => void;
  isPinned: boolean;
  onPinChange: (pinned: boolean) => void;
  hideControls?: boolean;
}

export const ChatWidget: React.FC<ChatWidgetProps> = React.memo(({
  state,
  onStateChange,
  isPinned,
  onPinChange,
  hideControls = false,
}) => {

  // Store
  const availableAgents = useStore((state) => state.availableAgents);
  const selectedAgent = useStore((state) => state.selectedAgent);
  const isHistoryPinned = useStore((state) => state.isHistoryPinned);
  const setIsHistoryPinned = useStore((state) => state.setIsHistoryPinned);
  const currentSessionId = useStore((state) => state.currentSessionId);
  const setXRaySessionId = useStore((state) => state.setXRaySessionId);

  const [isHistoryOpen, setIsHistoryOpen] = React.useState(false);

  const { handleAgentChange } = useAgentSelection();

  // Determine if the widget is in 'pane' mode
  const isPane = state === "pane";

  const toggleHistory = () => {
    if (isHistoryPinned) {
      setIsHistoryPinned(false);
    } else {
      setIsHistoryOpen(!isHistoryOpen);
    }
  };

  const toggleHistoryPin = () => {
    setIsHistoryPinned(!isHistoryPinned);
    if (!isHistoryPinned) {
      setIsHistoryOpen(false);
    }
  };

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Header/Controls */}
      <div className="flex items-center justify-between h-10 px-3 border-b border-white/10 flex-shrink-0 bg-black/20 backdrop-blur-sm">
        {/* Agent Selector + X-Ray Button */}
        <div className="flex items-center gap-1.5">
          <AgentSelect
            agents={availableAgents}
            selectedAgent={selectedAgent}
            onSelect={(name) => handleAgentChange(name)}
            variant="ghost"
            className="w-40 h-7 text-xs px-2 hover:bg-white/10 text-white justify-between"
          />
          {currentSessionId && (
            <button
              onClick={() => setXRaySessionId(currentSessionId)}
              className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
              title="Inspect Session (X-Ray)"
            >
              <ScanEye size={14} />
            </button>
          )}
        </div>

        {/* Controls */}
        <div className="flex items-center gap-1 flex-shrink-0 ml-2">

          {/* History Toggle */}
          <button
            onClick={toggleHistory}
            className={cn(
              "p-1.5 rounded transition-colors",
              isHistoryOpen || isHistoryPinned
                ? "bg-gray-800 text-white"
                : "text-gray-400 hover:text-white hover:bg-gray-800"
            )}
            title="Session History"
          >
            <History className="w-4 h-4" />
          </button>

          <div className="w-px h-3 bg-white/10 mx-1" />

          {/* Only show window controls if NOT in pane mode AND controls aren't hidden */}
          {!isPane && !hideControls && (
            <>
              <button
                onClick={() => onPinChange(!isPinned)}
                className={`p-1.5 rounded hover:bg-white/10 transition-colors ${isPinned ? "text-hector-green" : "text-gray-400"
                  }`}
                title={isPinned ? "Unpin" : "Pin"}
              >
                <Pin size={14} />
              </button>

              {state === "popup" && (
                <button
                  onClick={() => onStateChange("expanded")}
                  className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                  title="Expand"
                >
                  <Maximize2 size={14} />
                </button>
              )}

              {state === "expanded" && (
                <>
                  <button
                    onClick={() => onStateChange("popup")}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                    title="Minimize"
                  >
                    <Minimize2 size={14} />
                  </button>
                  <button
                    onClick={() => onStateChange("maximized")}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                    title="Maximize"
                  >
                    <Square size={14} />
                  </button>
                </>
              )}

              {state === "maximized" && (
                <button
                  onClick={() => onStateChange("expanded")}
                  className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                  title="Restore"
                >
                  <Minimize2 size={14} />
                </button>
              )}

              <button
                onClick={() => onStateChange("closed")}
                className="p-1.5 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
                title="Close"
              >
                <X size={14} />
              </button>
            </>
          )}
        </div>
      </div>

      {/* Replaced Custom Chat UI with ChatArea Component */}
      <div className="flex-1 min-h-0 flex flex-col relative overflow-hidden">
        <ChatArea
          isHistoryOpen={isHistoryOpen}
          isHistoryPinned={isHistoryPinned}
          onTogglePin={toggleHistoryPin}
          onCloseHistory={() => setIsHistoryOpen(false)}
        />
      </div>
    </div >
  );
});
