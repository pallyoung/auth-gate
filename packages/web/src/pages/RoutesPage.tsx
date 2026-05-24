import React from 'react'
import { Activity, Plus, Route as RouteIcon, Server, ToggleLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { RouteForm } from '../components/RouteForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { Route, RouteInput } from '../lib/api/types'

export function RoutesPage() {
  const { t } = useTranslation('routes')
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRoute, setEditingRoute] = React.useState<Route | null>(null)
  const [error, setError] = React.useState('')
  const canManageRoutes = getSessionUser()?.permissions?.can_manage_routes ?? false

  const fetchRoutes = React.useCallback(async () => {
    try {
      setError('')
      const data = await routesApi.list()
      setRoutes(data)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchRoutes()
  }, [fetchRoutes])

  const handleCreate = async (data: RouteInput) => {
    try {
      await routesApi.create(data)
      setShowForm(false)
      await fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleUpdate = async (data: RouteInput) => {
    if (!editingRoute) return
    try {
      await routesApi.update(editingRoute.id, data)
      setShowForm(false)
      setEditingRoute(null)
      await fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleDelete = async (route: Route) => {
    if (!confirm(t('page.deleteConfirm', { name: route.name || route.path_prefix }))) return
    try {
      await routesApi.delete(route.id)
      await fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
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
      render: (value: string) => <span className="app-code">{value}</span>,
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
            <span className="text-sm text-[var(--text-muted)]">
              {t('page.configuredEntries', { count: routes.length })}
            </span>
          </>
        }
        action={
          canManageRoutes ? (
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

      {error && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {error}
        </Alert>
      )}

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
            title={t('page.emptyTitle')}
            description={t('page.emptyDescription')}
            action={
              canManageRoutes ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button> : undefined
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
