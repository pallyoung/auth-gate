export interface Permissions {
  can_manage_routes: boolean
  can_manage_auth: boolean
  can_manage_users: boolean
  can_view_logs: boolean
}

export interface SessionUser {
  id: string
  username: string
  role: string
  permissions?: Permissions
}

export interface User {
  id: string
  username: string
  role: string
  enabled?: boolean
  created_at?: string
  updated_at?: string
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

export interface RouteInput {
  name?: string
  host?: string
  path_prefix: string
  backend: string
  strip_prefix: boolean
  enabled: boolean
  priority: number
}

export interface AuthRuleConfig {
  header_name?: string
  username?: string
}

export interface AuthRuleSecretConfig extends AuthRuleConfig {
  secret?: string
  password?: string
}

export interface AuthRule {
  id: string
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic'
  config: AuthRuleConfig
  created_at: string
  updated_at: string
}

export interface AuthRuleInput {
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic'
  config: AuthRuleSecretConfig
}

export interface LoginResponse {
  token: string
  user: SessionUser
  permissions: Permissions
}

export interface ApiErrorEnvelope {
  error?: {
    code?: string
    message?: string
  }
}
