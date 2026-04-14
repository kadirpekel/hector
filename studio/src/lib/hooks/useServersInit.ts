/**
 * useServersInit
 *
 * On mount:
 * 1. Probes /health on the same origin to detect if served by Hector.
 *    If so, auto-creates a host server entry and selects it.
 * 2. Probes all other persisted servers to refresh their status.
 */

import { useEffect, useRef } from 'react';
import { useServersStore } from '../../store/serversStore';
import { HOST_SERVER_ID } from '../embedded';
import { probeServer } from '../probeServer';

export function useServersInit() {
  const didInit = useRef(false);

  useEffect(() => {
    if (didInit.current) return;
    didInit.current = true;

    const store = useServersStore.getState();

    // 1. Probe same-origin /health to detect if served by Hector
    fetch('/health', { signal: AbortSignal.timeout(3000) })
      .then((res) => {
        if (!res.ok) return;
        // Validate the response is actually from Hector (not a Vite SPA fallback)
        return res.json().then((data) => {
          if (!data || typeof data !== 'object' || !('status' in data)) return;
          const { servers, addServer, selectServer } = useServersStore.getState();
          const hostUrl = window.location.origin;

          if (!servers[HOST_SERVER_ID]) {
            addServer({ id: HOST_SERVER_ID, name: 'Hector', url: hostUrl });
          }
          selectServer(HOST_SERVER_ID);
          probeServer(HOST_SERVER_ID, hostUrl, { timeout: 5000 });
        });
      })
      .catch(() => {
        // Not served by Hector - clean up stale host entry if present
        if (store.servers[HOST_SERVER_ID]) {
          useServersStore.getState().removeServer(HOST_SERVER_ID);
        }
      });

    // 2. Probe all other persisted servers
    for (const [id, server] of Object.entries(store.servers)) {
      if (id === HOST_SERVER_ID) continue;
      probeServer(id, server.config.url);
    }
  }, []);
}
