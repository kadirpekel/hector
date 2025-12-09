import { useStore } from "../store/useStore";
import type { Message } from "../types";

/**
 * Dispatcher interface for StreamParser.
 * Breaks circular dependencies by providing store methods without importing store.
 */
export interface StreamDispatcher {
  updateMessage: (sessionId: string, messageId: string, updates: Partial<Message>) => void;
  setActiveAgentId: (id: string | null) => void;
  getMessage: (sessionId: string, messageId: string) => Message | undefined;
  setSessionTaskId: (sessionId: string, taskId: string) => void;
  setIsGenerating: (isGenerating: boolean) => void;
}

/**
 * Creates a dispatcher object for StreamParser.
 *
 * This utility eliminates code duplication across InputArea and ApprovalWidget
 * by centralizing the dispatcher creation logic.
 *
 * The dispatcher pattern allows StreamParser to update store state without
 * directly importing useStore, which would create circular dependencies.
 *
 * @returns StreamDispatcher object with store update methods
 */
export function createStreamDispatcher(): StreamDispatcher {
  const store = useStore.getState();

  return {
    updateMessage: store.updateMessage,
    setActiveAgentId: store.setActiveAgentId,
    getMessage: (sessionId: string, messageId: string) => {
      const session = useStore.getState().sessions[sessionId];
      return session?.messages.find(m => m.id === messageId);
    },
    setSessionTaskId: store.setSessionTaskId,
    setIsGenerating: store.setIsGenerating,
  };
}
