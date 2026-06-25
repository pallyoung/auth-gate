import React from 'react'
import {
  ChevronDown,
  Copy,
  KeyRound,
  LockKeyhole,
  Plus,
  RefreshCw,
  Shield,
  Trash2,
  XCircle,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, Input, MetricCard, Modal, Select, Switch } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { routeAuthApi, apiKeyApi } from '../lib/api/route-auth'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { ApiKey, ApiKeyCreateInput, Route, RouteAuthConfig, RouteAuthConfigInput } from '../lib/api/types'

const EXPIRATION_OPTIONS = [
  { value: '', label: '永不过期' },
  { value: '7d', label: '7 天' },
  { value: '30d', label: '30 天' },
  { value: '90d', label: '90 天' },
  { value: '1y', label: '1 年' },
]

function parseExpiration(value: string): string | undefined {
  if (!value) return undefined
  const now = new Date()
  switch (value) {
    case '7d': now.setDate(now.getDate() + 7); break
    case '30d': now.setDate(now.getDate() + 30); break
    case '90d': now.setDate(now.getDate() + 90); break
    case '1y': now.setFullYear(now.getFullYear() + 1); break
    default: return undefined
  }
  return now.toISOString()
}

function formatDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}

function keyStatusBadge(status: string) {
  switch (status) {
    case 'active':
      return <Badge variant="success" badgeSize="sm">有效</Badge>
    case 'revoked':
      return <Badge variant="error" badgeSize="sm">已吊销</Badge>
    default:
      return <Badge variant="warning" badgeSize="sm">{status}</Badge>
  }
}

export function RouteAuthPage() {
  const { t } = useTranslation(['authRules', 'common'])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [selectedRouteId, setSelectedRouteId] = React.useState<string>('')
  const [authConfig, setAuthConfig] = React.useState<RouteAuthConfig | null>(null)
  const [apiKeys, setApiKeys] = React.useState<ApiKey[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [showCreateKey, setShowCreateKey] = React.useState(false)
  const [newKeyName, setNewKeyName] = React.useState('')
  const [newKeyExpiration, setNewKeyExpiration] = React.useState('')
  const [createdSecret, setCreatedSecret] = React.useState<string | null>(null)
  const [copiedSecret, setCopiedSecret] = React.useState(false)
  const [keyMenuOpen, setKeyMenuOpen] = React.useState<string | null>(null)
  const canManageAuth = getSessionUser()?.permissions?.can_manage_auth ?? false
  const errorMessage = resolveLocalizedText(t, error)

  const loadRoutes = React.useCallback(async () => {
    try {
      const list = await routesApi.list()
      setRoutes(list)
      if (list.length > 0 && !selectedRouteId) {
        setSelectedRouteId(list[0].id)
      }
    } catch {
      // ignore
    }
  }, [selectedRouteId])

  const loadAuthData = React.useCallback(async (routeId: string) => {
    if (!routeId) return
    setLoading(true)
    try {
      const [config, keys] = await Promise.all([
        routeAuthApi.getConfig(routeId),
        apiKeyApi.list(routeId),
      ])
      setAuthConfig(config)
      setApiKeys(keys)
      setError(null)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => { loadRoutes() }, [])
  React.useEffect(() => { if (selectedRouteId) loadAuthData(selectedRouteId) }, [selectedRouteId, loadAuthData])

  const updateConfig = async (patch: RouteAuthConfigInput) => {
    if (!selectedRouteId) return
    try {
      const updated = await routeAuthApi.updateConfig(selectedRouteId, patch)
      setAuthConfig(updated)
      setError(null)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    }
  }

  const handleCreateKey = async () => {
    if (!selectedRouteId || !newKeyName.trim()) return
    try {
      const resp = await apiKeyApi.create(selectedRouteId, {
        name: newKeyName.trim(),
        expires_at: parseExpiration(newKeyExpiration),
      })
      setCreatedSecret(resp.secret)
      setCopiedSecret(false)
      setNewKeyName('')
      setNewKeyExpiration('')
      await loadAuthData(selectedRouteId)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    }
  }

  const handleRotateKey = async (id: string) => {
    try {
      const resp = await apiKeyApi.rotate(id)
      setCreatedSecret(resp.secret)
      setCopiedSecret(false)
      await loadAuthData(selectedRouteId)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    }
  }

  const handleExpireKey = async (id: string) => {
    try {
      await apiKeyApi.expire(id)
      await loadAuthData(selectedRouteId)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    }
  }

  const handleDeleteKey = async (id: string) => {
    if (!confirm('确认删除此 API Key？')) return
    try {
      await apiKeyApi.delete(id)
      await loadAuthData(selectedRouteId)
    } catch (err) {
      setError({ translationKey: 'errors.authRuleStoreFailure' })
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedSecret(true)
  }

  const cfg = authConfig || {
    route_id: selectedRouteId,
    api_key_enabled: false,
    basic_enabled: false,
    gateway_enabled: false,
  }

  if (routes.length === 0 && !loading) {
    return (
      <div className="animate-rise-in">
        <PageHeader
          eyebrow="鉴权管理"
          title="路由鉴权配置"
          description="为每个路由配置认证方式和 API Key。"
        />
        <EmptyState
          icon={<Shield className="h-8 w-8" />}
          title="暂无路由"
          description="请先创建路由，再配置鉴权规则。"
        />
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="鉴权管理"
        title="路由鉴权配置"
        description="为每个路由配置认证方式。多种方式同时启用时，任一通过即放行。"
        meta={<Badge variant="primary">鉴权配置</Badge>}
      />

      {errorMessage && (
        <Alert variant="error" title="操作失败" className="mb-5">{errorMessage}</Alert>
      )}

      {/* 路由选择 */}
      <Card tone="soft" className="mb-6">
        <Select
          label="选择路由"
          value={selectedRouteId}
          onChange={(e) => setSelectedRouteId(e.target.value)}
          options={routes.map((r) => ({ value: r.id, label: r.name || r.path_prefix }))}
        />
      </Card>

      {loading ? (
        <div className="flex h-32 items-center justify-center text-[var(--text-muted)]">加载中...</div>
      ) : (
        <div className="space-y-6">
          {/* API Key 认证 */}
          <Card tone="soft" className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <KeyRound className="h-5 w-5 text-[var(--text-muted)]" />
                <div>
                  <div className="font-semibold text-[var(--text-primary)]">API Key 认证</div>
                  <div className="text-xs text-[var(--text-muted)]">通过请求头或查询参数传递 API Key</div>
                </div>
              </div>
              {canManageAuth && (
                <Switch
                  checked={cfg.api_key_enabled}
                  onChange={(e) => updateConfig({ api_key_enabled: e.target.checked })}
                />
              )}
            </div>

            {cfg.api_key_enabled && (
              <div className="space-y-4 border-t border-[var(--border-default)] pt-4">
                {canManageAuth && (
                  <Input
                    label="Header 名称"
                    value={cfg.api_key_header || ''}
                    onChange={(e) => updateConfig({ api_key_header: e.target.value })}
                    placeholder="X-API-Key"
                  />
                )}

                {/* API Key 列表 */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="text-sm font-medium text-[var(--text-primary)]">API Key 列表</div>
                    {canManageAuth && (
                      <Button size="sm" variant="ghost" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowCreateKey(true)}>
                        新建 Key
                      </Button>
                    )}
                  </div>

                  {apiKeys.length === 0 ? (
                    <div className="rounded-lg border border-dashed border-[var(--border-default)] p-4 text-center text-sm text-[var(--text-muted)]">
                      暂无 API Key，点击上方按钮创建
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {apiKeys.map((key) => (
                        <div key={key.id} className="flex items-center gap-3 rounded-lg border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-3">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-sm text-[var(--text-primary)]">{key.name}</span>
                              {keyStatusBadge(key.status)}
                            </div>
                            <div className="mt-1 flex items-center gap-3 text-xs text-[var(--text-muted)]">
                              <code className="rounded bg-[var(--surface-inset)] px-1.5 py-0.5">{key.key_prefix}...</code>
                              <span>过期: {formatDate(key.expires_at)}</span>
                              {key.last_used_at && <span>最后使用: {formatDate(key.last_used_at)}</span>}
                            </div>
                          </div>
                          {canManageAuth && key.status === 'active' && (
                            <div className="relative">
                              <Button variant="ghost" size="sm" onClick={() => setKeyMenuOpen(keyMenuOpen === key.id ? null : key.id)}>
                                <ChevronDown className="h-4 w-4" />
                              </Button>
                              {keyMenuOpen === key.id && (
                                <div className="absolute right-0 top-full z-10 mt-1 min-w-[140px] rounded-lg border border-[var(--border-default)] bg-[var(--bg-card)] py-1 shadow-lg">
                                  <button className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--surface-inset)]" onClick={() => { handleRotateKey(key.id); setKeyMenuOpen(null) }}>
                                    <RefreshCw className="h-3.5 w-3.5" /> 轮换 Key
                                  </button>
                                  <button className="flex w-full items-center gap-2 px-3 py-2 text-sm text-[var(--text-error)] hover:bg-[var(--surface-inset)]" onClick={() => { handleExpireKey(key.id); setKeyMenuOpen(null) }}>
                                    <XCircle className="h-3.5 w-3.5" /> 吊销
                                  </button>
                                  <button className="flex w-full items-center gap-2 px-3 py-2 text-sm text-[var(--text-error)] hover:bg-[var(--surface-inset)]" onClick={() => { handleDeleteKey(key.id); setKeyMenuOpen(null) }}>
                                    <Trash2 className="h-3.5 w-3.5" /> 删除
                                  </button>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}
          </Card>

          {/* Basic Auth */}
          <Card tone="soft" className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <LockKeyhole className="h-5 w-5 text-[var(--text-muted)]" />
                <div>
                  <div className="font-semibold text-[var(--text-primary)]">Basic Auth</div>
                  <div className="text-xs text-[var(--text-muted)]">HTTP Basic 认证（用户名 + 密码）</div>
                </div>
              </div>
              {canManageAuth && (
                <Switch
                  checked={cfg.basic_enabled}
                  onChange={(e) => updateConfig({ basic_enabled: e.target.checked })}
                />
              )}
            </div>
            {cfg.basic_enabled && canManageAuth && (
              <div className="grid grid-cols-1 gap-4 border-t border-[var(--border-default)] pt-4 md:grid-cols-2">
                <Input
                  label="用户名"
                  value={cfg.basic_username || ''}
                  onChange={(e) => updateConfig({ basic_username: e.target.value })}
                  placeholder="admin"
                />
                <Input
                  label="密码"
                  type="password"
                  value=""
                  onChange={(e) => updateConfig({ basic_password: e.target.value })}
                  placeholder="留空则保持不变"
                />
              </div>
            )}
          </Card>

          {/* 网关登录 */}
          <Card tone="soft" className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Shield className="h-5 w-5 text-[var(--text-muted)]" />
                <div>
                  <div className="font-semibold text-[var(--text-primary)]">网关登录</div>
                  <div className="text-xs text-[var(--text-muted)]">通过登录页面进行用户认证</div>
                </div>
              </div>
              {canManageAuth && (
                <Switch
                  checked={cfg.gateway_enabled}
                  onChange={(e) => updateConfig({ gateway_enabled: e.target.checked })}
                />
              )}
            </div>
          </Card>

          {/* 速率限制 */}
          <Card tone="soft" className="space-y-4">
            <div className="text-sm font-medium text-[var(--text-primary)]">速率限制</div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <Input
                label="限流 (次/秒)"
                type="number"
                min={0}
                value={cfg.rate_limit ?? ''}
                onChange={(e) => updateConfig({ rate_limit: e.target.value === '' ? undefined : Number(e.target.value) })}
                placeholder="不限制"
              />
              <Input
                label="突发上限"
                type="number"
                min={0}
                value={cfg.burst ?? ''}
                onChange={(e) => updateConfig({ burst: e.target.value === '' ? undefined : Number(e.target.value) })}
                placeholder="不限制"
              />
              <Input
                label="白名单"
                value={(cfg.whitelist || []).join(', ')}
                onChange={(e) => updateConfig({ whitelist: e.target.value.split(/[\n,]/).map(s => s.trim()).filter(Boolean) })}
                placeholder="127.0.0.1/32, 10.0.0.0/8"
              />
            </div>
          </Card>

          {/* CORS */}
          <Card tone="soft" className="space-y-4">
            <div className="text-sm font-medium text-[var(--text-primary)]">CORS 配置</div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input
                label="允许来源"
                value={cfg.cors_allowed_origins || ''}
                onChange={(e) => updateConfig({ cors_allowed_origins: e.target.value })}
                placeholder="https://app.example.com, *"
              />
              <Input
                label="允许方法"
                value={cfg.cors_allowed_methods || ''}
                onChange={(e) => updateConfig({ cors_allowed_methods: e.target.value })}
                placeholder="GET,POST,OPTIONS"
              />
              <Input
                label="允许请求头"
                value={cfg.cors_allowed_headers || ''}
                onChange={(e) => updateConfig({ cors_allowed_headers: e.target.value })}
                placeholder="Authorization,Content-Type"
              />
              <Input
                label="缓存时长 (秒)"
                type="number"
                min={0}
                value={cfg.cors_max_age ?? ''}
                onChange={(e) => updateConfig({ cors_max_age: e.target.value === '' ? undefined : Number(e.target.value) })}
                placeholder="86400"
              />
            </div>
            <Switch
              label="允许携带凭证"
              description="返回 Access-Control-Allow-Credentials"
              checked={cfg.cors_allow_credentials || false}
              onChange={(e) => updateConfig({ cors_allow_credentials: e.target.checked })}
            />
          </Card>
        </div>
      )}

      {/* 新建 Key Modal */}
      <Modal open={showCreateKey} onClose={() => { setShowCreateKey(false); setCreatedSecret(null) }} title="新建 API Key">
        {createdSecret ? (
          <div className="space-y-4">
            <Alert variant="success" title="API Key 已创建">
              请立即复制保存，此密钥只会显示一次。
            </Alert>
            <div className="flex items-center gap-2">
              <code className="flex-1 break-all rounded bg-[var(--surface-inset)] px-3 py-2 text-sm">{createdSecret}</code>
              <Button variant="ghost" size="sm" onClick={() => copyToClipboard(createdSecret)}>
                <Copy className="h-4 w-4" />
              </Button>
            </div>
            {copiedSecret && <div className="text-sm text-[var(--text-success)]">已复制到剪贴板</div>}
            <div className="flex justify-end">
              <Button onClick={() => { setShowCreateKey(false); setCreatedSecret(null) }}>完成</Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <Input
              label="Key 名称"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              placeholder="例如: Production, CI/CD"
              required
            />
            <Select
              label="过期时间"
              value={newKeyExpiration}
              onChange={(e) => setNewKeyExpiration(e.target.value)}
              options={EXPIRATION_OPTIONS}
            />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setShowCreateKey(false)}>取消</Button>
              <Button onClick={handleCreateKey} disabled={!newKeyName.trim()}>创建</Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
