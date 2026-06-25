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
}
