import React from 'react'
import { Button, Card, Badge, Modal, EmptyState } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { DataTable } from '../components/DataTable'
import { RouteForm } from '../components/RouteForm'
import { api, Route } from '../lib/api'
import { Plus, Route as RouteIcon } from 'lucide-react'

export function RoutesPage() {
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRoute, setEditingRoute] = React.useState<Route | null>(null)
  const [error, setError] = React.useState('')

  const fetchRoutes = React.useCallback(async () => {
    try {
      const data = await api.routes.list()
      setRoutes(data)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => { fetchRoutes() }, [fetchRoutes])

  const handleCreate = async (data: Partial<Route>) => {
    try {
      await api.routes.create(data)
      setShowForm(false)
      fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleUpdate = async (data: Partial<Route>) => {
    if (!editingRoute) return
    try {
      await api.routes.update(editingRoute.id, data)
      setShowForm(false)
      setEditingRoute(null)
      fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleDelete = async (route: Route) => {
    if (!confirm('Delete route "' + (route.name || route.path_prefix) + '"?')) return
    try {
      await api.routes.delete(route.id)
      fetchRoutes()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const columns = [
    { key: 'name', header: 'Name', render: (v: string) => v || '-' },
    { key: 'host', header: 'Host', render: (v: string) => <span className="text-[var(--text-muted)]">{v || '*'}</span> },
    { key: 'path_prefix', header: 'Path', render: (v: string) => <code className="text-sm bg-[var(--neutral-100)] px-1.5 py-0.5 rounded">{v}</code> },
    { key: 'backend', header: 'Backend', render: (v: string) => <code className="text-sm text-[var(--text-muted)]">{v}</code> },
    { key: 'enabled', header: 'Status', render: (v: boolean) => <Badge variant={v ? 'success' : 'default'} badgeSize="sm">{v ? 'Active' : 'Disabled'}</Badge>, className: 'w-24' },
  ]

  if (loading) {
    return <div className="flex items-center justify-center h-64"><div className="animate-spin w-8 h-8 border-2 border-[var(--primary-500)] border-t-transparent rounded-full" /></div>
  }

  return (
    <div className="animate-fade-in">
      <PageHeader title="Routes" description="Configure routing rules for your services"
        action={<Button icon={<Plus className="w-4 h-4" />} onClick={() => { setEditingRoute(null); setShowForm(true) }}>Add Route</Button>} />
      
      {error && <div className="mb-4 p-3 rounded-[var(--radius-md)] bg-[var(--error-light)] text-[var(--error)]">{error}</div>}
      
      <Card padding="none">
        {routes.length === 0 ? (
          <EmptyState icon={<RouteIcon className="w-12 h-12" />} title="No routes configured"
            description="Add your first route to start routing traffic to your services"
            action={<Button onClick={() => setShowForm(true)}>Add Route</Button>} />
        ) : (
          <DataTable columns={columns} data={routes}
            onEdit={(r) => { setEditingRoute(r); setShowForm(true) }}
            onDelete={handleDelete} />
        )}
      </Card>

      <Modal open={showForm} onClose={() => { setShowForm(false); setEditingRoute(null) }}
        title={editingRoute ? 'Edit Route' : 'Add Route'}>
        <RouteForm route={editingRoute} onSubmit={editingRoute ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditingRoute(null) }} />
      </Modal>
    </div>
  )
}
