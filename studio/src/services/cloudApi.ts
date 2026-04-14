/**
 * Cloud API Service
 *
 * Talks to hector-cloud control plane for instance lifecycle.
 * Authentication uses Fly OAuth2 - Studio stores a cloud-issued JWT
 * obtained via the popup OAuth flow.
 */

import type { CloudInstance, CloudInstanceConfig } from '../types';

// Cloud API URL - set via VITE_HECTOR_CLOUD_URL to enable cloud features
const CLOUD_API_URL = import.meta.env.VITE_HECTOR_CLOUD_URL || '';

// ============================================================================
// Cloud Auth helpers (called by cloudAuthStore)
// ============================================================================

let cachedToken: string | null = null;

/** Set the cloud JWT obtained from the OAuth callback. */
export function setCloudToken(jwt: string): void {
    cachedToken = jwt;
}

/** Return the cached cloud JWT (without validation - expiry is checked by the store). */
export function getCloudToken(): string | null {
    return cachedToken;
}

/** Clear cached cloud JWT. */
export function clearCloudAuth(): void {
    cachedToken = null;
}

// ============================================================================
// Internal fetch helper
// ============================================================================

async function cloudFetch(
    path: string,
    options: RequestInit = {}
): Promise<Response> {
    if (!cachedToken) {
        throw new Error('Not authenticated with Hector Cloud');
    }
    return fetch(`${CLOUD_API_URL}${path}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${cachedToken}`,
            ...((options.headers as Record<string, string>) || {}),
        },
    });
}

// ============================================================================
// Instance Lifecycle
// ============================================================================

/**
 * Provision a cloud instance (idempotent - returns existing if already provisioned).
 */
export async function provisionInstance(
    config?: CloudInstanceConfig
): Promise<CloudInstance> {
    const response = await cloudFetch('/instances', {
        method: 'POST',
        body: JSON.stringify(config || {}),
    });

    if (!response.ok) {
        const errText = await response.text();
        throw new Error(`Failed to provision instance: ${response.status} ${errText}`);
    }

    return response.json();
}

/**
 * Get instance status.
 */
export async function getInstance(
    instanceId: string
): Promise<CloudInstance> {
    const response = await cloudFetch(`/instances/${instanceId}`);

    if (!response.ok) {
        throw new Error(`Failed to get instance: ${response.status}`);
    }

    return response.json();
}

/**
 * Start a stopped/suspended instance.
 */
export async function startInstance(
    instanceId: string
): Promise<void> {
    const response = await cloudFetch(`/instances/${instanceId}/start`, {
        method: 'POST',
    });

    if (!response.ok) {
        throw new Error(`Failed to start instance: ${response.status}`);
    }
}

/**
 * Update instance config (env vars, server settings).
 */
export async function updateInstance(
    instanceId: string,
    config: Partial<CloudInstanceConfig>
): Promise<void> {
    const response = await cloudFetch(`/instances/${instanceId}`, {
        method: 'PUT',
        body: JSON.stringify(config),
    });

    if (!response.ok) {
        throw new Error(`Failed to update instance: ${response.status}`);
    }
}

/**
 * Delete/deprovision the cloud instance.
 */
export async function destroyInstance(): Promise<void> {
    const response = await cloudFetch('/instances', {
        method: 'DELETE',
    });

    // 404 = already deleted, treat as success
    if (response.status === 404) return;

    if (!response.ok) {
        throw new Error(`Failed to destroy instance: ${response.status}`);
    }
}

