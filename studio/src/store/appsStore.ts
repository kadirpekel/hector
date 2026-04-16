/**
 * Apps Store
 *
 * Manages apps scoped per server with JWT token storage.
 */

import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import type { AppToken } from '../types'
import { apiFetch } from '../lib/api-utils'

const EMPTY_APPS: App[] = []

export interface App {
  id: string
  name: string
  config_json?: string
  created_at: string
  updated_at: string
}

interface ServerAppsData {
  apps: App[]
  activeAppId: string | null
  appTokens: Record<string, AppToken>
}

interface AppsState {
  appsByServer: Record<string, ServerAppsData>
  loading: boolean
  error: string | null

  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  getAppsForServer: (serverId: string) => App[]
  getActiveAppId: (serverId: string) => string | null
  getActiveApp: (serverId: string) => App | null
  getAppToken: (serverId: string, appId: string) => AppToken | null

  loadApps: (serverId: string) => Promise<void>
  createApp: (serverId: string, name: string, config: string) => Promise<App>
  updateApp: (serverId: string, appId: string, name?: string, config?: string) => Promise<App>
  deleteApp: (serverId: string, appId: string) => Promise<void>
  selectApp: (serverId: string, appId: string) => void

  storeAppToken: (serverId: string, appId: string, token: AppToken) => void
  regenerateToken: (serverId: string, appId: string) => Promise<void>
  clearServerTokens: (serverId: string) => void
}

export const useAppsStore = create<AppsState>()(
  persist(
    (set, get) => ({
      appsByServer: {},
      loading: false,
      error: null,

      setLoading: (loading) => set({ loading }),
      setError: (error) => set({ error }),

      getAppsForServer: (serverId) => {
        return get().appsByServer[serverId]?.apps || EMPTY_APPS
      },

      getActiveAppId: (serverId) => {
        return get().appsByServer[serverId]?.activeAppId || null
      },

      getActiveApp: (serverId) => {
        const serverData = get().appsByServer[serverId]
        if (!serverData?.activeAppId) return null
        return serverData.apps.find((a) => a.id === serverData.activeAppId) || null
      },

      getAppToken: (serverId, appId) => {
        return get().appsByServer[serverId]?.appTokens[appId] || null
      },

      loadApps: async (serverId) => {
        set({ loading: true, error: null })
        try {
          const res = await apiFetch('/admin/apps')
          if (!res.ok) throw new Error(`Failed to list apps: ${res.status}`)
          const data = await res.json()
          const apps: App[] = Array.isArray(data) ? data : (data.apps ?? [])

          const currentActive = get().appsByServer[serverId]?.activeAppId
          let activeAppId = currentActive && apps.find(a => a.id === currentActive)
            ? currentActive
            : null

          if (!activeAppId && apps.length > 0) {
            const defaultApp = apps.find(a => a.id === 'default')
            activeAppId = defaultApp?.id || apps[0].id
          }

          set((state) => ({
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                apps,
                activeAppId,
                appTokens: state.appsByServer[serverId]?.appTokens || {}
              }
            },
            loading: false
          }))

          // Ensure active app has a token
          if (activeAppId) {
            const existingToken = get().appsByServer[serverId]?.appTokens[activeAppId]
            if (!existingToken?.accessToken) {
              try {
                await get().regenerateToken(serverId, activeAppId)
              } catch (tokenErr) {
                console.error('[appsStore] Failed to fetch token for active app:', tokenErr)
              }
            }
          }
        } catch (e) {
          set({ error: (e as Error).message, loading: false })
        }
      },

      createApp: async (serverId, name, config) => {
        const res = await apiFetch('/admin/apps', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name, config_json: config }),
        })
        if (!res.ok) throw new Error(`Failed to create app: ${res.status}`)
        const response = await res.json()

        const app = response.app || response
        const hasToken = response.access_token

        set((state) => {
          const serverData = state.appsByServer[serverId] || {
            apps: [],
            activeAppId: null,
            appTokens: {}
          }

          const newTokens = hasToken ? {
            ...serverData.appTokens,
            [app.id]: {
              appId: app.id,
              accessToken: response.access_token,
              tokenType: response.token_type,
              issuer: response.issuer
            }
          } : serverData.appTokens

          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                apps: [...serverData.apps, app],
                activeAppId: app.id,
                appTokens: newTokens
              }
            }
          }
        })

        return app
      },

      updateApp: async (serverId, appId, name, config) => {
        const body: Record<string, string | undefined> = {}
        if (name !== undefined) body.name = name
        if (config !== undefined) body.config_json = config

        const res = await apiFetch(`/admin/apps/${appId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        })
        if (!res.ok) throw new Error(`Failed to update app: ${res.status}`)
        const app = await res.json()

        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state
          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                ...serverData,
                apps: serverData.apps.map((a) => (a.id === appId ? app : a))
              }
            }
          }
        })

        return app
      },

      deleteApp: async (serverId, appId) => {
        const res = await apiFetch(`/admin/apps/${appId}`, { method: 'DELETE' })
        if (!res.ok && res.status !== 404) throw new Error(`Failed to delete app: ${res.status}`)

        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state

          const { [appId]: _removed, ...remainingTokens } = serverData.appTokens
          const remainingApps = serverData.apps.filter((a) => a.id !== appId)

          let newActiveAppId = serverData.activeAppId
          if (serverData.activeAppId === appId) {
            const defaultApp = remainingApps.find((a) => a.id === 'default')
            newActiveAppId = defaultApp?.id || remainingApps[0]?.id || null
          }

          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                apps: remainingApps,
                activeAppId: newActiveAppId,
                appTokens: remainingTokens
              }
            }
          }
        })
      },

      selectApp: (serverId, appId) => {
        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state
          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                ...serverData,
                activeAppId: appId
              }
            }
          }
        })

        // Trigger agent reload and restore the last session for this app
        import('./useStore').then(async ({ useStore }) => {
          // Await so selectedAgent is set for the new app before we pick a session
          await useStore.getState().reloadAgents()

          const sessions = useStore.getState().sessions
          const agentName = useStore.getState().selectedAgent?.name
          const appSessions = Object.values(sessions)
            .filter(s => s.serverId === serverId && s.appId === appId && (!agentName || s.agentName === agentName || !s.agentName))
            .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime())
          if (appSessions.length > 0) {
            useStore.getState().selectSession(appSessions[0].id)
            useStore.getState().loadSessionEvents(appSessions[0].id)
          } else {
            useStore.getState().createSession()
          }
          useStore.getState().syncSessions()
        })
      },

      storeAppToken: (serverId, appId, token) => {
        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state
          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                ...serverData,
                appTokens: {
                  ...serverData.appTokens,
                  [appId]: token
                }
              }
            }
          }
        })
      },

      regenerateToken: async (serverId, appId) => {
        const res = await apiFetch(`/admin/apps/${appId}/token`, { method: 'POST' })
        if (!res.ok) throw new Error(`Failed to regenerate token: ${res.status}`)
        const response = await res.json()

        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state
          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                ...serverData,
                appTokens: {
                  ...serverData.appTokens,
                  [appId]: {
                    appId,
                    accessToken: response.access_token,
                    tokenType: response.token_type,
                    issuer: response.issuer || 'hector'
                  }
                }
              }
            }
          }
        })
      },

      clearServerTokens: (serverId) => {
        set((state) => {
          const serverData = state.appsByServer[serverId]
          if (!serverData) return state
          return {
            appsByServer: {
              ...state.appsByServer,
              [serverId]: {
                ...serverData,
                appTokens: {}
              }
            }
          }
        })
      }
    }),
    {
      name: 'hector_apps_store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        appsByServer: Object.entries(state.appsByServer).reduce((acc, [serverId, data]) => {
          acc[serverId] = {
            apps: [],
            activeAppId: data.activeAppId,
            appTokens: data.appTokens
          }
          return acc
        }, {} as Record<string, ServerAppsData>)
      })
    }
  )
)
