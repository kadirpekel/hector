import { v4 as uuidv4 } from 'uuid';

/**
 * Generate a unique ID for client-side use
 * Uses UUID for consistency with session/message IDs
 */
export function generateId(): string {
    return uuidv4();
}

/**
 * Generate a short ID for temporary use (e.g., request IDs)
 * Uses timestamp + random for shorter IDs when UUID is not needed
 */
export function generateShortId(): string {
    return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

