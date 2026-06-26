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
  // Route type: "proxy" (default) or "static"
  type?: 'proxy' | 'static'
  static_root?: string
  static_spa?: boolean
  tls_cert?: string
  tls_key?: string
  tls_enabled?: boolean
  https_redirect?: boolean
  certificate_id?: string
  timeout_ms?: number
  retry_attempts?: number
  path_match_mode?: string
  header_name?: string
  header_value?: string
  rewrite_target?: string
  redirect_code?: number
  // Header manipulation
  set_request_headers?: Record<string, string>
  remove_request_headers?: string[]
  add_response_headers?: Record<string, string>
  remove_response_headers?: string[]
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
  // Route type: "proxy" (default) or "static"
  type?: 'proxy' | 'static'
  static_root?: string
  static_spa?: boolean
  tls_cert?: string
  tls_key?: string
  tls_enabled?: boolean
  https_redirect?: boolean
  certificate_id?: string
  timeout_ms?: number
  retry_attempts?: number
  path_match_mode?: string
  header_name?: string
  header_value?: string
  rewrite_target?: string
  redirect_code?: number
  // Header manipulation
  set_request_headers?: Record<string, string>
  remove_request_headers?: string[]
  add_response_headers?: Record<string, string>
  remove_response_headers?: string[]
}

export interface AuthRuleConfig {
  header_name?: string
  secret?: string
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
  type: 'none' | 'apikey' | 'gateway'
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
  type: 'none' | 'apikey' | 'gateway'
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

// ---- New: Route Auth Config ----

export interface RouteAuthConfig {
  route_id: string
  api_key_enabled: boolean
  api_key_header?: string
  gateway_enabled: boolean
  gateway_login_mode?: string
  whitelist?: string[]
  rate_limit?: number
  burst?: number
  cors_allowed_origins?: string
  cors_allowed_methods?: string
  cors_allowed_headers?: string
  cors_allow_credentials?: boolean
  cors_max_age?: number
}

export interface RouteAuthConfigInput {
  api_key_enabled?: boolean
  api_key_header?: string
  gateway_enabled?: boolean
  gateway_login_mode?: string
  whitelist?: string[]
  rate_limit?: number
  burst?: number
  cors_allowed_origins?: string
  cors_allowed_methods?: string
  cors_allowed_headers?: string
  cors_allow_credentials?: boolean
  cors_max_age?: number
}

// ---- New: API Keys ----

export interface ApiKey {
  id: string
  route_id: string
  name: string
  key_prefix: string
  secret?: string
  expires_at?: string
  status: 'active' | 'expired' | 'revoked'
  last_used_at?: string
  created_at: string
}

export interface ApiKeyCreateResponse extends ApiKey {
  secret: string
}

export interface ApiKeyCreateInput {
  name?: string
  expires_at?: string
}

export interface LoginResponse {
  token: string
  user: SessionUser
  permissions: Permissions
}

export interface SetupStatusResponse {
  setup_required: boolean
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
  source: string
  ca_id?: string
  status: 'active' | 'failed'
  not_before?: string
  not_after?: string
  renew_at?: string
  created_at: string
  updated_at: string
}

export interface CertificateInput {
  name: string
  domain: string
  source?: string
  cert_pem?: string
  key_pem?: string
  organization?: string
  organizational_unit?: string
  country?: string
  province?: string
  locality?: string
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

export interface AccessLogEntry {
  request_id: string
  route_id: string
  route_name?: string
  method: string
  path: string
  backend_url: string
  backend_latency_ms: number
  status_code: number
  client_ip: string
  user_agent: string
  username?: string
  auth_result: 'pass' | 'fail' | 'none'
  timestamp: string
}

export interface AccessLogListResponse {
  entries: AccessLogEntry[]
  total: number
  page: number
  per_page: number
  total_pages: number
}

export interface AccessLogStats {
  total_requests: number
  success_count: number
  error_count: number
  avg_latency_ms: number
  p95_latency_ms: number
  requests_per_minute: TimeBucket[]
  error_rate_per_hour: TimeBucket[]
  latency_per_hour: LatencyBucket[]
  top_paths: PathCount[]
  top_ips: IPCount[]
}

export interface TimeBucket {
  time: string
  count: number
}

export interface LatencyBucket {
  time: string
  avg_ms: number
  p95_ms: number
}

export interface PathCount {
  path: string
  count: number
}

export interface IPCount {
  ip: string
  count: number
}

export interface AccessLogQueryParams {
  client_ip?: string
  path?: string
  username?: string
  auth_result?: string
  route_id?: string
  status_code?: number
  start_time?: string
  end_time?: string
  page?: number
  per_page?: number
}

export interface ListenEntry {
  addr: string
  tls: boolean
}

export interface ServerConfig {
  listen: ListenEntry[]
}
