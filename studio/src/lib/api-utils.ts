import { useServersStore } from "../store/serversStore";
import { useAppsStore } from "../store/appsStore";

/**
 * Get the base URL for API requests.
 * Single source of truth: derives from active server in serversStore.
 * Falls back to useStore.endpointUrl for backward compatibility (will be removed).
 */
export function getBaseUrl(): string {
    // Primary source: active server from serversStore
    const activeServer = useServersStore.getState().getActiveServer();
    if (activeServer?.status === 'authenticated') {
        return activeServer.config.url.replace(/\/$/, ''); // Remove trailing slash
    }

    // Default fallback
    return "http://localhost:8080";
}

/**
 * Get admin key for /admin/* endpoints.
 */
export function getAdminToken(): string | null {
    const activeServer = useServersStore.getState().getActiveServer();
    if (!activeServer) return null;

    return activeServer.config.adminKey || null;
}

/**
 * Get app JWT for regular API calls (/agents, /sessions, etc.).
 * Returns the JWT token for the currently active app on the active server.
 */
export function getAppToken(): string | null {
    const activeServer = useServersStore.getState().getActiveServer();
    if (!activeServer) return null;

    const serverId = activeServer.config.id;
    const serverData = useAppsStore.getState().appsByServer[serverId];
    if (!serverData?.activeAppId) return null;

    const appToken = serverData.appTokens[serverData.activeAppId];
    return appToken?.accessToken || null;
}

/**
 * Get auth token for current request (admin or app token based on path).
 * Centralized auth token retrieval - single source of truth.
 *
 * @param path - The API path being called (used to determine token type)
 * @returns The appropriate Bearer token or null
 */
export async function getAuthToken(path?: string): Promise<string | null> {
    // Public endpoints that don't require authentication
    const publicEndpoints = ['/health', '/schema'];
    if (path && publicEndpoints.some(endpoint => path === endpoint)) {
        return null;
    }

    // Determine if this is an admin endpoint
    const isAdminEndpoint = path?.startsWith('/admin');

    if (isAdminEndpoint) {
        // Admin endpoints: use admin key
        return getAdminToken();
    } else {
        // Regular endpoints: prefer app JWT, fall back to admin key
        // Admin key fallback allows access to "default" app - intentional for CLI users.
        const appToken = getAppToken();
        if (appToken) {
            return appToken;
        }
        return getAdminToken();
    }
}

/**
 * Get headers with auth token injected.
 * Use this when you need custom fetch logic but want auth headers.
 */
export async function getAuthHeaders(
    path?: string,
    additionalHeaders?: Record<string, string>
): Promise<Record<string, string>> {
    const headers: Record<string, string> = { ...additionalHeaders };
    const token = await getAuthToken(path);
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    return headers;
}

/**
 * Authenticated fetch wrapper.
 * Automatically injects auth headers and uses configured base URL.
 * Determines whether to use admin key or app JWT based on the path.
 *
 * @param path - Relative path (e.g., '/agents') or full URL
 * @param options - Standard fetch options
 * @returns Promise<Response>
 *
 * @example
 * // Simple GET to agent endpoint (uses app JWT)
 * const response = await apiFetch('/agents');
 *
 * // Admin API call (uses admin key)
 * const response = await apiFetch('/admin/apps');
 *
 * // POST with body
 * const response = await apiFetch('/sessions', {
 *   method: 'POST',
 *   headers: { 'Content-Type': 'application/json' },
 *   body: JSON.stringify({ agent: 'myagent' })
 * });
 */
export async function apiFetch(
    path: string,
    options: RequestInit = {}
): Promise<Response> {
    const baseUrl = getBaseUrl();
    const url = path.startsWith('http') ? path : `${baseUrl}${path}`;

    // Merge user-provided headers with auth headers
    const existingHeaders = options.headers as Record<string, string> | undefined;
    const headers = await getAuthHeaders(path, existingHeaders);

    const response = await fetch(url, {
        ...options,
        headers
    });

    // Detect auth errors (token expired/invalid) and dispatch event
    if (response.status === 401 || response.status === 403) {
        console.warn(`[apiFetch] Auth error ${response.status} for ${path}`);
        window.dispatchEvent(new CustomEvent('auth-expired', {
            detail: { path, status: response.status }
        }));
    }

    return response;
}
