import { useStore } from '../store/useStore';

/**
 * Get the base URL for API requests
 * Uses configured endpoint URL or falls back to current origin
 */
export function getBaseUrl(): string {
    const state = useStore.getState();
    return state.endpointUrl || window.location.origin;
}

