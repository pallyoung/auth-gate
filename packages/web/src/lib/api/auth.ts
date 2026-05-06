import { request } from './client'
import type { LoginResponse, SessionUser } from './types'

export const authApi = {
  login: (username: string, password: string) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<SessionUser>('/auth/me'),
  logout: () => request<{ message: string }>('/auth/logout', { method: 'POST' }),
}
