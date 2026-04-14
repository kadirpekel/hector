import { useServersStore } from '../store/serversStore'

/**
 * Probes a server's /health endpoint and updates its status in the store.
 *
 * Maps HTTP responses to server states:
 *   200       → authenticated
 *   401 / 403 → auth_required
 *   other     → error
 *   network   → unreachable
 */
export function probeServer(
  id: string,
  url: string,
  options?: { timeout?: number }
): void {
  const { setServerStatus } = useServersStore.getState()
  setServerStatus(id, 'checking')

  const signal = options?.timeout
    ? AbortSignal.timeout(options.timeout)
    : undefined

  fetch(`${url.replace(/\/$/, '')}/health`, { signal })
    .then(async (res) => {
      const store = useServersStore.getState()
      if (res.ok) {
        // Validate JSON response is from a real Hector server
        try {
          const data = await res.json()
          if (data && typeof data === 'object' && 'status' in data) {
            store.setServerStatus(id, 'authenticated')
          } else {
            store.setServerStatus(id, 'error', 'Not a Hector server')
          }
        } catch {
          store.setServerStatus(id, 'error', 'Not a Hector server')
        }
      } else if (res.status === 401 || res.status === 403) {
        store.setServerStatus(id, 'auth_required')
      } else {
        store.setServerStatus(id, 'error', `Server returned ${res.status}`)
      }
    })
    .catch(() => {
      useServersStore.getState().setServerStatus(id, 'unreachable', 'Could not connect')
    })
}
