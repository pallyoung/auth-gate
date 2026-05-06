import { listResource, request } from './client'
import type { User } from './types'

export const usersApi = {
  list: () => listResource<User>('/users'),
  create: (data: { username: string; password: string; role: string }) =>
    request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Partial<User>) => request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/users/${id}`, { method: 'DELETE' }),
}
