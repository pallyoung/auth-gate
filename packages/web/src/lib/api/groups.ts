import { listResource, request } from './client'
import type { PermissionGroup, PermissionGroupInput } from './types'

export const permissionGroupsApi = {
  list: () => listResource<PermissionGroup>('/permission-groups'),
  get: (id: string) => request<PermissionGroup>(`/permission-groups/${id}`),
  create: (data: PermissionGroupInput) =>
    request<PermissionGroup>('/permission-groups', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: PermissionGroupInput) =>
    request<PermissionGroup>(`/permission-groups/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/permission-groups/${id}`, { method: 'DELETE' }),
}
