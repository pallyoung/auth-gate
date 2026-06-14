export interface Permissions {
  can_manage_routes: boolean
  can_manage_auth: boolean
  can_manage_users: boolean
  can_view_logs: boolean
  can_manage_hosts: boolean
}

export interface Features {
  certificates: boolean
}

export interface SessionUser {
  id: string
  username: string
  role: string
  route_ids?: string[]
  permissions?: Permissions
  features?: Features
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
  backends?: RouteBackend[]
  strip_prefix: boolean
  enabled: boolean
  priority: number
  tls_cert?: string
  tls_key?: string
  tls_enabled?: boolean
  timeout_ms?: number
  retry_attempts?: number
  path_match_mode?: string
  rewrite_target?: string
  redirect_code?: number
  created_at: string
  updated_at: string
}

export interface RouteBackend {
  url: string
  weight: number
  dial_timeout_ms?: number
  read_timeout_ms?: number
  write_timeout_ms?: number
  max_idle_conns?: number
  rewrite_target?: string
  redirect_code?: number
}

export interface RouteInput {
  name?: string
  host?: string
  path_prefix: string
  backend: string
  backends?: RouteBackend[]
  strip_prefix: boolean
  enabled: boolean
  priority: number
  tls_cert?: string
  tls_key?: string
  tls_enabled?: boolean
  timeout_ms?: number
  retry_attempts?: number
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
  whitelist?: string[]
  rate_limit?: number
  burst?: number
  cors_allowed_origins?: string
  cors_allowed_methods?: string
  cors_allowed_headers?: string
  cors_allow_credentials?: boolean
  cors_max_age?: number
  created_at: string
  updated_at: string
}

export interface AuthRuleInput {
  route_id: string
  type: 'none' | 'apikey' | 'bearer' | 'basic' | 'gateway'
  config: AuthRuleSecretConfig
  whitelist?: string[]
  rate_limit?: number
  burst?: number
  cors_allowed_origins?: string
  cors_allowed_methods?: string
  cors_allowed_headers?: string
  cors_allow_credentials?: boolean
  cors_max_age?: number
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

export interface Certificate {
  id: string
  name: string
  domain: string
  cert_path: string
  key_path: string
  dns_provider: string
  status: 'pending' | 'active' | 'renewing' | 'failed'
  not_before?: string
  not_after?: string
  renew_at?: string
  created_at: string
  updated_at: string
}

export interface CertificateInput {
  name: string
  domain: string
  dns_provider: string
  provider_config: Record<string, string>
}

export interface HostProfile {
  id: string
  name: string
  description: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface HostEntry {
  id: string
  profile_id: string
  position: number
  ip: string
  hostnames: string
  comment: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface HostProfileListResponse {
  profiles: HostProfile[]
  active_id: string
}

export interface HostProfileInput {
  name: string
  description: string
}

export interface HostEntryInput {
  ip: string
  comment: string
  hostnames: string[]
  enabled: boolean
}

export interface HostEntryReorderInput {
  entry_ids: string[]
}
