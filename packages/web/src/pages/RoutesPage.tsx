import React from 'react'
import { Activity, Plus, Route as RouteIcon, Server, ToggleLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { RouteForm } from '../components/RouteForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { routesApi } from '../lib/api/routes'
import { certificatesApi } from '../lib/api/certificates'
import { getSessionUser } from '../lib/session-store'
import type { Certificate, Route, RouteInput } from '../lib/api/types'

function getPrimaryBackendTarget(route: Route) {
  const pooledTarget = route.backends?.find((backend) => backend.url?.trim())?.url
  if (pooledTarget) {
    return pooledTarget
  }
  return route.backend
}

function getRuntimePolicySummary(route: Route, t: ReturnType<typeof useTranslation>['t']) {
  const summary: string[] = []
  if ((route.timeout_ms || 0) > 0) {
    summary.push(t('table.timeoutSummary', { count: route.timeout_ms }))
  }
  if ((route.retry_attempts || 0) > 0) {
    summary.push(t('table.retrySummary', { count: route.retry_attempts }))
  }
  return summary
}

function getBackendTimeoutSummary(route: Route, t: ReturnType<typeof useTranslation>['t']) {
  const summary: string[] = []

  route.backends?.forEach((backend, index) => {
    const backendNumber = index + 1

    if ((backend.dial_timeout_ms || 0) > 0) {
      summary.push(
        t('table.backendDialTimeoutSummary', {
          count: backendNumber,
          timeout: backend.dial_timeout_ms,
        })
      )
    }
    if ((backend.read_timeout_ms || 0) > 0) {
      summary.push(
        t('table.backendReadTimeoutSummary', {
          count: backendNumber,
          timeout: backend.read_timeout_ms,
        })
      )
    }
    if ((backend.write_timeout_ms || 0) > 0) {
      summary.push(
        t('table.backendWriteTimeoutSummary', {
          count: backendNumber,
          timeout: backend.write_timeout_ms,
        })
      )
    }
    if ((backend.max_idle_conns || 0) > 0) {
      summary.push(
        t('table.backendMaxIdleConnsSummary', {
          count: backendNumber,
          idleConns: backend.max_idle_conns,
        })
      )
    }
  })

  return summary
}

export function RoutesPage() {
  const { t } = useTranslation('routes')
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [certificates, setCertificates] = React.useState<Certificate[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRoute, setEditingRoute] = React.useState<Route | null>(null)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [routeListUnavailable, setRouteListUnavailable] = React.useState(false)
  const [routeDirectoryUnavailable, setRouteDirectoryUnavailable] = React.useState(false)
  const requestGenerationRef = React.useRef(0)
  const canManageRoutes = getSessionUser()?.permissions?.can_manage_routes ?? false
  const showDirectoryMetrics = !routeListUnavailable
  const errorMessage = resolveLocalizedText(t, error)

  const getErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (!(err instanceof ApiError)) {
      return { translationKey: 'errors.network' }
    }

    switch (err.code) {
      case 'unauthorized':
      case 'invalid_token':
        return { translationKey: 'errors.unauthorized' }
      case 'insufficient_permissions':
        return { translationKey: 'errors.insufficientPermissions' }
      case 'route_not_found':
        return { translationKey: 'errors.routeNotFound' }
      case 'missing_route_fields':
        return { translationKey: 'errors.missingRouteFields' }
      case 'invalid_route_path_prefix':
        return { translationKey: 'errors.invalidRoutePathPrefix' }
      case 'invalid_route_path_match_mode':
        return { translationKey: 'errors.invalidRoutePathMatchMode' }
      case 'invalid_route_path_regex':
        return { translationKey: 'errors.invalidRoutePathRegex' }
      case 'reserved_route_path_prefix':
        return { translationKey: 'errors.reservedRoutePathPrefix' }
      case 'invalid_route_host':
        return { translationKey: 'errors.invalidRouteHost' }
      case 'invalid_route_backend':
        return { translationKey: 'errors.invalidRouteBackend' }
      case 'invalid_route_backend_weight':
        return { translationKey: 'errors.invalidRouteBackendWeight' }
      case 'invalid_route_redirect_code':
        return { translationKey: 'errors.invalidRouteRedirectCode' }
      case 'route_store_failure':
        return { translationKey: 'errors.routeStoreFailure' }
      default:
        return { message: err.message }
    }
  }, [])

  const getListErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (err instanceof ApiError && err.code === 'route_store_failure') {
      return { translationKey: 'errors.routeDirectoryUnavailable' }
    }

    return getErrorState(err)
  }, [getErrorState])

  const fetchRoutes = React.useCallback(async () => {
    const requestGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = requestGeneration

    try {
      setError(null)
      setRouteListUnavailable(false)
      setRouteDirectoryUnavailable(false)
      const [data, certs] = await Promise.all([
        routesApi.list(),
        certificatesApi.list().catch(() => [] as Certificate[]),
      ])
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      setRoutes(data)
      setCertificates(certs)
    } catch (err) {
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      setRoutes([])
      setRouteListUnavailable(true)
      setRouteDirectoryUnavailable(err instanceof ApiError && err.code === 'route_store_failure')
      setError(getListErrorState(err))
    } finally {
      if (requestGenerationRef.current === requestGeneration) {
        setLoading(false)
      }
    }
  }, [getListErrorState])

  React.useEffect(() => {
    fetchRoutes()
  }, [fetchRoutes])

  const handleCreate = async (data: RouteInput) => {
    try {
      await routesApi.create(data)
      setShowForm(false)
      await fetchRoutes()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleUpdate = async (data: RouteInput) => {
    if (!editingRoute) return
    try {
      await routesApi.update(editingRoute.id, data)
      setShowForm(false)
      setEditingRoute(null)
      await fetchRoutes()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleDelete = async (route: Route) => {
    if (!confirm(t('page.deleteConfirm', { name: route.name || route.path_prefix }))) return
    try {
      await routesApi.delete(route.id)
      await fetchRoutes()
    } catch (err) {
      setError(getErrorState(err))
    }
  }

  const activeCount = routes.filter((route) => route.enabled).length
  const uniqueHosts = new Set(routes.map((route) => route.host).filter(Boolean)).size
  const highestPriority = routes.reduce((max, route) => Math.max(max, route.priority), 0)

  const columns = [
    {
      key: 'name',
      header: t('table.route'),
      render: (value: string, row: Route) => (
        <div className="min-w-0">
          <div className="font-semibold text-[var(--text-primary)]">{value || t('page.untitled')}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">{row.path_prefix}</div>
        </div>
      ),
    },
    {
      key: 'host',
      header: t('table.hostMatch'),
      render: (value: string) => <span className="app-code">{value || t('page.allHosts')}</span>,
    },
    {
      key: 'backend',
      header: t('table.backendTarget'),
      render: (_value: string, row: Route) => {
        const primaryTarget = getPrimaryBackendTarget(row)
        const pooledBackendCount = row.backends?.length || 0
        const runtimePolicySummary = getRuntimePolicySummary(row, t)
        const backendTimeoutSummary = getBackendTimeoutSummary(row, t)

        return (
          <div className="min-w-0">
            <div className="app-code break-all">{primaryTarget || t('table.noBackend')}</div>
            {(pooledBackendCount > 0 || row.tls_enabled || runtimePolicySummary.length > 0 || backendTimeoutSummary.length > 0) ? (
              <div className="mt-1 flex flex-wrap gap-2 text-xs text-[var(--text-muted)]">
                {pooledBackendCount > 0 ? <span>{t('table.backendPoolCount', { count: pooledBackendCount })}</span> : null}
                {row.tls_enabled ? <span>{t('table.tlsEnabled')}</span> : null}
                {runtimePolicySummary.map((item) => <span key={item}>{item}</span>)}
                {backendTimeoutSummary.map((item) => <span key={item}>{item}</span>)}
              </div>
            ) : null}
          </div>
        )
      },
    },
    {
      key: 'priority',
      header: t('table.priority'),
      className: 'w-28',
      render: (value: number) => <span className="font-semibold text-[var(--text-secondary)]">{value}</span>,
    },
    {
      key: 'enabled',
      header: t('table.status'),
      className: 'w-36',
      render: (value: boolean) => (
        <Badge variant={value ? 'success' : 'default'} badgeSize="sm">
          {value ? t('page.active') : t('page.disabled')}
        </Badge>
      ),
    },
  ]

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          {t('page.loading')}
        </div>
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
          <>
            <Badge variant="primary">{t('page.badge')}</Badge>
            {showDirectoryMetrics ? (
              <span className="text-sm text-[var(--text-muted)]">
                {t('page.configuredEntries', { count: routes.length })}
              </span>
            ) : null}
          </>
        }
        action={
          canManageRoutes && !routeListUnavailable ? (
            <Button
              icon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingRoute(null)
                setShowForm(true)
              }}
            >
              {t('page.addRoute')}
            </Button>
          ) : null
        }
      />

      {errorMessage && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {errorMessage}
        </Alert>
      )}

      {showDirectoryMetrics ? (
        <div className="mb-6 grid gap-4 md:grid-cols-3">
          <MetricCard
            label={t('page.totalRoutes')}
            value={routes.length}
            hint={t('page.totalRoutesHint')}
            icon={<RouteIcon className="h-5 w-5" />}
            tone="primary"
          />
          <MetricCard
            label={t('page.activeRoutes')}
            value={activeCount}
            hint={t('page.activeRoutesHint')}
            icon={<ToggleLeft className="h-5 w-5" />}
            tone="accent"
          />
          <MetricCard
            label={t('page.hostCoverage')}
            value={uniqueHosts || t('page.wildcard')}
            hint={t('page.highestPriority', { count: highestPriority })}
            icon={<Server className="h-5 w-5" />}
          />
        </div>
      ) : null}

      <Card padding="lg" className="space-y-5">
        <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              {t('page.directoryEyebrow')}
            </div>
            <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
              {t('page.directoryTitle')}
            </h2>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              {t('page.directoryDescription')}
            </p>
          </div>
          <div className="inline-flex items-center gap-2 rounded-full bg-[rgba(255,255,255,0.54)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.12em] text-[var(--text-muted)]">
            <Activity className="h-3.5 w-3.5 text-[var(--primary-600)]" />
            {t('page.liveSnapshot')}
          </div>
        </div>

        {routes.length === 0 ? (
          <EmptyState
            icon={<RouteIcon className="h-8 w-8" />}
            title={
              routeDirectoryUnavailable
                ? t('page.directoryUnavailableTitle')
                : routeListUnavailable
                ? t('page.listUnavailableTitle')
                : t('page.emptyTitle')
            }
            description={
              routeDirectoryUnavailable
                ? t('page.directoryUnavailableDescription')
                : routeListUnavailable
                ? t('page.listUnavailableDescription')
                : canManageRoutes
                ? t('page.emptyDescription')
                : t('page.readOnlyEmptyDescription')
            }
            action={
              canManageRoutes && !routeListUnavailable
                ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button>
                : undefined
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={routes}
            onEdit={canManageRoutes ? (route) => { setEditingRoute(route); setShowForm(true) } : undefined}
            onDelete={canManageRoutes ? handleDelete : undefined}
          />
        )}
      </Card>

      <Modal
        open={canManageRoutes && showForm}
        onClose={() => {
          setShowForm(false)
          setEditingRoute(null)
        }}
        title={editingRoute ? t('page.editModalTitle') : t('page.addModalTitle')}
      >
        <RouteForm
          route={editingRoute}
          certificates={certificates}
          onSubmit={editingRoute ? handleUpdate : handleCreate}
          onCancel={() => {
            setShowForm(false)
            setEditingRoute(null)
          }}
        />
      </Modal>
    </div>
  )
}
