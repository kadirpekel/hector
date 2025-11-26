import { useStore } from '../store/useStore';

/**
 * Centralized error handling utility
 */
export function handleError(error: unknown, context?: string): void {
    const errorMessage = error instanceof Error 
        ? error.message 
        : typeof error === 'string'
        ? error
        : 'An unknown error occurred';
    
    const fullMessage = context 
        ? `${context}: ${errorMessage}`
        : errorMessage;
    
    // Set global error state (will be displayed by ErrorDisplay component)
    useStore.getState().setError(fullMessage);
    
    // Log to console for debugging (in development)
    if (import.meta.env.DEV) {
        console.error(fullMessage, error);
    }
}

