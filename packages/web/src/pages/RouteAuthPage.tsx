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
import { Alert, Badge, Button, EmptyState, Input, Modal, Select, Switch } from '../components/ui'
import { routeAuthApi, apiKeyApi } from '../lib/api/route-auth'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import { normalizeCORSMethods } from '../lib/cors'
import type { ApiKey, ApiKeyCreateInput, Route, RouteAuthConfig, RouteAuthConfigInput } from '../lib/api/types'

type RouteAuthConfigDraft = {
  api_key_enabled: boolean
  api_key_header: string
  gateway_enabled: boolean
  gateway_login_mode: string
  whitelist: string
  rate_limit: string
  burst: string
  cors_allowed_origins: string
  cors_allowed_methods: string
  cors_allowed_headers: string
  cors_allow_credentials: boolean
  cors_max_age: string
}

type DetailFieldErrors = Partial<Record<'rateLimit' | 'burst' | 'corsMaxAge' | 'corsAllowedMethods', string>>
type DetailKeyMutation = {
  kind: 'create' | 'rotate' | 'expire' | 'delete'
  keyId?: string
}

function createConfigDraft(config: RouteAuthConfig | null): RouteAuthConfigDraft {
  return {
    api_key_enabled: config?.api_key_enabled ?? false,
    api_key_header: config?.api_key_header ?? '',
    gateway_enabled: config?.gateway_enabled ?? false,
    gateway_login_mode: config?.gateway_login_mode ?? '',
    whitelist: (config?.whitelist ?? []).join(', '),
    rate_limit: config?.rate_limit == null ? '' : String(config.rate_limit),
    burst: config?.burst == null ? '' : String(config.burst),
    cors_allowed_origins: config?.cors_allowed_origins ?? '',
    cors_allowed_methods: config?.cors_allowed_methods ?? '',
    cors_allowed_headers: config?.cors_allowed_headers ?? '',
    cors_allow_credentials: config?.cors_allow_credentials ?? false,
    cors_max_age: config?.cors_max_age == null ? '' : String(config.cors_max_age),
  }
}

function normalizeCommaSeparated(value: string): string {
  return value
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
    .join(',')
}

function parseNonNegativeInteger(value: string): number | null {
  if (value.trim() === '') return 0
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : null
}

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

function formatDate(locale: string, iso?: string | null): string {
  if (!iso) return '—'
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? '—' : new Intl.DateTimeFormat(locale).format(date)
}

function getActiveAuthTypes(
  cfg: Pick<RouteAuthConfig, 'api_key_enabled' | 'gateway_enabled'> | null
): string[] {
  if (!cfg) return []
  const types: string[] = []
  if (cfg.api_key_enabled) types.push('apikey')
  if (cfg.gateway_enabled) types.push('gateway')
  return types
}

export function RouteAuthPage() {
  const { t, i18n } = useTranslation(['authRules', 'common'])
  const locale = i18n.resolvedLanguage || 'en'
  const expirationOptions = ['', '7d', '30d', '90d', '1y'].map(value => ({ value, label: t(`expiration.${value || 'never'}`) }))

  const [routes, setRoutes] = React.useState<Route[]>([])
  const [configs, setConfigs] = React.useState<Map<string, RouteAuthConfig>>(new Map())
  const [configLoadFailures, setConfigLoadFailures] = React.useState<Set<string>>(new Set())
  const [keyCounts, setKeyCounts] = React.useState<Map<string, number>>(new Map())
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)

  // Detail modal state
  const [detailRouteId, setDetailRouteId] = React.useState<string | null>(null)
  const [detailConfig, setDetailConfig] = React.useState<RouteAuthConfig | null>(null)
  const [detailDraft, setDetailDraft] = React.useState<RouteAuthConfigDraft | null>(null)
  const [detailKeys, setDetailKeys] = React.useState<ApiKey[]>([])
  const [detailLoading, setDetailLoading] = React.useState(false)
  const [detailSaving, setDetailSaving] = React.useState(false)
  const [detailError, setDetailError] = React.useState<string | null>(null)
  const [detailSaved, setDetailSaved] = React.useState(false)
  const [detailFieldErrors, setDetailFieldErrors] = React.useState<DetailFieldErrors>({})
  const [showDiscardConfirmation, setShowDiscardConfirmation] = React.useState(false)
  const detailRouteIdRef = React.useRef<string | null>(null)
  const detailLoadGenerationRef = React.useRef(0)
  const detailSaveGenerationRef = React.useRef(0)
  const activeDetailSaveRef = React.useRef<symbol | null>(null)
  const detailKeyRefreshGenerationRef = React.useRef(0)
  const activeKeyMutationRef = React.useRef<symbol | null>(null)

  // Key creation state
  const [showCreateKey, setShowCreateKey] = React.useState(false)
  const [newKeyExpiration, setNewKeyExpiration] = React.useState('')
  const [createdSecret, setCreatedSecret] = React.useState<string | null>(null)
  const [copiedId, setCopiedId] = React.useState<string | null>(null)
  const [revealedIds, setRevealedIds] = React.useState<Set<string>>(new Set())
  const [detailKeyMutation, setDetailKeyMutation] = React.useState<DetailKeyMutation | null>(null)

  const canManageAuth = getSessionUser()?.permissions?.can_manage_auth ?? false

  // Load all routes + their configs on mount
  const loadAll = React.useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const routeList = await routesApi.list()
      setRoutes(routeList)

      // Load configs for all routes in parallel
      const [configResults, keyResults] = await Promise.all([
        Promise.allSettled(routeList.map((r) => routeAuthApi.getConfig(r.id))),
        Promise.allSettled(routeList.map((r) => apiKeyApi.list(r.id))),
      ])

      const configMap = new Map<string, RouteAuthConfig>()
      const configFailures = new Set<string>()
      const countMap = new Map<string, number>()

      routeList.forEach((route, idx) => {
        const configResult = configResults[idx]
        const keyResult = keyResults[idx]
        const routeId = routeList[idx].id
        if (configResult.status === 'fulfilled') {
          configMap.set(routeId, configResult.value)
        } else {
          configFailures.add(route.id)
        }
        if (keyResult.status === 'fulfilled') {
          countMap.set(routeId, keyResult.value.length)
        }
      })

      setConfigs(configMap)
      setConfigLoadFailures(configFailures)
      setKeyCounts(countMap)
    } catch {
      setError('errors.routeStoreFailure')
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => { loadAll() }, [loadAll])

  const openDetail = (routeId: string) => {
    detailLoadGenerationRef.current += 1
    detailSaveGenerationRef.current += 1
    activeDetailSaveRef.current = null
    detailKeyRefreshGenerationRef.current += 1
    activeKeyMutationRef.current = null
    detailRouteIdRef.current = routeId
    setDetailRouteId(routeId)
    setDetailConfig(null)
    setDetailDraft(null)
    setDetailKeys([])
    setDetailLoading(true)
    setDetailSaving(false)
    setDetailError(null)
    setDetailSaved(false)
    setDetailFieldErrors({})
    setShowDiscardConfirmation(false)
    setCreatedSecret(null)
    setShowCreateKey(false)
    setNewKeyExpiration('')
    setRevealedIds(new Set())
    setDetailKeyMutation(null)
  }

  // Load detail data for modal
  const loadDetail = React.useCallback(async (routeId: string) => {
    const requestGeneration = ++detailLoadGenerationRef.current
    setDetailLoading(true)
    setDetailError(null)
    try {
      const [config, keys] = await Promise.all([
        routeAuthApi.getConfig(routeId),
        apiKeyApi.list(routeId).catch(() => []),
      ])
      if (requestGeneration !== detailLoadGenerationRef.current || detailRouteIdRef.current !== routeId) {
        return
      }
      setDetailConfig(config)
      setDetailDraft(createConfigDraft(config))
      setDetailKeys(keys)
      setConfigs((prev) => {
        const next = new Map(prev)
        next.set(routeId, config)
        return next
      })
      setConfigLoadFailures((prev) => {
        if (!prev.has(routeId)) return prev
        const next = new Set(prev)
        next.delete(routeId)
        return next
      })
    } catch {
      if (requestGeneration !== detailLoadGenerationRef.current || detailRouteIdRef.current !== routeId) {
        return
      }
      setDetailConfig(null)
      setDetailDraft(null)
      setDetailKeys([])
      setDetailError('errors.routeAuthConfigLoadFailure')
    } finally {
      if (requestGeneration === detailLoadGenerationRef.current && detailRouteIdRef.current === routeId) {
        setDetailLoading(false)
      }
    }
  }, [])

  React.useEffect(() => {
    if (detailRouteId) loadDetail(detailRouteId)
    return () => {
      detailLoadGenerationRef.current += 1
    }
  }, [detailRouteId, loadDetail])

  // Quick toggle: update config from the list view
  const quickToggle = async (routeId: string, patch: RouteAuthConfigInput) => {
    if (configLoadFailures.has(routeId)) return
    try {
      const updated = await routeAuthApi.updateConfig(routeId, patch)
      setConfigs((prev) => {
        const next = new Map(prev)
        next.set(routeId, updated)
        return next
      })
      setConfigLoadFailures((prev) => {
        if (!prev.has(routeId)) return prev
        const next = new Set(prev)
        next.delete(routeId)
        return next
      })
    } catch {
      setError('errors.authRuleStoreFailure')
    }
  }

  const updateDetailDraft = (patch: Partial<RouteAuthConfigDraft>) => {
    if (activeDetailSaveRef.current) return
    setDetailDraft((current) => current ? { ...current, ...patch } : current)
    setDetailSaved(false)
    setDetailFieldErrors({})
  }

  // Detail: validate and save the complete local draft once.
  const saveDetailConfig = async () => {
    if (!detailRouteId || !detailDraft || activeDetailSaveRef.current || activeKeyMutationRef.current) return

    const rateLimit = parseNonNegativeInteger(detailDraft.rate_limit)
    const burst = parseNonNegativeInteger(detailDraft.burst)
    const corsMaxAge = parseNonNegativeInteger(detailDraft.cors_max_age)
    const corsAllowedMethods = normalizeCORSMethods(detailDraft.cors_allowed_methods)
    const nextFieldErrors: DetailFieldErrors = {}
    const numberError = 'form.nonNegativeIntegerError'
    if (rateLimit == null) nextFieldErrors.rateLimit = numberError
    if (burst == null) nextFieldErrors.burst = numberError
    if (corsMaxAge == null) nextFieldErrors.corsMaxAge = numberError
    if (corsAllowedMethods == null) {
      nextFieldErrors.corsAllowedMethods = 'form.allowedMethodsError'
    }
    if (Object.keys(nextFieldErrors).length > 0) {
      setDetailFieldErrors(nextFieldErrors)
      setDetailSaved(false)
      return
    }

    const routeId = detailRouteId
    const saveGeneration = ++detailSaveGenerationRef.current
    const saveToken = Symbol('detail-save')
    activeDetailSaveRef.current = saveToken

    const input: RouteAuthConfigInput = {
      api_key_enabled: detailDraft.api_key_enabled,
      api_key_header: detailDraft.api_key_header.trim(),
      gateway_enabled: detailDraft.gateway_enabled,
      gateway_login_mode: detailDraft.gateway_login_mode.trim(),
      whitelist: detailDraft.whitelist
        .split(/[\n,]/)
        .map((item) => item.trim())
        .filter(Boolean),
      rate_limit: rateLimit,
      burst,
      cors_allowed_origins: normalizeCommaSeparated(detailDraft.cors_allowed_origins),
      cors_allowed_methods: corsAllowedMethods,
      cors_allowed_headers: normalizeCommaSeparated(detailDraft.cors_allowed_headers),
      cors_allow_credentials: detailDraft.cors_allow_credentials,
      cors_max_age: corsMaxAge,
    }

    setDetailSaving(true)
    setShowDiscardConfirmation(false)
    setDetailError(null)
    setDetailFieldErrors({})
    setDetailSaved(false)
    try {
      const updated = await routeAuthApi.updateConfig(routeId, input)
      setConfigs((prev) => {
        const next = new Map(prev)
        next.set(routeId, updated)
        return next
      })
      setConfigLoadFailures((prev) => {
        if (!prev.has(routeId)) return prev
        const next = new Set(prev)
        next.delete(routeId)
        return next
      })
      if (activeDetailSaveRef.current === saveToken && saveGeneration === detailSaveGenerationRef.current && detailRouteIdRef.current === routeId) {
        setDetailConfig(updated)
        setDetailDraft(createConfigDraft(updated))
        setDetailSaved(true)
      }
    } catch {
      if (activeDetailSaveRef.current === saveToken && saveGeneration === detailSaveGenerationRef.current && detailRouteIdRef.current === routeId) {
        setDetailError('errors.authRuleStoreFailure')
      }
    } finally {
      if (activeDetailSaveRef.current === saveToken && saveGeneration === detailSaveGenerationRef.current && detailRouteIdRef.current === routeId) {
        activeDetailSaveRef.current = null
        setDetailSaving(false)
      }
    }
  }

  const refreshDetailKeys = React.useCallback(async (routeId: string) => {
    const requestGeneration = ++detailKeyRefreshGenerationRef.current
    const keys = await apiKeyApi.list(routeId)
    if (requestGeneration !== detailKeyRefreshGenerationRef.current) {
      return
    }
    setKeyCounts((prev) => {
      const next = new Map(prev)
      next.set(routeId, keys.length)
      return next
    })
    if (detailRouteIdRef.current === routeId) {
      setDetailKeys(keys)
    }
  }, [])

  const beginKeyMutation = (routeId: string, mutation: DetailKeyMutation): symbol | null => {
    if (activeDetailSaveRef.current || activeKeyMutationRef.current || detailRouteIdRef.current !== routeId) return null
    const token = Symbol(mutation.kind)
    activeKeyMutationRef.current = token
    setDetailKeyMutation(mutation)
    setDetailError(null)
    return token
  }

  const isCurrentKeyMutation = (token: symbol, routeId: string) =>
    activeKeyMutationRef.current === token && detailRouteIdRef.current === routeId

  const finishKeyMutation = (token: symbol, routeId: string) => {
    if (activeKeyMutationRef.current !== token) return
    activeKeyMutationRef.current = null
    if (detailRouteIdRef.current === routeId) {
      setDetailKeyMutation(null)
    }
  }

  const refreshKeysAfterMutation = async (routeId: string, token: symbol) => {
    try {
      await refreshDetailKeys(routeId)
    } catch {
      if (isCurrentKeyMutation(token, routeId)) {
        setDetailError('errors.apiKeyListRefreshFailure')
      }
    }
  }

  // API Key operations
  const handleCreateKey = async () => {
    if (!detailRouteId) return
    const routeId = detailRouteId
    const mutationToken = beginKeyMutation(routeId, { kind: 'create' })
    if (!mutationToken) return
    try {
      const input: ApiKeyCreateInput = { expires_at: parseExpiration(newKeyExpiration) }
      const resp = await apiKeyApi.create(routeId, input)
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setCreatedSecret(resp.secret)
        setNewKeyExpiration('')
        setShowCreateKey(false)
      }
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        await refreshKeysAfterMutation(routeId, mutationToken)
      }
    } catch {
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setDetailError('errors.missingAPIKeySecret')
      }
    } finally {
      finishKeyMutation(mutationToken, routeId)
    }
  }

  const handleRotateKey = async (id: string) => {
    if (!detailRouteId) return
    const routeId = detailRouteId
    const mutationToken = beginKeyMutation(routeId, { kind: 'rotate', keyId: id })
    if (!mutationToken) return
    try {
      const resp = await apiKeyApi.rotate(id)
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setCreatedSecret(resp.secret)
      }
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        await refreshKeysAfterMutation(routeId, mutationToken)
      }
    } catch {
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setDetailError('errors.authRuleStoreFailure')
      }
    } finally {
      finishKeyMutation(mutationToken, routeId)
    }
  }

  const handleExpireKey = async (id: string) => {
    if (!detailRouteId) return
    const routeId = detailRouteId
    const mutationToken = beginKeyMutation(routeId, { kind: 'expire', keyId: id })
    if (!mutationToken) return
    try {
      await apiKeyApi.expire(id)
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        await refreshKeysAfterMutation(routeId, mutationToken)
      }
    } catch {
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setDetailError('errors.authRuleStoreFailure')
      }
    } finally {
      finishKeyMutation(mutationToken, routeId)
    }
  }

  const handleDeleteKey = async (id: string) => {
    if (!detailRouteId) return
    const routeId = detailRouteId
    const mutationToken = beginKeyMutation(routeId, { kind: 'delete', keyId: id })
    if (!mutationToken) return
    try {
      await apiKeyApi.delete(id)
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        await refreshKeysAfterMutation(routeId, mutationToken)
      }
    } catch {
      if (isCurrentKeyMutation(mutationToken, routeId)) {
        setDetailError('errors.authRuleStoreFailure')
      }
    } finally {
      finishKeyMutation(mutationToken, routeId)
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
  const detailCfg = detailDraft ?? createConfigDraft(detailConfig)
  const keyControlsDisabled = detailSaving || detailKeyMutation != null
  const detailDirty = detailDraft != null && detailConfig != null &&
    JSON.stringify(detailDraft) !== JSON.stringify(createConfigDraft(detailConfig))

  const closeDetail = () => {
    if (activeDetailSaveRef.current || activeKeyMutationRef.current) return
    detailLoadGenerationRef.current += 1
    detailSaveGenerationRef.current += 1
    activeDetailSaveRef.current = null
    detailKeyRefreshGenerationRef.current += 1
    activeKeyMutationRef.current = null
    detailRouteIdRef.current = null
    setDetailRouteId(null)
    setDetailConfig(null)
    setDetailDraft(null)
    setDetailKeys([])
    setDetailLoading(false)
    setDetailSaving(false)
    setDetailError(null)
    setDetailSaved(false)
    setDetailFieldErrors({})
    setShowDiscardConfirmation(false)
    setDetailKeyMutation(null)
  }

  const requestCloseDetail = () => {
    if (activeDetailSaveRef.current || activeKeyMutationRef.current) {
      return
    }
    if (detailDirty) {
      setShowDiscardConfirmation(true)
      return
    }
    closeDetail()
  }

  const typeBadge = (type: string) => {
    switch (type) {
      case 'apikey': return <Badge variant="primary" badgeSize="sm">API Key</Badge>
      case 'gateway': return <Badge variant="primary" badgeSize="sm">{t('screen.gateway')}</Badge>
      default: return null
    }
  }

  const keyStatusBadge = (status: string) => {
    switch (status) {
      case 'active': return <Badge variant="success" badgeSize="sm">{t('screen.active')}</Badge>
      case 'revoked': return <Badge variant="error" badgeSize="sm">{t('screen.revoked')}</Badge>
      case 'expired': return <Badge variant="warning" badgeSize="sm">{t('screen.expired')}</Badge>
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
            <Badge variant="primary">{t('screen.routeCount', { count: routes.length })}</Badge>
          ) : undefined
        }
      />

      {error && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">{t(error)}</Alert>
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
            <div className="min-w-0 flex-1">{t('screen.route')}</div>
            <div className="w-24 text-center">{"API Key"}</div>
            <div className="w-24 text-center">{t('screen.gateway')}</div>
            <div className="w-20 text-center">{t('screen.keys')}</div>
            <div className="w-16" />
          </div>

          {/* Route List */}
          {routes.map((route) => {
            const cfg = configs.get(route.id) || null
            const configUnavailable = configLoadFailures.has(route.id)
            const activeTypes = getActiveAuthTypes(cfg)
            const keyCount = keyCounts.get(route.id) || 0
            const hasAuth = !configUnavailable && activeTypes.length > 0

            return (
              <div
                key={route.id}
                className="flex items-center gap-4 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-3 transition-colors hover:bg-[var(--bg-hover)]"
              >
                {/* Route info */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-[var(--text-primary)]">{route.name || route.path_prefix}</span>
                    {configUnavailable ? (
                      <Badge variant="warning" badgeSize="sm">{t('page.configUnavailable')}</Badge>
                    ) : hasAuth ? (
                      <Badge variant="success" badgeSize="sm">{t('screen.protected')}</Badge>
                    ) : (
                      <Badge variant="default" badgeSize="sm">{t('screen.public')}</Badge>
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
                            aria-label={t('screen.apiKeyForRoute', { name: route.name || route.path_prefix })}
                        checked={cfg?.api_key_enabled || false}
                            disabled={configUnavailable}
                            onChange={(e) => quickToggle(route.id, { api_key_enabled: e.target.checked })}
                            className="h-3.5 w-3.5 rounded border-[var(--border-default)] bg-[var(--bg-input)] text-[var(--primary-500)]"
                          />
                          API Key
                        </label>
                        <label className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                          <input
                            type="checkbox"
                            aria-label={t('screen.gatewayForRoute', { name: route.name || route.path_prefix })}
                        checked={cfg?.gateway_enabled || false}
                            disabled={configUnavailable}
                            onChange={(e) => quickToggle(route.id, { gateway_enabled: e.target.checked })}
                            className="h-3.5 w-3.5 rounded border-[var(--border-default)] bg-[var(--bg-input)] text-[var(--primary-500)]"
                          />
                          {t('screen.gateway')}
                        </label>
                      </>
                    )}
                    <span className="ml-auto text-xs text-[var(--text-muted)]">{t('screen.keyCount', { count: keyCount })}</span>
                  </div>
                </div>

                {/* Desktop: toggles */}
                <div className="hidden items-center gap-4 md:flex">
                  <div className="w-24 flex justify-center">
                    {canManageAuth ? (
                      <Switch
                        switchSize="sm"
                        aria-label={t('screen.apiKeyForRoute', { name: route.name || route.path_prefix })}
                        checked={cfg?.api_key_enabled || false}
                        disabled={configUnavailable}
                        onChange={(e) => quickToggle(route.id, { api_key_enabled: e.target.checked })}
                      />
                    ) : (
                      <Badge variant={configUnavailable ? 'warning' : cfg?.api_key_enabled ? 'success' : 'default'} badgeSize="sm">
                        {configUnavailable ? t('page.configUnknown') : cfg?.api_key_enabled ? t('screen.on') : t('screen.off')}
                      </Badge>
                    )}
                  </div>
                  <div className="w-24 flex justify-center">
                    {canManageAuth ? (
                      <Switch
                        switchSize="sm"
                        aria-label={t('screen.gatewayForRoute', { name: route.name || route.path_prefix })}
                        checked={cfg?.gateway_enabled || false}
                        disabled={configUnavailable}
                        onChange={(e) => quickToggle(route.id, { gateway_enabled: e.target.checked })}
                      />
                    ) : (
                      <Badge variant={configUnavailable ? 'warning' : cfg?.gateway_enabled ? 'success' : 'default'} badgeSize="sm">
                        {configUnavailable ? t('page.configUnknown') : cfg?.gateway_enabled ? t('screen.on') : t('screen.off')}
                      </Badge>
                    )}
                  </div>
                  <div className="w-20 flex justify-center">
                    <span className="text-sm font-medium text-[var(--text-secondary)]">{new Intl.NumberFormat(locale).format(keyCount)}</span>
                  </div>
                  <div className="w-16 flex justify-center">
                    <button
                      type="button"
                      onClick={() => openDetail(route.id)}
                      className="flex h-8 w-8 items-center justify-center rounded-[8px] border border-[var(--border-default)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                      aria-label={t('screen.edit')}
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                {/* Mobile: edit button */}
                <button
                  type="button"
                  onClick={() => openDetail(route.id)}
                  className="flex h-8 w-8 items-center justify-center rounded-[8px] border border-[var(--border-default)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] md:hidden"
                  aria-label={t('screen.edit')}
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
        onClose={requestCloseDetail}
        title={selectedRoute ? t('screen.configurationTitle', { name: selectedRoute.name || selectedRoute.path_prefix }) : ''}
        modalSize="lg"
        closeDisabled={detailSaving || detailKeyMutation != null}
        inactive={showDiscardConfirmation}
      >
        {detailLoading ? (
          <div className="flex h-48 items-center justify-center">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          </div>
        ) : !detailDraft ? (
          <Alert variant="error" title={t('page.errorTitle')}>
            {t(detailError || 'errors.routeAuthConfigLoadFailure')}
          </Alert>
        ) : (
          <div className="space-y-6">
            {detailError && (
              <Alert variant="error" title={t('page.errorTitle')}>{t(detailError)}</Alert>
            )}

            {detailSaved && (
              <Alert variant="success">{t('detail.configurationSaved')}</Alert>
            )}

            {createdSecret && (
              <Alert variant="success" title={t('screen.apiKeyCreated')}>
                <div className="mt-2 flex items-center gap-2">
                  <code className="flex-1 break-all rounded bg-[var(--bg-input)] px-3 py-2 text-sm text-[var(--text-primary)]">{createdSecret}</code>
                  <Button variant="ghost" size="sm" aria-label={t('screen.copyKey')} onClick={() => copyToClipboard(createdSecret, 'created')}>
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                {copiedId === 'created' && (
                  <div className="mt-1 text-sm text-[var(--success)]">{t('screen.copiedToClipboard')}</div>
                )}
                <div className="mt-2">
                  <Button size="sm" variant="ghost" onClick={() => setCreatedSecret(null)}>
                    {t('screen.dismiss')}
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
                    <Badge variant="default" badgeSize="sm">{t('screen.noAuth')}</Badge>
                  )}
                </div>
              </div>
            )}

            {/* Auth Toggles */}
            <div className="space-y-3">
              <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {t('screen.authenticationMethods')}
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
                        {t('screen.apiKeyDescription')}
                      </div>
                    </div>
                    {canManageAuth && (
                      <Switch
                        switchSize="sm"
                        aria-label={t('types.apikey')}
                        checked={detailCfg.api_key_enabled}
                        disabled={detailSaving}
                        onChange={(e) => updateDetailDraft({ api_key_enabled: e.target.checked })}
                      />
                    )}
                  </div>
                  {detailCfg.api_key_enabled && (
                    <div className="border-t border-[var(--border-soft)] px-4 py-3">
                      <Input
                        label={t('screen.headerName')}
                        value={detailCfg.api_key_header || ''}
                        disabled={detailSaving}
                        onChange={(e) => updateDetailDraft({ api_key_header: e.target.value })}
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
                      <div className="text-sm font-medium text-[var(--text-primary)]">{t('screen.gatewayLogin')}</div>
                      <div className="text-xs text-[var(--text-muted)]">
                        {t('screen.gatewayDescription')}
                      </div>
                    </div>
                    {canManageAuth && (
                      <Switch
                        switchSize="sm"
                        aria-label={t('types.gateway')}
                        checked={detailCfg.gateway_enabled}
                        disabled={detailSaving}
                        onChange={(e) => updateDetailDraft({ gateway_enabled: e.target.checked })}
                      />
                    )}
                  </div>
                  {detailCfg.gateway_enabled && (
                    <div className="border-t border-[var(--border-soft)] px-4 py-3">
                      <div className="rounded-[10px] bg-[var(--bg-surface-tint)] px-3 py-2 text-xs text-[var(--text-muted)]">
                        {t('screen.gatewayHint')}
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
                    {t('screen.apiKeyManagement')}
                  </div>
                  {canManageAuth && (
                    <Button size="sm" variant="ghost" icon={<Plus className="h-3.5 w-3.5" />} disabled={keyControlsDisabled} onClick={() => setShowCreateKey(true)}>
                      {t('screen.newKey')}
                    </Button>
                  )}
                </div>

                {showCreateKey && canManageAuth && (
                  <div className="flex items-end gap-3 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-surface-tint)] p-4">
                    <Select
                      label={t('screen.expiration')}
                      value={newKeyExpiration}
                      disabled={keyControlsDisabled}
                      onChange={(e) => setNewKeyExpiration(e.target.value)}
                      options={expirationOptions}
                      className="flex-1"
                    />
                    <Button disabled={detailSaving} loading={detailKeyMutation?.kind === 'create'} onClick={handleCreateKey}>{t('screen.create')}</Button>
                    <Button variant="ghost" disabled={keyControlsDisabled} onClick={() => setShowCreateKey(false)}>{t('screen.cancel')}</Button>
                  </div>
                )}

                {detailKeys.length === 0 ? (
                  <div className="rounded-[12px] border border-dashed border-[var(--border-default)] p-6 text-center text-sm text-[var(--text-muted)]">
                    {t('screen.noApiKeysYet')}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {detailKeys.map((key) => (
                      <div key={key.id} className="flex items-center gap-3 rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-3">
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium text-[var(--text-primary)]">{key.name || (t('screen.unnamed'))}</span>
                            {keyStatusBadge(key.status)}
                          </div>
                          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-[var(--text-muted)]">
                            <code className="rounded bg-[var(--bg-input)] px-1.5 py-0.5 font-mono text-[var(--text-secondary)]">
                              {key.secret
                                ? (revealedIds.has(key.id) ? key.secret : key.key_prefix + '••••••••')
                                : key.key_prefix + '...'}
                            </code>
                            <span>{t('screen.expires')}: {formatDate(locale, key.expires_at)}</span>
                            {key.last_used_at && <span>{t('screen.lastUsed')}: {formatDate(locale, key.last_used_at)}</span>}
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          {key.secret && (
                            <>
                              <Button variant="ghost" size="sm" aria-label={revealedIds.has(key.id) ? t('screen.hideKey') : t('screen.showKey')} onClick={() => toggleReveal(key.id)}>
                                {revealedIds.has(key.id) ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                              </Button>
                              <Button variant="ghost" size="sm" aria-label={t('screen.copyKey')} onClick={() => copyToClipboard(key.secret!, key.id)}>
                                {copiedId === key.id ? <span className="text-xs text-[var(--success)]">{t('screen.copied')}</span> : <Copy className="h-3.5 w-3.5" />}
                              </Button>
                            </>
                          )}
                          {canManageAuth && key.status === 'active' && (
                            <>
                              <Button variant="ghost" size="sm" disabled={keyControlsDisabled} loading={detailKeyMutation?.kind === 'rotate' && detailKeyMutation.keyId === key.id} onClick={() => handleRotateKey(key.id)} title={t('screen.rotateKey')}>
                                <RefreshCw className="h-3.5 w-3.5" />
                              </Button>
                              <Button variant="ghost" size="sm" disabled={keyControlsDisabled} loading={detailKeyMutation?.kind === 'expire' && detailKeyMutation.keyId === key.id} onClick={() => handleExpireKey(key.id)} title={t('screen.revoke')}>
                                <XCircle className="h-3.5 w-3.5 text-[var(--error)]" />
                              </Button>
                              <Button variant="ghost" size="sm" disabled={keyControlsDisabled} loading={detailKeyMutation?.kind === 'delete' && detailKeyMutation.keyId === key.id} onClick={() => handleDeleteKey(key.id)} title={t('screen.delete')}>
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
                {t('screen.rateLimiting')}
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <Input
                  label={t('screen.rateLimit')}
                  type="number"
                  min={0}
                  step={1}
                  value={detailCfg.rate_limit}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ rate_limit: e.target.value })}
                  error={detailFieldErrors.rateLimit ? t(detailFieldErrors.rateLimit!) : undefined}
                  placeholder={t('screen.unlimited')}
                />
                <Input
                  label={t('screen.burst')}
                  type="number"
                  min={0}
                  step={1}
                  value={detailCfg.burst}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ burst: e.target.value })}
                  error={detailFieldErrors.burst ? t(detailFieldErrors.burst!) : undefined}
                  placeholder={t('screen.unlimited')}
                />
                <Input
                  label={t('screen.whitelist')}
                  value={detailCfg.whitelist}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ whitelist: e.target.value })}
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
                  label={t('screen.allowedOrigins')}
                  value={detailCfg.cors_allowed_origins || ''}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ cors_allowed_origins: e.target.value })}
                  placeholder="https://app.example.com, *"
                />
                <Input
                  label={t('screen.allowedMethods')}
                  value={detailCfg.cors_allowed_methods || ''}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ cors_allowed_methods: e.target.value })}
                  error={detailFieldErrors.corsAllowedMethods ? t(detailFieldErrors.corsAllowedMethods!) : undefined}
                  placeholder="GET,POST,OPTIONS"
                />
                <Input
                  label={t('screen.allowedHeaders')}
                  value={detailCfg.cors_allowed_headers || ''}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ cors_allowed_headers: e.target.value })}
                  placeholder="Authorization,Content-Type"
                />
                <Input
                  label={t('screen.maxAge')}
                  type="number"
                  min={0}
                  step={1}
                  value={detailCfg.cors_max_age}
                  disabled={detailSaving}
                  onChange={(e) => updateDetailDraft({ cors_max_age: e.target.value })}
                  error={detailFieldErrors.corsMaxAge ? t(detailFieldErrors.corsMaxAge!) : undefined}
                  placeholder="86400"
                />
              </div>
              <Switch
                label={t('screen.allowCredentials')}
                description={t('screen.allowCredentialsHint')}
                checked={detailCfg.cors_allow_credentials || false}
                disabled={detailSaving}
                onChange={(e) => updateDetailDraft({ cors_allow_credentials: e.target.checked })}
              />
            </div>

            {canManageAuth && (
              <div className="flex justify-end border-t border-[var(--border-default)] pt-4">
                <Button disabled={detailKeyMutation != null} loading={detailSaving} onClick={saveDetailConfig}>
                  {t('detail.saveConfiguration')}
                </Button>
              </div>
            )}
          </div>
        )}
      </Modal>

      <Modal
        open={!!detailRouteId && showDiscardConfirmation}
        onClose={() => {
          if (!activeDetailSaveRef.current) setShowDiscardConfirmation(false)
        }}
        title={t('detail.discardTitle')}
        modalSize="sm"
        dialogRole="alertdialog"
        showCloseButton={false}
      >
        <p className="text-sm text-[var(--text-muted)]">{t('detail.discardDescription')}</p>
        <div className="mt-6 flex flex-col-reverse justify-end gap-2 sm:flex-row">
          <Button variant="ghost" onClick={() => setShowDiscardConfirmation(false)}>
            {t('detail.keepEditing')}
          </Button>
          <Button variant="danger" onClick={closeDetail}>
            {t('detail.discardChanges')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
