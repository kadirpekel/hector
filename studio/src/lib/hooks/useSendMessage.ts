import { useCallback } from 'react';
import { useStore } from '../../store/useStore';
import { StreamParser } from '../stream-parser';
import { createStreamDispatcher } from '../stream-utils';
import { generateId, generateShortId } from '../id-generator';
import { UI } from '../constants';
import { getBaseUrl } from '../api-utils';
import type { Attachment } from '../../types';

/**
 * Hook to handle message sending logic.
 * Extracts complex message preparation and streaming logic from InputArea component.
 */
export function useSendMessage() {
  const currentSessionId = useStore((s) => s.currentSessionId);
  const selectedAgent = useStore((s) => s.selectedAgent);
  const addMessage = useStore((s) => s.addMessage);
  const updateSessionTitle = useStore((s) => s.updateSessionTitle);
  const setIsGenerating = useStore((s) => s.setIsGenerating);
  const setActiveStreamParser = useStore((s) => s.setActiveStreamParser);
  const setError = useStore((s) => s.setError);

  const sendMessage = useCallback(
    async (text: string, attachments: Attachment[]): Promise<void> => {
      if (!currentSessionId) {
        console.error('[useSendMessage] No active session');
        return;
      }

      if (!selectedAgent) {
        setError('No agent selected. Please select an agent to continue.');
        return;
      }

      const messageText = text.trim();

      // Add user message to UI
      const userMessageId = generateId();
      addMessage(currentSessionId, {
        id: userMessageId,
        role: 'user',
        text: messageText,
        metadata: {
          images: attachments,
        },
        widgets: [],
        time: new Date().toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        }),
      });

      // Update session title from first user message
      const sessions = useStore.getState().sessions;
      const session = sessions[currentSessionId];
      if (session && session.title === 'New conversation' && messageText) {
        const title =
          messageText.length > UI.MAX_TITLE_LENGTH
            ? messageText.substring(0, UI.MAX_TITLE_LENGTH) + '...'
            : messageText;
        updateSessionTitle(currentSessionId, title);
      }

      // Create agent message placeholder
      const agentMessageId = generateId();
      addMessage(currentSessionId, {
        id: agentMessageId,
        role: 'agent',
        text: '',
        metadata: {},
        widgets: [],
        time: new Date().toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        }),
      });

      setIsGenerating(true);

      // Create dispatcher and parser
      const dispatcher = createStreamDispatcher();
      const parser = new StreamParser(currentSessionId, agentMessageId, dispatcher);
      setActiveStreamParser(parser);

      try {
        // Prepare parts - A2A spec requires "kind" field
        const parts: Array<{
          kind: string;
          text?: string;
          file?: { bytes: string; mimeType: string; name: string };
        }> = [];

        if (messageText) {
          parts.push({ kind: 'text', text: messageText });
        }

        for (const att of attachments) {
          parts.push({
            kind: 'file',
            file: {
              bytes: att.base64,
              mimeType: att.mediaType,
              name: att.file.name,
            },
          });
        }

        // A2A spec: params.message with contextId for multi-turn
        const currentSession = sessions[currentSessionId];
        const messageParams: any = {
          contextId: currentSession.contextId,
          role: 'user',
          parts: parts,
        };

        const requestBody = {
          jsonrpc: '2.0',
          method: 'message/stream',
          params: {
            message: messageParams,
          },
          id: generateShortId(),
        };

      // Resolve full URL
      const baseUrl = getBaseUrl();
      
      if (!selectedAgent.url) {
        throw new Error(`Agent "${selectedAgent.name}" has no URL - backend should return URL in /agents discovery`);
      }

      // Handle relative URLs
      let fullUrl = selectedAgent.url;
      if (!fullUrl.startsWith('http')) {
        // Ensure strictly one slash between base and path
        const base = baseUrl.replace(/\/$/, '');
        const path = fullUrl.startsWith('/') ? fullUrl : `/${fullUrl}`;
        fullUrl = `${base}${path}`;
      }

        // A2A spec: POST to agent's URL - streaming is determined by method name
        await parser.stream(fullUrl, requestBody);
      } catch (error: unknown) {
        if (error instanceof Error && error.name !== 'AbortError') {
          console.error('[useSendMessage] Failed to send message:', error);
          setError(`Failed to send message: ${error.message}`);
          setIsGenerating(false);
        }
      } finally {
        // Note: Don't set isGenerating(false) here - StreamParser.stream() handles it
        // This prevents prematurely showing the send button during HITL approval flow
        parser.cleanup(); // Clean up widget tracking to prevent memory leaks
        setActiveStreamParser(null);
      }
    },
    [
      currentSessionId,
      selectedAgent,
      addMessage,
      updateSessionTitle,
      setIsGenerating,
      setActiveStreamParser,
      setError,
    ]
  );

  return { sendMessage };
}
