/**
 * Hook that polls the active session for new events.
 * 
 * Design:
 * - Only polls sessions that have been initially loaded (have a cursor)
 * - Polls every 5 seconds, NOT on mount (initial load handled separately)
 * - Uses cursor-based fetching to only get new events
 * - Stops polling when session changes or component unmounts
 */

import { useEffect, useRef } from 'react';
import { useStore } from '../../store/useStore';

const POLLING_INTERVAL_MS = 5000; // 5 seconds

export function useSessionPolling() {
  const currentSessionId = useStore((state) => state.currentSessionId);
  const isPollingRef = useRef(false);
  const intervalIdRef = useRef<NodeJS.Timeout | null>(null);
  // Track which session we're polling to prevent stale closures
  const pollingSessionIdRef = useRef<string | null>(null);

  useEffect(() => {
    // Clear any existing polling
    if (intervalIdRef.current) {
      clearInterval(intervalIdRef.current);
      intervalIdRef.current = null;
    }
    isPollingRef.current = false;
    pollingSessionIdRef.current = currentSessionId;

    // Only poll if we have an active session
    if (!currentSessionId) {
      return;
    }

    // Capture session ID for this effect instance
    const sessionIdToPoll = currentSessionId;

    // Poll for new events (cursor-based incremental fetch)
    const pollSession = async () => {
      // Double-check we're still polling the right session
      if (pollingSessionIdRef.current !== sessionIdToPoll) {
        return;
      }
      
      if (isPollingRef.current) return; // Prevent concurrent polls
      
      const state = useStore.getState();
      const session = state.sessions[sessionIdToPoll];
      
      // Skip if session doesn't exist or hasn't been initially loaded yet
      if (!session || !session.eventsLoaded) {
        return;
      }
      
      // Skip if session has active task (streaming in progress)
      // Stream events are handled in real-time, no need to poll
      if (session.taskId) {
        return;
      }
      
      // Skip if session has no cursor (means it was never loaded with events)
      // Polling only makes sense for sessions that have been loaded
      if (!session.lastEventId) {
        return;
      }
      
      isPollingRef.current = true;
      try {
        await state.pollSessionForNewEvents(sessionIdToPoll);
      } catch (error) {
        // Silent fail - don't spam console
        console.debug('Session polling error:', error);
      } finally {
        isPollingRef.current = false;
      }
    };

    // Don't poll immediately - initial load is handled by loadSessionEvents
    // Only set up interval for subsequent polls
    intervalIdRef.current = setInterval(pollSession, POLLING_INTERVAL_MS);

    // Cleanup on unmount or session change
    return () => {
      if (intervalIdRef.current) {
        clearInterval(intervalIdRef.current);
        intervalIdRef.current = null;
      }
      isPollingRef.current = false;
      pollingSessionIdRef.current = null;
    };
  }, [currentSessionId]);
}
