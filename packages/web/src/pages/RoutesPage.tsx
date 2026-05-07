import React from 'react'
import { Activity, Plus, Route as RouteIcon, Server, ToggleLeft } from 'lucide-react'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { RouteForm } from '../components/RouteForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { Route, RouteInput } from '../lib/api/types'

export function RoutesPage() {
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
    if (!confirm(`Delete route "${route.name || route.path_prefix}"?`)) return
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
      header: 'Route',
      render: (value: string, row: Route) => (
        <div className="min-w-0">
          <div className="font-semibold text-[var(--text-primary)]">{value || 'Untitled Route'}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">{row.path_prefix}</div>
        </div>
      ),
    },
    {
      key: 'host',
      header: 'Host Match',
      render: (value: string) => <span className="app-code">{value || 'all hosts'}</span>,
    },
    {
      key: 'backend',
      header: 'Backend Target',
      render: (value: string) => <span className="app-code">{value}</span>,
    },
    {
      key: 'priority',
      header: 'Priority',
      className: 'w-28',
      render: (value: number) => <span className="font-semibold text-[var(--text-secondary)]">{value}</span>,
    },
    {
      key: 'enabled',
      header: 'Status',
      className: 'w-36',
      render: (value: boolean) => (
        <Badge variant={value ? 'success' : 'default'} badgeSize="sm">
          {value ? 'Active' : 'Disabled'}
        </Badge>
      ),
    },
  ]

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          Loading routes...
        </div>
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="Traffic Orchestration"
        title="Routes"
        description="Define how inbound requests are matched and forwarded across your services."
        meta={
          <>
            <Badge variant="primary">Routing Matrix</Badge>
            <span className="text-sm text-[var(--text-muted)]">{routes.length} configured route entries</span>
          </>
        }
        action={
          canManageRoutes ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => { setEditingRoute(null); setShowForm(true) }}>
              Add Route
            </Button>
          ) : null
        }
      />

      {error && (
        <Alert variant="error" title="Route operation failed" className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard label="Total Routes" value={routes.length} hint="All configured forwarding rules." icon={<RouteIcon className="h-5 w-5" />} tone="primary" />
        <MetricCard label="Active Routes" value={activeCount} hint="Enabled rules currently available to runtime." icon={<ToggleLeft className="h-5 w-5" />} tone="accent" />
        <MetricCard label="Host Coverage" value={uniqueHosts || 'Wildcard'} hint={`Highest priority ${highestPriority}`} icon={<Server className="h-5 w-5" />} />
      </div>

      <Card padding="lg" className="space-y-5">
        <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              Route Directory
            </div>
            <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
              Forwarding topology
            </h2>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              Inspect route health, backend targets, and match precedence at a glance.
            </p>
          </div>
          <div className="inline-flex items-center gap-2 rounded-full bg-[rgba(255,255,255,0.54)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.12em] text-[var(--text-muted)]">
            <Activity className="h-3.5 w-3.5 text-[var(--primary-600)]" />
            Live config snapshot
          </div>
        </div>

        {routes.length === 0 ? (
          <EmptyState
            icon={<RouteIcon className="h-8 w-8" />}
            title="No routes configured"
            description="Create your first route to start forwarding traffic into protected backend services."
            action={canManageRoutes ? <Button onClick={() => setShowForm(true)}>Create First Route</Button> : undefined}
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
        onClose={() => { setShowForm(false); setEditingRoute(null) }}
        title={editingRoute ? 'Edit Route' : 'Add Route'}
      >
        <RouteForm
          route={editingRoute}
          onSubmit={editingRoute ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditingRoute(null) }}
        />
      </Modal>
    </div>
  )
}
