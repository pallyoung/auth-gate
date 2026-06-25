import { request } from './client'
import type {
  RouteAuthConfig,
  RouteAuthConfigInput,
  ApiKey,
  ApiKeyCreateResponse,
  ApiKeyCreateInput,
} from './types'

export const routeAuthApi = {
  getConfig: (routeId: string) =>
    request<RouteAuthConfig>(`/route-auth-config/${routeId}`),

  updateConfig: (routeId: string, data: RouteAuthConfigInput) =>
    request<RouteAuthConfig>(`/route-auth-config/${routeId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteConfig: (routeId: string) =>
    request<void>(`/route-auth-config/${routeId}`, { method: 'DELETE' }),
}

export const apiKeyApi = {
  list: (routeId: string) =>
    request<ApiKey[]>(`/route-api-keys/${routeId}`),

  create: (routeId: string, data: ApiKeyCreateInput) =>
    request<ApiKeyCreateResponse>(`/route-api-keys/${routeId}`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  updateName: (id: string, name: string) =>
    request<void>(`/api-keys/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name }),
    }),

  rotate: (id: string) =>
    request<ApiKeyCreateResponse>(`/api-keys/${id}/rotate`, { method: 'POST' }),

  expire: (id: string) =>
    request<void>(`/api-keys/${id}/expire`, { method: 'POST' }),

  delete: (id: string) =>
    request<void>(`/api-keys/${id}`, { method: 'DELETE' }),
}
