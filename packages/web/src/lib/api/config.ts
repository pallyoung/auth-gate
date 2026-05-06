import { request } from './client'

export const configApi = {
  reload: () => request<{ message: string }>('/config/reload', { method: 'POST' }),
}
