import { useEffect, useRef, useCallback } from 'react';
import { useServersStore } from '../../store/serversStore';
import { useStore } from '../../store/useStore';
import { logger } from '../utils/logger';

const POLLING_INTERVAL_MS = 10_000; // 10 seconds (normal)
const POLLING_INTERVAL_INDEXING_MS = 3_000; // 3 seconds (during indexing)
const STARTUP_TIMEOUT_MS = 30_000; // 30 seconds

/**
 * Hook that polls the active server's health endpoint every 10 seconds.
 * Automatically updates server status to 'disconnected' on failure,
 * and restores to 'authenticated' on recovery.
 * Shows toast notifications on connection state changes.
 * Implements 30-second timeout for startup 'checking' state.
 */
export function useHealthPolling() {
    const activeServer = useServersStore((s) => s.getActiveServer());
    const setServerStatus = useServersStore((s) => s.setServerStatus);

    // Track previous status to detect transitions
    const prevStatusRef = useRef<string | null>(null);
    const isPollingRef = useRef(false);
    // Track active server ID to prevent stale closure issues
    const activeServerIdRef = useRef<string | null>(null);

    // Update ref when active server changes
    useEffect(() => {
        activeServerIdRef.current = activeServer?.config.id ?? null;
    }, [activeServer?.config.id]);

    const checkHealth = useCallback(async () => {
        // Get fresh server state from store to avoid stale closure
        const currentServer = useServersStore.getState().getActiveServer();

        if (!currentServer || isPollingRef.current) return;

        // Verify we're still on the same server (prevent stale callback execution)
        if (currentServer.config.id !== activeServerIdRef.current) {
            logger.debug('Aborting stale health check');
            return;
        }

        // Skip polling for placeholder URLs (e.g. during cloud provision)
        if (currentServer.config.url.startsWith('cloud://')) {
            logger.debug('Skipping health check for placeholder URL', { url: currentServer.config.url });
            return;
        }

        logger.debug('Starting health check for server:', {
            id: currentServer.config.id,
            url: currentServer.config.url,
            status: currentServer.status
        });

        // Only poll if currently authenticated, disconnected (recovery), or checking (startup verification)
        // Skip during transient states: 'stopping', 'stopped', 'added'
        const pollableStates = ['authenticated', 'disconnected', 'checking'];
        if (!pollableStates.includes(currentServer.status)) {
            return;
        }

        // Check for startup timeout
        if (currentServer.status === 'checking' && currentServer.startedAt) {
            const elapsed = Date.now() - currentServer.startedAt;
            if (elapsed > STARTUP_TIMEOUT_MS) {
                logger.warn('Startup timeout exceeded', {
                    serverId: currentServer.config.id,
                    elapsed
                });
                setServerStatus(
                    currentServer.config.id,
                    'disconnected',
                    'Startup timeout - server not responding'
                );
                useStore.getState().setError(
                    `${currentServer.config.name} failed to start within 30 seconds`
                );
                return;
            }
        }

        isPollingRef.current = true;
        const serverIdAtStart = currentServer.config.id;

        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 5000); // 5s timeout

            const res = await fetch(`${currentServer.config.url}/health`, {
                signal: controller.signal,
            });
            clearTimeout(timeoutId);

            // Re-check: server may have switched during fetch
            const serverAfterFetch = useServersStore.getState().getActiveServer();
            if (serverAfterFetch?.config.id !== serverIdAtStart) {
                logger.debug('Server switched during health check, ignoring result');
                return;
            }

            if (res.ok) {
                // Parse health response for RAG status
                try {
                    const healthData = await res.json();

                    // Update RAG indexing status if present
                    if (healthData.rag) {
                        useStore.getState().setRagStatus({
                            isIndexing: healthData.rag.indexing || false,
                            stores: healthData.rag.stores || {},
                        });
                    } else {
                        // No RAG configured - clear status
                        useStore.getState().setRagStatus(null);
                    }
                } catch {
                    // Failed to parse JSON - just continue with status update
                }

                // SUCCESS: Promote to authenticated
                // This is the ONLY place where a server becomes 'authenticated' now
                if (currentServer.status !== 'authenticated') {
                    logger.info('Connection verified', { server: currentServer.config.name });
                    setServerStatus(currentServer.config.id, 'authenticated');
                    useStore.getState().setSuccessMessage(
                        `Connected to ${currentServer.config.name}`
                    );
                }
            } else {
                throw new Error(`Health check returned ${res.status}`);
            }
        } catch (error) {
            // Re-check current state from store
            const serverAfterError = useServersStore.getState().getActiveServer();
            const stillSameServer = serverAfterError?.config.id === serverIdAtStart;

            if (!stillSameServer) return;

            // STARTUP GRACE PERIOD LOGIC
            // If we are 'checking' (startup), log and keep retrying
            // The timeout logic above will handle marking as disconnected after 30s
            if (currentServer.status === 'checking') {
                 logger.debug('Startup health check failed, retrying', {
                     server: currentServer.config.name,
                     error: error instanceof Error ? error.message : String(error)
                 });
                 return;
            }

            // Normal operation error handling
            const wasAuthenticated = currentServer.status === 'authenticated';

            // Only report error if same server is still active and was authenticated
            if (wasAuthenticated) {
                logger.warn('Connection lost', {
                    server: currentServer.config.name,
                    error: error instanceof Error ? error.message : String(error)
                });
                setServerStatus(
                    currentServer.config.id,
                    'disconnected',
                    'Connection lost. Retrying...'
                );
                useStore.getState().setError(
                    `Connection lost to ${currentServer.config.name}`
                );
            }
        } finally {
            isPollingRef.current = false;
        }
    }, [setServerStatus]);

    useEffect(() => {
        // Track status transitions for debugging
        if (activeServer?.status !== prevStatusRef.current) {
            logger.debug('Server status changed', {
                from: prevStatusRef.current,
                to: activeServer?.status,
                server: activeServer?.config.name
            });
            prevStatusRef.current = activeServer?.status ?? null;
        }
    }, [activeServer?.status]);

    useEffect(() => {
        // Only start polling if we have an active server in pollable state
        if (!activeServer || (
            activeServer.status !== 'authenticated' &&
            activeServer.status !== 'disconnected' &&
            activeServer.status !== 'checking'
        )) {
            return;
        }

        // Initial check (with slight delay to avoid initial load race)
        const initialTimeout = setTimeout(checkHealth, 1000);

        // Use faster polling during indexing for better UX
        const ragStatus = useStore.getState().ragStatus;
        const interval = ragStatus?.isIndexing ? POLLING_INTERVAL_INDEXING_MS : POLLING_INTERVAL_MS;
        const intervalId = setInterval(checkHealth, interval);

        return () => {
            clearTimeout(initialTimeout);
            clearInterval(intervalId);
        };
    }, [activeServer?.config.id, activeServer?.status, checkHealth]);
}
