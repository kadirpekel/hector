/**
 * Application constants
 *
 * Centralized location for all magic numbers and configuration values.
 * Add documentation for any non-obvious values.
 */

// Default supported file types for attachments
export const DEFAULT_SUPPORTED_FILE_TYPES = [
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
] as const;

// Timing constants (in milliseconds)
export const TIMING = {
  /** Delay before auto-focusing input after session change */
  AUTO_FOCUS_DELAY: 150,
  /** Delay before auto-focusing input after generation completes */
  POST_GENERATION_FOCUS_DELAY: 200,
  /** How long error messages display before auto-dismissing */
  ERROR_AUTO_DISMISS: 10000,
} as const;

// UI constants
export const UI = {
  /** Maximum characters for session title before truncation */
  MAX_TITLE_LENGTH: 50,
  /** Maximum height of textarea input in pixels */
  MAX_TEXTAREA_HEIGHT: 200,
  /** Maximum width of chat content area in pixels */
  CHAT_MAX_WIDTH: 760,
} as const;

// Scroll behavior constants
export const SCROLL = {
  /**
   * Distance from bottom (in pixels) within which auto-scroll remains active.
   * If user scrolls more than this distance from bottom, auto-scroll pauses.
   */
  THRESHOLD_PX: 100,
  /** Delay before re-enabling auto-scroll after user stops scrolling */
  RESET_DELAY_MS: 1000,
  /** Debounce delay for MutationObserver scroll triggers */
  MUTATION_DEBOUNCE_MS: 50,
} as const;

// Widget behavior constants
export const WIDGET = {
  /**
   * Delay (in milliseconds) before auto-collapsing completed widgets.
   * Gives users time to see the result before the widget collapses.
   */
  COLLAPSE_DELAY_MS: 4000,
} as const;

// Stream parsing constants
export const STREAM = {
  /**
   * Maximum number of parse errors before surfacing to user.
   * Prevents error spam while still alerting on persistent issues.
   */
  MAX_PARSE_ERRORS: 5,
} as const;

// Editor constants
export const EDITOR = {
  /**
   * Debounce delay for YAML editor changes in milliseconds.
   * Balances responsiveness with performance by batching rapid edits.
   */
  DEBOUNCE_DELAY_MS: 500,
} as const;
