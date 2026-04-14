/**
 * Cloud connectivity is enabled when a cloud URL is configured.
 *
 * Set VITE_HECTOR_CLOUD_URL in your environment or .env file
 * to enable Hector Cloud features (provisioning, authentication).
 *
 * Default: disabled (pure self-hosted mode).
 */
export const CLOUD_ENABLED = !!import.meta.env.VITE_HECTOR_CLOUD_URL
