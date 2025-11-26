/**
 * Application constants
 */

// Default supported file types for attachments
export const DEFAULT_SUPPORTED_FILE_TYPES = [
    'image/jpeg',
    'image/png',
    'image/gif',
    'image/webp'
] as const;

// Timing constants (in milliseconds)
export const TIMING = {
    AUTO_FOCUS_DELAY: 150,
    POST_GENERATION_FOCUS_DELAY: 200,
    ERROR_AUTO_DISMISS: 10000,
} as const;

// UI constants
export const UI = {
    MAX_TITLE_LENGTH: 50,
    MAX_TEXTAREA_HEIGHT: 200,
    CHAT_MAX_WIDTH: 760,
} as const;

