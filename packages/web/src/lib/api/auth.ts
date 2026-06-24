import { publicRequest, request } from './client'
import type { LoginResponse, RouteAccessLoginResponse, SessionUser, SetupStatusResponse } from './types'

export const authApi = {
  login: (username: string, password: string) =>
    publicRequest<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  setupStatus: () =>
    publicRequest<SetupStatusResponse>('/auth/setup-status'),
  setup: (username: string, password: string) =>
    publicRequest<LoginResponse>('/auth/setup', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  accessLogin: (payload: { route_id: string; username: string; password: string; next?: string }) =>
    publicRequest<RouteAccessLoginResponse>('/access/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  accessLogout: () => publicRequest<{ message: string }>('/access/logout', { method: 'POST' }),
  me: () => request<SessionUser>('/auth/me'),
  logout: () => request<{ message: string }>('/auth/logout', { method: 'POST' }),
}
