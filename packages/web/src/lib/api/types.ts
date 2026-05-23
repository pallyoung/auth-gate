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
  route_ids?: string[]
  permissions?: Permissions
}

export interface User {
  id: string
  username: string
  role: string
  enabled?: boolean
  route_ids?: string[]
  created_at?: string
  updated_at?: string
}

export interface UserInput {
  username: string
  password?: string
  role: string
  enabled: boolean
  route_ids: string[]
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
  path_match_mode?: string
  rewrite_target?: string
  redirect_code?: number
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
  path_match_mode?: string
  rewrite_target?: string
  redirect_code?: number
}

export interface AuthRuleConfig {
  header_name?: string
  username?: string
  login_mode?: string
}

export interface AuthRuleSecretConfig extends AuthRuleConfig {
  secret?: string
  password?: string
}

export interface AuthRule {
  id: string
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic' | 'gateway'
  config: AuthRuleConfig
  created_at: string
  updated_at: string
}

export interface AuthRuleInput {
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic' | 'gateway'
  config: AuthRuleSecretConfig
}

export interface LoginResponse {
  token: string
  user: SessionUser
  permissions: Permissions
}

export interface RouteAccessLoginResponse {
  next: string
  user: SessionUser
}

export interface ApiErrorEnvelope {
  error?: {
    code?: string
    message?: string
  }
}
