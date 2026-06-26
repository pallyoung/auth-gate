import React from 'react'
import { Activity, Plus, Route as RouteIcon, Server, Shield, ToggleLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { RouteForm } from '../components/RouteForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { routesApi } from '../lib/api/routes'
import { certificatesApi } from '../lib/api/certificates'
import { authRulesApi } from '../lib/api/auth-rules'
import { getSessionUser } from '../lib/session-store'
import type { AuthRule, Certificate, Route, RouteInput } from '../lib/api/types'

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
  const [authRules, setAuthRules] = React.useState<AuthRule[]>([])
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
      const [data, certs, rules] = await Promise.all([
        routesApi.list(),
        certificatesApi.list().catch(() => [] as Certificate[]),
        authRulesApi.list().catch(() => [] as AuthRule[]),
      ])
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      setRoutes(data)
      setCertificates(certs)
      setAuthRules(rules)
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
      key: 'host',
      header: t('table.host'),
      render: (value: string) => (
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full bg-[var(--primary-500)]" />
          <span className="app-code">{value || '*'}</span>
        </div>
      ),
    },
    {
      key: 'path_prefix',
      header: t('table.pathPrefix'),
      render: (value: string) => (
        <span className="app-code text-[var(--primary-600)]">{value}</span>
      ),
    },
    {
      key: 'backend',
      header: t('table.backendTarget'),
      render: (_value: string, row: Route) => {
        const primaryTarget = getPrimaryBackendTarget(row)
        return <span className="app-code">{primaryTarget || t('table.noBackend')}</span>
      },
    },
    {
      key: 'enabled',
      header: t('table.status'),
      className: 'w-36',
      render: (value: boolean, row: Route) => (
        <Badge variant={value ? 'success' : 'default'} badgeSize="sm">
          {value ? t('page.active') : t('page.disabled')}
        </Badge>
      ),
    },
    {
      key: 'tls_enabled',
      header: t('table.authType'),
      className: 'w-32',
      render: (_value: boolean, row: Route) => (
        <Badge variant={row.tls_enabled ? 'primary' : 'default'} badgeSize="sm">
          {row.tls_enabled ? 'TLS' : row.certificate_id ? 'Cert' : 'Public'}
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
        <div className="mb-6 grid gap-4 md:grid-cols-4">
          <MetricCard
            label={t('page.totalRoutes')}
            value={routes.length}
            hint={t('page.totalRoutesHint')}
            icon={<RouteIcon className="h-5 w-5" />}
            tone="primary"
          />
          <MetricCard
            label={t('page.activeRoutes')}
            value={`${activeCount}/${routes.length}`}
            hint="Unlicensed"
            icon={<Server className="h-5 w-5" />}
            tone="accent"
          />
          <MetricCard
            label="Auth Rules"
            value={authRules.length}
            hint="Enabled"
            icon={<Shield className="h-5 w-5" />}
            tone="warning"
          />
          <MetricCard
            label="Certificates"
            value={certificates.length}
            hint="All Valid"
            icon={<Shield className="h-5 w-5" />}
            tone="primary"
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
          <div className="inline-flex items-center gap-2 rounded-full bg-[var(--bg-hover)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.12em] text-[var(--text-muted)]">
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
