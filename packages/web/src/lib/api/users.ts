import { listResource, request } from './client'
import type { User, UserInput } from './types'

export const usersApi = {
  list: () => listResource<User>('/users'),
  create: (data: UserInput) =>
    request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: UserInput) => request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/users/${id}`, { method: 'DELETE' }),
}
