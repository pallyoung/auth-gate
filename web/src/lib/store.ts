import { create } from 'zustand'
import { api, Route, AuthRule } from './api'

interface Store {
  routes: Route[]
  authRules: AuthRule[]
  loading: boolean
  error: string | null
  fetchRoutes: () => Promise<void>
  fetchAuthRules: () => Promise<void>
  createRoute: (data: Partial<Route>) => Promise<void>
  updateRoute: (id: string, data: Partial<Route>) => Promise<void>
  deleteRoute: (id: string) => Promise<void>
  createAuthRule: (data: Partial<AuthRule>) => Promise<void>
  updateAuthRule: (id: string, data: Partial<AuthRule>) => Promise<void>
  deleteAuthRule: (id: string) => Promise<void>
}

export const useStore = create<Store>((set, get) => ({
  routes: [],
  authRules: [],
  loading: false,
  error: null,

  fetchRoutes: async () => {
    set({ loading: true, error: null })
    try {
      const routes = await api.listRoutes()
      set({ routes, loading: false })
    } catch (e) {
      set({ error: (e as Error).message, loading: false })
    }
  },

  fetchAuthRules: async () => {
    try {
      const authRules = await api.listAuthRules()
      set({ authRules })
    } catch (e) {
      set({ error: (e as Error).message })
    }
  },

  createRoute: async (data) => {
    const route = await api.createRoute(data)
    set({ routes: [...get().routes, route] })
  },

  updateRoute: async (id, data) => {
    const route = await api.updateRoute(id, data)
    set({ routes: get().routes.map(r => r.id === id ? route : r) })
  },

  deleteRoute: async (id) => {
    await api.deleteRoute(id)
    set({ routes: get().routes.filter(r => r.id !== id) })
  },

  createAuthRule: async (data) => {
    const rule = await api.createAuthRule(data)
    set({ authRules: [...get().authRules, rule] })
  },

  updateAuthRule: async (id, data) => {
    const rule = await api.updateAuthRule(id, data)
    set({ authRules: get().authRules.map(r => r.id === id ? rule : r) })
  },

  deleteAuthRule: async (id) => {
    await api.deleteAuthRule(id)
    set({ authRules: get().authRules.filter(r => r.id !== id) })
  },
}))
