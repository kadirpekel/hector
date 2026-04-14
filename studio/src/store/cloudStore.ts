/**
 * Cloud Store
 *
 * Orchestrates cloud instance lifecycle:
 *   license key → cloud JWT → provision → auto-add server → health check → ready
 *
 * The provisioned instance IS a regular hector server.
 * Once connected, it works identically to a self-hosted one.
 */

import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import {
    provisionInstance,
    getInstance,
    startInstance,
    updateInstance,
    destroyInstance,
    clearCloudAuth,
} from '../services/cloudApi'
import { useCloudAuthStore } from './cloudAuthStore'
import { useServersStore } from './serversStore'
import type { CloudInstance } from '../types'

// ============================================================================
// Progress Steps - ordered lifecycle for the UI progress tracker
// ============================================================================

export interface CloudStep {
    id: string
    label: string
    status: 'pending' | 'active' | 'done' | 'error'
    detail?: string
}

const CONNECT_STEPS: () => CloudStep[] = () => [
    { id: 'auth',       label: 'Authenticating',          status: 'pending' },
    { id: 'provision',  label: 'Provisioning instance',   status: 'pending' },
    { id: 'start',      label: 'Starting instance',       status: 'pending' },
    { id: 'server',     label: 'Configuring server',      status: 'pending' },
    { id: 'health',     label: 'Waiting for health check', status: 'pending' },
]

type CloudStatus = 'idle' | 'working' | 'connected' | 'error'

const HEALTH_POLL_INTERVAL = 2_000  // 2s between retries
const HEALTH_TIMEOUT = 60_000       // 60s total timeout

interface CloudState {
    // State
    status: CloudStatus
    steps: CloudStep[]
    instanceId: string | null
    serverId: string | null
    host: string | null
    region: string | null
    error: string | null

    // Actions
    connect: () => Promise<void>
    disconnect: () => void
    destroy: () => Promise<void>
    refreshStatus: () => Promise<void>
    updateEnvVars: (envVars: Record<string, string>) => Promise<void>
    dismissProgress: () => void
}

// Helper to advance steps immutably
function advanceStep(steps: CloudStep[], stepId: string, status: CloudStep['status'], detail?: string): CloudStep[] {
    return steps.map(s => {
        if (s.id === stepId) return { ...s, status, detail }
        // Mark earlier steps as done if we're advancing past them
        if (status === 'active' && s.status === 'active') return { ...s, status: 'done' }
        return s
    })
}

function markAllDone(steps: CloudStep[]): CloudStep[] {
    return steps.map(s => ({ ...s, status: 'done' as const }))
}

function markError(steps: CloudStep[], stepId: string, detail: string): CloudStep[] {
    return steps.map(s => {
        if (s.id === stepId) return { ...s, status: 'error' as const, detail }
        return s
    })
}

export const useCloudStore = create<CloudState>()(
    persist(
        (set, get) => ({
            status: 'idle',
            steps: [],
            instanceId: null,
            serverId: null,
            host: null,
            region: null,
            error: null,

            /**
             * Connect to Hector Cloud with step-by-step progress tracking.
             */
            connect: async () => {
                const isAuthenticated = useCloudAuthStore.getState().isAuthenticated
                if (!isAuthenticated) {
                    set({ status: 'error', steps: [], error: 'Not authenticated. Please sign in with Fly first.' })
                    return
                }

                const steps = CONNECT_STEPS()
                set({ status: 'working', steps, error: null })

                try {
                    // Step 1: Authenticate
                    set({ steps: advanceStep(steps, 'auth', 'active', 'Verifying cloud credentials...') })
                    set({ steps: advanceStep(steps, 'auth', 'done') })

                    // Step 2: Provision
                    let currentSteps = advanceStep(get().steps, 'provision', 'active', 'Creating cloud instance...')
                    set({ steps: currentSteps })

                    const instance: CloudInstance = await provisionInstance()
                    if (!instance.id || !instance.host) {
                        throw new Error('Invalid instance response: missing id or host')
                    }

                    currentSteps = advanceStep(get().steps, 'provision', 'done', `Instance ${instance.id.slice(0, 8)}...`)
                    set({ steps: currentSteps })

                    // Step 3: Start (if needed)
                    if (instance.status === 'stopped' || instance.status === 'suspended') {
                        currentSteps = advanceStep(get().steps, 'start', 'active', 'Waking up instance...')
                        set({ steps: currentSteps })
                        await startInstance(instance.id)
                        currentSteps = advanceStep(get().steps, 'start', 'done')
                        set({ steps: currentSteps })
                    } else {
                        currentSteps = advanceStep(get().steps, 'start', 'done', 'Already running')
                        set({ steps: currentSteps })
                    }

                    // Step 4: Configure server entry
                    currentSteps = advanceStep(get().steps, 'server', 'active', 'Adding server...')
                    set({ steps: currentSteps })

                    const serverUrl = `https://${instance.host}`
                    const adminKey = instance.auth_secret || undefined
                    const { serverId: existingServerId } = get()
                    const serversStore = useServersStore.getState()

                    let serverId = existingServerId

                    if (serverId && serversStore.servers[serverId]) {
                        serversStore.updateServerConfig(serverId, {
                            url: serverUrl,
                            ...(adminKey ? { adminKey } : {}),
                            type: 'managed-cloud',
                            cloud: { instanceId: instance.id, region: instance.region },
                        })
                    } else {
                        serverId = crypto.randomUUID()
                        serversStore.addServer({
                            id: serverId,
                            name: 'Hector Cloud',
                            url: serverUrl,
                            adminKey,
                            type: 'managed-cloud',
                            cloud: { instanceId: instance.id, region: instance.region },
                        })
                    }

                    serversStore.selectServer(serverId)
                    // Set to 'checking' - NOT 'authenticated' yet
                    serversStore.setServerStatus(serverId, 'checking')

                    currentSteps = advanceStep(get().steps, 'server', 'done')
                    set({
                        steps: currentSteps,
                        instanceId: instance.id,
                        serverId,
                        host: instance.host,
                        region: instance.region || null,
                    })

                    // Step 5: Wait for /health to succeed
                    currentSteps = advanceStep(get().steps, 'health', 'active', 'Waiting for server to become ready...')
                    set({ steps: currentSteps })

                    await waitForHealth(serverUrl, adminKey)

                    // Health passed - mark authenticated
                    serversStore.setServerStatus(serverId, 'authenticated')

                    set({
                        status: 'connected',
                        steps: markAllDone(get().steps),
                        error: null,
                    })

                } catch (err) {
                    const msg = err instanceof Error ? err.message : String(err)
                    // Find which step was active and mark it as error
                    const activeStep = get().steps.find(s => s.status === 'active')
                    set({
                        status: 'error',
                        steps: activeStep
                            ? markError(get().steps, activeStep.id, msg)
                            : get().steps,
                        error: msg,
                    })
                }
            },

            disconnect: () => {
                const { serverId } = get()
                if (serverId) {
                    const serversStore = useServersStore.getState()
                    if (serversStore.servers[serverId]) {
                        serversStore.setServerStatus(serverId, 'disconnected')
                    }
                }
                set({ status: 'idle', steps: [], error: null })
            },

            destroy: async () => {
                try {
                    await destroyInstance()
                } catch (err) {
                    console.error('[cloudStore] Failed to destroy instance:', err)
                }

                const { serverId } = get()
                if (serverId) {
                    useServersStore.getState().removeServer(serverId)
                }

                clearCloudAuth()
                set({
                    status: 'idle',
                    steps: [],
                    instanceId: null,
                    serverId: null,
                    host: null,
                    region: null,
                    error: null,
                })
            },

            refreshStatus: async () => {
                const { instanceId, serverId } = get()
                if (!instanceId) return

                try {
                    const instance = await getInstance(instanceId)
                    set({ host: instance.host, region: instance.region || null })

                    if (serverId) {
                        const serversStore = useServersStore.getState()
                        if (instance.status === 'running') {
                            serversStore.setServerStatus(serverId, 'authenticated')
                            set({ status: 'connected' })
                        } else if (instance.status === 'stopped' || instance.status === 'suspended') {
                            serversStore.setServerStatus(serverId, 'stopped')
                            set({ status: 'idle' })
                        } else if (instance.status === 'failed') {
                            serversStore.setServerStatus(serverId, 'error', 'Cloud instance failed')
                            set({ status: 'error', error: 'Cloud instance failed' })
                        }
                    }
                } catch (err) {
                    console.error('[cloudStore] Failed to refresh status:', err)
                }
            },

            updateEnvVars: async (envVars: Record<string, string>) => {
                const { instanceId } = get()
                if (!instanceId) return

                await updateInstance(instanceId, { env: envVars })
            },

            /** Dismiss the progress overlay (after success or error acknowledgment). */
            dismissProgress: () => {
                set({ steps: [] })
            },
        }),
        {
            name: 'hector_cloud',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                instanceId: state.instanceId,
                serverId: state.serverId,
                host: state.host,
                region: state.region,
            }),
        }
    )
)

// ============================================================================
// Health Check Poller - waits until /health returns 200
// ============================================================================

async function waitForHealth(serverUrl: string, adminKey?: string): Promise<void> {
    const start = Date.now()

    while (Date.now() - start < HEALTH_TIMEOUT) {
        try {
            const headers: Record<string, string> = {}
            if (adminKey) headers['Authorization'] = `Bearer ${adminKey}`

            const res = await fetch(`${serverUrl}/health`, {
                headers,
                signal: AbortSignal.timeout(5000),
            })
            if (res.ok) return // Success!
        } catch {
            // Ignore - server not ready yet
        }

        // Update step detail with elapsed time
        const elapsed = Math.round((Date.now() - start) / 1000)
        const store = useCloudStore.getState()
        const updatedSteps = store.steps.map(s =>
            s.id === 'health'
                ? { ...s, detail: `Waiting for server... (${elapsed}s)` }
                : s
        )
        useCloudStore.setState({ steps: updatedSteps })

        await new Promise(r => setTimeout(r, HEALTH_POLL_INTERVAL))
    }

    throw new Error('Server did not become healthy within 60 seconds')
}
