import { request } from './client'
import type { ListenEntry, ServerConfig } from './types'

export const configApi = {
  reload: () => request<{ message: string }>('/config/reload', { method: 'POST' }),
  get: () => request<ServerConfig>('/config'),
  update: (listen: ListenEntry[]) =>
    request<{ message: string }>('/config', {
      method: 'PUT',
      body: JSON.stringify({ listen }),
    }),
  getLogRetention: () => request<{ days: number }>('/settings/log-retention'),
  updateLogRetention: (days: number) =>
    request<{ days: number }>('/settings/log-retention', {
      method: 'PUT',
      body: JSON.stringify({ days }),
    }),
}
