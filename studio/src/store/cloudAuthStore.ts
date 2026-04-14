/**
 * Cloud Auth Store
 *
 * Manages authentication for Hector Cloud.
 * Users paste their Fly.io API token; the cloud validates it and returns
 * a short-lived cloud JWT. The JWT is persisted in localStorage.
 */

import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { setCloudToken, clearCloudAuth } from '../services/cloudApi'

const CLOUD_API_URL = import.meta.env.VITE_HECTOR_CLOUD_URL || ''

// Decode a JWT payload without verification (expiry check only)
function getJWTExpiry(token: string): number {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.exp ? payload.exp * 1000 : 0
  } catch {
    return 0
  }
}

function isTokenExpired(token: string | null): boolean {
  if (!token) return true
  const exp = getJWTExpiry(token)
  if (!exp) return false // no expiry = treat as valid
  return Date.now() > exp - 5 * 60 * 1000 // 5 min buffer
}

interface CloudAuthState {
  // State
  isAuthenticated: boolean | null  // null = not checked yet
  token: string | null
  error: string | null

  // Actions
  loginWithToken: (flyToken: string, appName: string) => Promise<void>
  validate: () => boolean
  logout: () => void
  clearError: () => void
}

export const useCloudAuthStore = create<CloudAuthState>()(
  persist(
    (set, get) => ({
      isAuthenticated: null,
      token: null,
      error: null,

      loginWithToken: async (flyToken: string, appName: string) => {
        if (!CLOUD_API_URL) {
          set({ error: 'Cloud URL not configured' })
          return
        }
        set({ error: null })
        try {
          const res = await fetch(`${CLOUD_API_URL}/auth/token`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fly_token: flyToken, app_name: appName }),
          })
          if (!res.ok) {
            const text = await res.text()
            set({ error: text.trim() || 'Authentication failed' })
            return
          }
          const data = await res.json()
          setCloudToken(data.token)
          set({ token: data.token, isAuthenticated: true, error: null })
        } catch (e) {
          set({ error: 'Could not reach cloud service' })
        }
      },

      validate: () => {
        const { token } = get()
        if (!token || isTokenExpired(token)) {
          set({ isAuthenticated: false })
          return false
        }
        // Re-seed cloudApi cache on hydration/validate
        setCloudToken(token)
        set({ isAuthenticated: true })
        return true
      },

      logout: () => {
        clearCloudAuth()
        set({ isAuthenticated: false, token: null, error: null })
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: 'hector_cloud_auth',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        token: state.token,
      }),
      onRehydrateStorage: () => (state) => {
        // Re-seed the cloudApi token cache when the store is hydrated from localStorage
        if (state?.token && !isTokenExpired(state.token)) {
          setCloudToken(state.token)
        } else if (state) {
          state.isAuthenticated = false
          state.token = null
        }
      },
    }
  )
)

