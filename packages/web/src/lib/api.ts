const getToken = () => localStorage.getItem('token')

const request = async <T>(path: string, options?: RequestInit): Promise<T> => {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string> || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`/api${path}`, { ...options, headers })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export interface User {
  id: string
  username: string
  role: string
}

export interface Route {
  id: string
  name: string
  host: string
  path_prefix: string
  backend: string
  strip_prefix: boolean
  enabled: boolean
  priority: number
  created_at: string
  updated_at: string
}

export interface AuthRule {
  id: string
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic'
  config: { header_name?: string; secret?: string; username?: string; password?: string }
  whitelist: string[]
  rate_limit: number
  created_at: string
  updated_at: string
}

export interface LoginResponse {
  token: string
  user: User
  permissions: {
    CanManageRoutes: boolean
    CanManageAuth: boolean
    CanManageUsers: boolean
    CanViewLogs: boolean
  }
}

export const api = {
  auth: {
    login: (username: string, password: string) =>
      request<LoginResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      }),
    me: () => request<User>('/auth/me'),
    logout: () => request('/auth/logout', { method: 'POST' }),
  },
  routes: {
    list: () => request<Route[]>('/routes'),
    get: (id: string) => request<Route>(`/routes/${id}`),
    create: (data: Partial<Route>) => request<Route>('/routes', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Route>) => request<Route>(`/routes/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/routes/${id}`, { method: 'DELETE' }),
  },
  authRules: {
    list: () => request<AuthRule[]>('/auth-rules'),
    get: (id: string) => request<AuthRule>(`/auth-rules/${id}`),
    create: (data: Partial<AuthRule>) => request<AuthRule>('/auth-rules', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<AuthRule>) => request<AuthRule>(`/auth-rules/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/auth-rules/${id}`, { method: 'DELETE' }),
  },
  users: {
    list: () => request<User[]>('/users'),
    create: (data: { username: string; password: string; role: string }) =>
      request<User>('/users', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<User>) => request<User>(`/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request<void>(`/users/${id}`, { method: 'DELETE' }),
  },
  config: {
    reload: () => request<{ message: string }>('/config/reload'),
  },
}
