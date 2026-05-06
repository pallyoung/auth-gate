import { listResource, request } from './client'
import type { AuthRule, AuthRuleInput } from './types'

export const authRulesApi = {
  list: () => listResource<AuthRule>('/auth-rules'),
  get: (id: string) => request<AuthRule>(`/auth-rules/${id}`),
  create: (data: AuthRuleInput) => request<AuthRule>('/auth-rules', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: AuthRuleInput) => request<AuthRule>(`/auth-rules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/auth-rules/${id}`, { method: 'DELETE' }),
}
