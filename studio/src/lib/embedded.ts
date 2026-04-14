/**
 * Host server auto-detection.
 *
 * When Studio is served by a Hector server (embedded mode), it auto-connects
 * to the host by probing /health on the same origin. In standalone mode
 * (e.g. npm run dev), the probe fails and normal server management is shown.
 */

/** Stable server ID for the auto-detected host server */
export const HOST_SERVER_ID = '__host__'
