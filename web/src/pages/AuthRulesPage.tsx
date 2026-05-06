import React from 'react'
import { Button, Card, Badge, Modal, EmptyState } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { DataTable } from '../components/DataTable'
import { AuthRule, Route } from '../lib/api'
import { Plus, Shield, KeyRound } from 'lucide-react'

export function AuthRulesPage() {
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [rules, setRules] = React.useState<AuthRule[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<AuthRule | null>(null)

  const fetchData = React.useCallback(async () => {
    try {
      const [routesRes, rulesRes] = await Promise.all([
        fetch('/api/routes'),
        fetch('/api/auth-rules'),
      ])
      setRoutes(await routesRes.json())
      setRules(await rulesRes.json())
    } catch (e) {
      console.error('Failed to fetch data:', e)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleDelete = async (rule: AuthRule) => {
    if (!confirm('Delete this auth rule?')) return
    await fetch(`/api/auth-rules/${rule.id}`, { method: 'DELETE' })
    fetchData()
  }

  const getRouteName = (routeId: string) => {
    const route = routes.find(r => r.id === routeId)
    return route?.name || route?.path_prefix || routeId
  }

  const columns = [
    { key: 'route_id', header: 'Route', render: (v: string) => getRouteName(v) },
    { key: 'type', header: 'Type', render: (v: string) => <Badge variant="primary" badgeSize="sm">{v}</Badge> },
    {
      key: 'config', header: 'Configuration', render: (v: any, row: AuthRule) => {
        if (row.type === 'apikey') return v.header_name ? 'Header: ${v.header_name}' : 'API Key'
        if (row.type === 'bearer') return 'Bearer Token'
        if (row.type === 'basic') return 'User: ${v.username}'
        return '-'
      }
    },
  ]

  if (loading) {
    return <div className="flex items-center justify-center h-64"><div className="text-[var(--text-muted)]">Loading...</div></div>
  }

  return (
    <div className="animate-fade-in">
      <PageHeader title="Auth Rules" description="Configure authentication for your routes" action={
        <Button icon={<Plus className="w-4 h-4" />} onClick={() => { setEditingRule(null); setShowForm(true) }}>Add Rule</Button>
      } />
      <Card padding="none">
        {rules.length === 0 ? (
          <EmptyState icon={<Shield className="w-12 h-12" />} title="No auth rules configured" description="Add authentication rules to protect your routes" action={<Button onClick={() => setShowForm(true)}>Add Rule</Button>} />
        ) : (
          <DataTable columns={columns} data={rules} onEdit={(r) => { setEditingRule(r); setShowForm(true) }} onDelete={handleDelete} />
        )}
      </Card>
    </div>
  )
}
