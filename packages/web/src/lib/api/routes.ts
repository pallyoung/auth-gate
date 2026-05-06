import { listResource, request } from './client'
import type { Route, RouteInput } from './types'

export const routesApi = {
  list: () => listResource<Route>('/routes'),
  get: (id: string) => request<Route>(`/routes/${id}`),
  create: (data: RouteInput) => request<Route>('/routes', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: RouteInput) => request<Route>(`/routes/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/routes/${id}`, { method: 'DELETE' }),
}
