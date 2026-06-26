import React from 'react'
import {
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  XCircle,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, Input, Modal, Select, Switch } from '../components/ui'
import { routeAuthApi, apiKeyApi } from '../lib/api/route-auth'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { ApiKey, ApiKeyCreateInput, Route, RouteAuthConfig, RouteAuthConfigInput } from '../lib/api/types'

const EXPIRATION_OPTIONS = [
  { value: '', label: 'Never' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: '1y', label: '1 year' },
]

const EXPIRATION_OPTIONS_ZH = [
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

function getActiveAuthTypes(cfg: RouteAuthConfig | null): string[] {
  if (!cfg) return []
  const types: string[] = []
  if (cfg.api_key_enabled) types.push('apikey')
  if (cfg.gateway_enabled) types.push('gateway')
  return types
}

export function RouteAuthPage() {
  const { t, i18n } = useTranslation(['authRules', 'common'])
  const isZh = i18n.resolvedLanguage === 'zh-CN'
  const expirationOptions = isZh ? EXPIRATION_OPTIONS_ZH : EXPIRATION_OPTIONS

  const [routes, setRoutes] = React.useState<Route[]>([])
  const [configs, setConfigs] = React.useState<Map<string, RouteAuthConfig>>(new Map())
  const [keyCounts, setKeyCounts] = React.useState<Map<string, number>>(new Map())
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)

  // Detail modal state
  const [detailRouteId, setDetailRouteId] = React.useState<string | null>(null)
  const [detailConfig, setDetailConfig] = React.useState<RouteAuthConfig | null>(null)
  const [detailKeys, setDetailKeys] = React.useState<ApiKey[]>([])
  const [detailLoading, setDetailLoading] = React.useState(false)
  const [detailError, setDetailError] = React.useState<string | null>(null)

  // Key creation state
  const [showCreateKey, setShowCreateKey] = React.useState(false)
  const [newKeyExpiration, setNewKeyExpiration] = React.useState('')
  const [createdSecret, setCreatedSecret] = React.useState<string | null>(null)
  const [copiedId, setCopiedId] = React.useState<string | null>(null)
  const [revealedIds, setRevealedIds] = React.useState<Set<string>>(new Set())

  const canManageAuth = getSessionUser()?.permissions?.can_manage_auth ?? false

  // Load all routes + their configs on mount
  const loadAll = React.useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const routeList = await routesApi.list()
      setRoutes(routeList)

      // Load configs for all routes in parallel
      const results = await Promise.allSettled(
        routeList.map((r) =>
          Promise.all([
            routeAuthApi.getConfig(r.id).catch(() => null),
            apiKeyApi.list(r.id).catch(() => []),
          ])
        )
      )

      const configMap = new Map<string, RouteAuthConfig>()
      const countMap = new Map<string, number>()

      results.forEach((result, idx) => {
        const routeId = routeList[idx].id
        if (result.status === 'fulfilled') {
          const [config, keys] = result.value
          if (config) configMap.set(routeId, config)
          countMap.set(routeId, keys.length)
        }
      })

      setConfigs(configMap)
      setKeyCounts(countMap)
    } catch {
      setError(t('errors.routeStoreFailure'))
    } finally {
      setLoading(false)
    }
  }, [t])

  React.useEffect(() => { loadAll() }, [loadAll])

  // Load detail data for modal
  const loadDetail = React.useCallback(async (routeId: string) => {
    setDetailLoading(true)
    setDetailError(null)
    setCreatedSecret(null)
    setShowCreateKey(false)
    setNewKeyExpiration('')
    setRevealedIds(new Set())
    try {
      const [config, keys] = await Promise.all([
        routeAuthApi.getConfig(routeId).catch(() => null),
        apiKeyApi.list(routeId).catch(() => []),
      ])
      setDetailConfig(config)
      setDetailKeys(keys)
    } catch {
      setDetailError(t('errors.authRuleStoreFailure'))
    } finally {
      setDetailLoading(false)
    }
  }, [t])

  React.useEffect(() => {
    if (detailRouteId) loadDetail(detailRouteId)
  }, [detailRouteId, loadDetail])

  // Quick toggle: update config from the list view
  const quickToggle = async (routeId: string, patch: RouteAuthConfigInput) => {
    try {
      const updated = await routeAuthApi.updateConfig(routeId, patch)
      setConfigs((prev) => {
        const next = new Map(prev)
        next.set(routeId, updated)
        return next
      })
    } catch {
      setError(t('errors.authRuleStoreFailure'))
    }
  }

  // Detail: update config
  const detailUpdateConfig = async (patch: RouteAuthConfigInput) => {
    if (!detailRouteId) return
    try {
      const updated = await routeAuthApi.updateConfig(detailRouteId, patch)
      setDetailConfig(updated)
      // Also update the list view config
      setConfigs((prev) => {
        const next = new Map(prev)
        next.set(detailRouteId, updated)
        return next
      })
    } catch {
      setDetailError(t('errors.authRuleStoreFailure'))
    }
  }

  // API Key operations
  const handleCreateKey = async () => {
    if (!detailRouteId) return
    try {
      const input: ApiKeyCreateInput = { expires_at: parseExpiration(newKeyExpiration) }
      const resp = await apiKeyApi.create(detailRouteId, input)
      setCreatedSecret(resp.secret)
      setNewKeyExpiration('')
      setShowCreateKey(false)
      await loadDetail(detailRouteId)
      // Update key count in list
      const keys = await apiKeyApi.list(detailRouteId).catch(() => [])
      setKeyCounts((prev) => {
        const next = new Map(prev)
        next.set(detailRouteId, keys.length)
        return next
      })
    } catch {
      setDetailError(t('errors.missingAPIKeySecret'))
    }
  }

  const handleRotateKey = async (id: string) => {
    if (!detailRouteId) return
    try {
      const resp = await apiKeyApi.rotate(id)
      setCreatedSecret(resp.secret)
      await loadDetail(detailRouteId)
    } catch {
      setDetailError(t('errors.authRuleStoreFailure'))
    }
  }

  const handleExpireKey = async (id: string) => {
    if (!detailRouteId) return
    try {
      await apiKeyApi.expire(id)
      await loadDetail(detailRouteId)
    } catch {
      setDetailError(t('errors.authRuleStoreFailure'))
    }
  }

  const handleDeleteKey = async (id: string) => {
    if (!detailRouteId) return
    try {
      await apiKeyApi.delete(id)
      await loadDetail(detailRouteId)
      const keys = await apiKeyApi.list(detailRouteId).catch(() => [])
      setKeyCounts((prev) => {
        const next = new Map(prev)
        next.set(detailRouteId, keys.length)
        return next
      })
    } catch {
      setDetailError(t('errors.authRuleStoreFailure'))
    }
  }

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const toggleReveal = (id: string) => {
    setRevealedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const selectedRoute = routes.find((r) => r.id === detailRouteId)
  const detailCfg = detailConfig || {
    route_id: detailRouteId || '',
    api_key_enabled: false,
    gateway_enabled: false,
  }

  const typeBadge = (type: string) => {
    switch (type) {
      case 'apikey': return <Badge variant="primary" badgeSize="sm">API Key</Badge>
      case 'gateway': return <Badge variant="primary" badgeSize="sm">Gateway</Badge>
      default: return null
    }
  }

  const keyStatusBadge = (status: string) => {
    switch (status) {
      case 'active': return <Badge variant="success" badgeSize="sm">{isZh ? '有效' : 'Active'}</Badge>
      case 'revoked': return <Badge variant="error" badgeSize="sm">{isZh ? '已吊销' : 'Revoked'}</Badge>
      default: return <Badge variant="warning" badgeSize="sm">{status}</Badge>
    }
  }

  if (routes.length === 0 && !loading) {
    return (
      <div className="animate-rise-in">
        <PageHeader
          eyebrow={t('page.eyebrow')}
          title={t('page.title')}
          description={t('page.description')}
        />
        <EmptyState
          icon={<KeyRound className="h-8 w-8" />}
          title={t('page.emptyTitle')}
          description={t('page.noRoutesDescription')}
        />
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow={t('page.eyebrow')}
        title={t('page.title')}
        description={t('page.description')}
        meta={
          routes.length > 0 ? (
            <Badge variant="primary">{isZh ? `${routes.length} 个路由` : `${routes.length} routes`}</Badge>
          ) : undefined
        }
      />

      {error && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">{error}</Alert>
      )}

      {loading ? (
        <div className="flex h-48 items-center justify-center">
          <div className="flex items-center gap-3 text-[var(--text-muted)]">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
            {t('page.loading')}
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          {/* List Header */}
          <div className="hidden items-center px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)] md:flex">
            <div className="min-w-0 flex-1">{isZh ? '路由' : 'Route'}</div>
            <div className="w-24 text-center">{isZh ? 'API Key' : 'API Key'}</div>
            <div className="w-24 text-center">{isZh ? '网关' : 'Gateway'}</div>
            <div className="w-20 text-center">{isZh ? '密钥数' : 'Keys'}</div>
            <div className="w-16" />
          </div>

          {/* Route List */}
          {routes.map((route) => {
            const cfg = configs.get(route.id) || null
            const activeTypes = getActiveAuthTypes(cfg)
            const keyCount = keyCounts.get(route.id) || 0
            const hasAuth = activeTypes.length > 0

            return (
              <div
                key={route.id}
                className="flex items-center gap-4 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-3 transition-colors hover:bg-[var(--bg-hover)]"
              >
                {/* Route info */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-[var(--text-primary)]">{route.name || route.path_prefix}</span>
                    {hasAuth ? (
                      <Badge variant="success" badgeSize="sm">{isZh ? '已保护' : 'Protected'}</Badge>
                    ) : (
                      <Badge variant="default" badgeSize="sm">{isZh ? '公开' : 'Public'}</Badge>
                    )}
                  </div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-[var(--text-muted)]">
                    {route.host && <span className="app-code">{route.host}</span>}
                    <span className="app-code text-[var(--primary-600)]">{route.path_prefix}</span>
                  </div>
                  {/* Mobile: show toggles inline */}
                  <div className="mt-2 flex items-center gap-3 md:hidden">
                    {canManageAuth && (
                      <>
                        <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                          <input
                            type="checkbox"
                            checked={cfg?.api_key_enabled || false}
                            onChange={(e) => quickToggle(route.id, { api_key_enabled: e.target.checked })}
                            className="h-3.5 w-3.5 rounded border-[var(--border-default)] bg-[var(--bg-input)] text-[var(--primary-500)]"
                          />
                          API Key
                        </label>
                        <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                          <input
                            type="checkbox"
                            checked={cfg?.gateway_enabled || false}
                            onChange={(e) => quickToggle(route.id, { gateway_enabled: e.target.checked })}
                            className="h-3.5 w-3.5 rounded border-[var(--border-default)] bg-[var(--bg-input)] text-[var(--primary-500)]"
                          />
                          GW
                        </label>
                      </>
                    )}
                    <span className="ml-auto text-xs text-[var(--text-muted)]">{keyCount} {isZh ? '个密钥' : 'keys'}</span>
                  </div>
                </div>

                {/* Desktop: toggles */}
                <div className="hidden items-center gap-4 md:flex">
                  <div className="w-24 flex justify-center">
                    {canManageAuth ? (
                      <Switch
                        switchSize="sm"
                        checked={cfg?.api_key_enabled || false}
                        onChange={(e) => quickToggle(route.id, { api_key_enabled: e.target.checked })}
                      />
                    ) : (
                      <Badge variant={cfg?.api_key_enabled ? 'success' : 'default'} badgeSize="sm">
                        {cfg?.api_key_enabled ? 'ON' : 'OFF'}
                      </Badge>
                    )}
                  </div>
                  <div className="w-24 flex justify-center">
                    {canManageAuth ? (
                      <Switch
                        switchSize="sm"
                        checked={cfg?.gateway_enabled || false}
                        onChange={(e) => quickToggle(route.id, { gateway_enabled: e.target.checked })}
                      />
                    ) : (
                      <Badge variant={cfg?.gateway_enabled ? 'success' : 'default'} badgeSize="sm">
                        {cfg?.gateway_enabled ? 'ON' : 'OFF'}
                      </Badge>
                    )}
                  </div>
                  <div className="w-20 flex justify-center">
                    <span className="text-sm font-medium text-[var(--text-secondary)]">{keyCount}</span>
                  </div>
                  <div className="w-16 flex justify-center">
                    <button
                      type="button"
                      onClick={() => setDetailRouteId(route.id)}
                      className="flex h-8 w-8 items-center justify-center rounded-[8px] border border-[var(--border-default)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                      aria-label={isZh ? '编辑' : 'Edit'}
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                {/* Mobile: edit button */}
                <button
                  type="button"
                  onClick={() => setDetailRouteId(route.id)}
                  className="flex h-8 w-8 items-center justify-center rounded-[8px] border border-[var(--border-default)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] md:hidden"
                  aria-label={isZh ? '编辑' : 'Edit'}
                >
                  <Pencil className="h-4 w-4" />
                </button>
              </div>
            )
          })}
        </div>
      )}

      {/* Detail Modal */}
      <Modal
        open={!!detailRouteId}
        onClose={() => setDetailRouteId(null)}
        title={selectedRoute ? `${isZh ? '鉴权配置' : 'Auth Config'} — ${selectedRoute.name || selectedRoute.path_prefix}` : ''}
        modalSize="lg"
      >
        {detailLoading ? (
          <div className="flex h-48 items-center justify-center">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          </div>
        ) : (
          <div className="space-y-6">
            {detailError && (
              <Alert variant="error" title={t('page.errorTitle')}>{detailError}</Alert>
            )}

            {createdSecret && (
              <Alert variant="success" title={isZh ? 'API Key 已创建' : 'API Key Created'}>
                <div className="mt-2 flex items-center gap-2">
                  <code className="flex-1 break-all rounded bg-[var(--bg-input)] px-3 py-2 text-sm text-[var(--text-primary)]">{createdSecret}</code>
                  <Button variant="ghost" size="sm" onClick={() => copyToClipboard(createdSecret, 'created')}>
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                {copiedId === 'created' && (
                  <div className="mt-1 text-sm text-[var(--success)]">{isZh ? '已复制到剪贴板' : 'Copied to clipboard'}</div>
                )}
                <div className="mt-2">
                  <Button size="sm" variant="ghost" onClick={() => setCreatedSecret(null)}>
                    {isZh ? '关闭' : 'Dismiss'}
                  </Button>
                </div>
              </Alert>
            )}

            {/* Route info */}
            {selectedRoute && (
              <div className="flex items-center gap-4 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-surface-tint)] px-4 py-3">
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold text-[var(--text-primary)]">{selectedRoute.name || selectedRoute.path_prefix}</div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-[var(--text-muted)]">
                    {selectedRoute.host && <span className="app-code">{selectedRoute.host}</span>}
                    <span className="app-code text-[var(--primary-600)]">{selectedRoute.path_prefix}</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {getActiveAuthTypes(detailCfg).map((type) => (
                    <span key={type}>{typeBadge(type)}</span>
                  ))}
                  {getActiveAuthTypes(detailCfg).length === 0 && (
                    <Badge variant="default" badgeSize="sm">{isZh ? '无鉴权' : 'No auth'}</Badge>
                  )}
                </div>
              </div>
            )}

            {/* Auth Toggles */}
            <div className="space-y-3">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {isZh ? '鉴权方式' : 'Authentication Methods'}
              </div>
              <div className="space-y-3">
                {/* API Key */}
                <div
                  className={`relative overflow-hidden rounded-[14px] border transition-all duration-[var(--duration-normal)] ${
                    detailCfg.api_key_enabled
                      ? 'border-l-2 border-l-[var(--primary-500)] border-y-transparent border-r-transparent bg-[var(--bg-card)]'
                      : 'border-[var(--border-default)] bg-[var(--bg-card)]'
                  }`}
                >
                  <div className="flex items-center gap-3 px-4 py-3.5">
                    <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] transition-colors ${
                      detailCfg.api_key_enabled
                        ? 'bg-[var(--primary-500)]/15 text-[var(--primary-600)]'
                        : 'bg-[var(--bg-surface-tint)] text-[var(--text-muted)]'
                    }`}>
                      <KeyRound className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-[var(--text-primary)]">API Key</div>
                      <div className="text-xs text-[var(--text-muted)]">
                        {isZh ? '通过请求头传递 API Key 进行鉴权' : 'Authenticate via request header with an API key'}
                      </div>
                    </div>
                    {canManageAuth && (
                      <Switch
                        switchSize="sm"
                        checked={detailCfg.api_key_enabled}
                        onChange={(e) => detailUpdateConfig({ api_key_enabled: e.target.checked })}
                      />
                    )}
                  </div>
                  {detailCfg.api_key_enabled && (
                    <div className="border-t border-[var(--border-soft)] px-4 py-3">
                      <Input
                        label={isZh ? 'Header 名称' : 'Header Name'}
                        value={detailCfg.api_key_header || ''}
                        onChange={(e) => detailUpdateConfig({ api_key_header: e.target.value })}
                        placeholder="X-API-Key"
                      />
                    </div>
                  )}
                </div>

                {/* Gateway Login */}
                <div
                  className={`relative overflow-hidden rounded-[14px] border transition-all duration-[var(--duration-normal)] ${
                    detailCfg.gateway_enabled
                      ? 'border-l-2 border-l-[var(--accent-500)] border-y-transparent border-r-transparent bg-[var(--bg-card)]'
                      : 'border-[var(--border-default)] bg-[var(--bg-card)]'
                  }`}
                >
                  <div className="flex items-center gap-3 px-4 py-3.5">
                    <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] transition-colors ${
                      detailCfg.gateway_enabled
                        ? 'bg-[var(--accent-500)]/15 text-[var(--accent-600)]'
                        : 'bg-[var(--bg-surface-tint)] text-[var(--text-muted)]'
                    }`}>
                      <KeyRound className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-[var(--text-primary)]">{isZh ? '网关登录' : 'Gateway Login'}</div>
                      <div className="text-xs text-[var(--text-muted)]">
                        {isZh ? '未登录用户自动跳转至登录页' : 'Unauthenticated users are redirected to the login page'}
                      </div>
                    </div>
                    {canManageAuth && (
                      <Switch
                        switchSize="sm"
                        checked={detailCfg.gateway_enabled}
                        onChange={(e) => detailUpdateConfig({ gateway_enabled: e.target.checked })}
                      />
                    )}
                  </div>
                  {detailCfg.gateway_enabled && (
                    <div className="border-t border-[var(--border-soft)] px-4 py-3">
                      <div className="rounded-[10px] bg-[var(--bg-surface-tint)] px-3 py-2 text-xs text-[var(--text-muted)]">
                        {isZh
                          ? '用户访问此路由时将自动跳转到统一登录页，登录后可访问所有已授权路由'
                          : 'Users visiting this route will be redirected to a unified login page. After login, they can access all authorized routes'}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* API Keys Section */}
            {detailCfg.api_key_enabled && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                    {isZh ? 'API Key 管理' : 'API Key Management'}
                  </div>
                  {canManageAuth && (
                    <Button size="sm" variant="ghost" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowCreateKey(true)}>
                      {isZh ? '新建 Key' : 'New Key'}
                    </Button>
                  )}
                </div>

                {showCreateKey && canManageAuth && (
                  <div className="flex items-end gap-3 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-surface-tint)] p-4">
                    <Select
                      label={isZh ? '过期时间' : 'Expiration'}
                      value={newKeyExpiration}
                      onChange={(e) => setNewKeyExpiration(e.target.value)}
                      options={expirationOptions}
                      className="flex-1"
                    />
                    <Button onClick={handleCreateKey}>{isZh ? '创建' : 'Create'}</Button>
                    <Button variant="ghost" onClick={() => setShowCreateKey(false)}>{isZh ? '取消' : 'Cancel'}</Button>
                  </div>
                )}

                {detailKeys.length === 0 ? (
                  <div className="rounded-[12px] border border-dashed border-[var(--border-default)] p-6 text-center text-sm text-[var(--text-muted)]">
                    {isZh ? '暂无 API Key' : 'No API keys yet'}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {detailKeys.map((key) => (
                      <div key={key.id} className="flex items-center gap-3 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-3">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-[var(--text-primary)]">{key.name || (isZh ? '未命名' : 'Unnamed')}</span>
                            {keyStatusBadge(key.status)}
                          </div>
                          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[var(--text-muted)]">
                            <code className="rounded bg-[var(--bg-input)] px-1.5 py-0.5 font-mono text-[var(--text-secondary)]">
                              {key.secret
                                ? (revealedIds.has(key.id) ? key.secret : key.key_prefix + '••••••••')
                                : key.key_prefix + '...'}
                            </code>
                            <span>{isZh ? '过期' : 'Exp'}: {formatDate(key.expires_at)}</span>
                            {key.last_used_at && <span>{isZh ? '最后使用' : 'Used'}: {formatDate(key.last_used_at)}</span>}
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          {key.secret && (
                            <>
                              <Button variant="ghost" size="sm" onClick={() => toggleReveal(key.id)}>
                                {revealedIds.has(key.id) ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                              </Button>
                              <Button variant="ghost" size="sm" onClick={() => copyToClipboard(key.secret!, key.id)}>
                                {copiedId === key.id ? <span className="text-xs text-[var(--success)]">{isZh ? '已复制' : 'Copied'}</span> : <Copy className="h-3.5 w-3.5" />}
                              </Button>
                            </>
                          )}
                          {canManageAuth && key.status === 'active' && (
                            <>
                              <Button variant="ghost" size="sm" onClick={() => handleRotateKey(key.id)} title={isZh ? '轮换 Key' : 'Rotate key'}>
                                <RefreshCw className="h-3.5 w-3.5" />
                              </Button>
                              <Button variant="ghost" size="sm" onClick={() => handleExpireKey(key.id)} title={isZh ? '吊销' : 'Revoke'}>
                                <XCircle className="h-3.5 w-3.5 text-[var(--error)]" />
                              </Button>
                              <Button variant="ghost" size="sm" onClick={() => handleDeleteKey(key.id)} title={isZh ? '删除' : 'Delete'}>
                                <Trash2 className="h-3.5 w-3.5 text-[var(--error)]" />
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Rate Limiting */}
            <div className="space-y-3">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {isZh ? '速率限制' : 'Rate Limiting'}
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <Input
                  label={isZh ? '限流 (次/秒)' : 'Rate (req/s)'}
                  type="number"
                  min={0}
                  value={detailCfg.rate_limit ?? ''}
                  onChange={(e) => detailUpdateConfig({ rate_limit: e.target.value === '' ? undefined : Number(e.target.value) })}
                  placeholder={isZh ? '不限制' : 'Unlimited'}
                />
                <Input
                  label={isZh ? '突发上限' : 'Burst'}
                  type="number"
                  min={0}
                  value={detailCfg.burst ?? ''}
                  onChange={(e) => detailUpdateConfig({ burst: e.target.value === '' ? undefined : Number(e.target.value) })}
                  placeholder={isZh ? '不限制' : 'Unlimited'}
                />
                <Input
                  label={isZh ? '白名单' : 'Whitelist'}
                  value={(detailCfg.whitelist || []).join(', ')}
                  onChange={(e) => detailUpdateConfig({ whitelist: e.target.value.split(/[\n,]/).map((s) => s.trim()).filter(Boolean) })}
                  placeholder="127.0.0.1/32, 10.0.0.0/8"
                />
              </div>
            </div>

            {/* CORS */}
            <div className="space-y-3">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                CORS
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Input
                  label={isZh ? '允许来源' : 'Allowed Origins'}
                  value={detailCfg.cors_allowed_origins || ''}
                  onChange={(e) => detailUpdateConfig({ cors_allowed_origins: e.target.value })}
                  placeholder="https://app.example.com, *"
                />
                <Input
                  label={isZh ? '允许方法' : 'Allowed Methods'}
                  value={detailCfg.cors_allowed_methods || ''}
                  onChange={(e) => detailUpdateConfig({ cors_allowed_methods: e.target.value })}
                  placeholder="GET,POST,OPTIONS"
                />
                <Input
                  label={isZh ? '允许请求头' : 'Allowed Headers'}
                  value={detailCfg.cors_allowed_headers || ''}
                  onChange={(e) => detailUpdateConfig({ cors_allowed_headers: e.target.value })}
                  placeholder="Authorization,Content-Type"
                />
                <Input
                  label={isZh ? '缓存时长 (秒)' : 'Max Age (s)'}
                  type="number"
                  min={0}
                  value={detailCfg.cors_max_age ?? ''}
                  onChange={(e) => detailUpdateConfig({ cors_max_age: e.target.value === '' ? undefined : Number(e.target.value) })}
                  placeholder="86400"
                />
              </div>
              <Switch
                label={isZh ? '允许携带凭证' : 'Allow Credentials'}
                description={isZh ? '返回 Access-Control-Allow-Credentials' : 'Return Access-Control-Allow-Credentials'}
                checked={detailCfg.cors_allow_credentials || false}
                onChange={(e) => detailUpdateConfig({ cors_allow_credentials: e.target.checked })}
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
