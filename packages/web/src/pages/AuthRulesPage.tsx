import React from 'react'
import { Button, Card, Badge, Modal, EmptyState } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { DataTable } from '../components/DataTable'
import { AuthRule, Route, api } from '../lib/api'
import { AuthRuleForm } from '../components/AuthRuleForm'
import { Plus, Shield } from 'lucide-react'

export function AuthRulesPage() {
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [rules, setRules] = React.useState<AuthRule[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<AuthRule | null>(null)
  const [error, setError] = React.useState('')

  const fetchData = React.useCallback(async () => {
    try {
      setError('')
      const [routesRes, rulesRes] = await Promise.all([
        api.routes.list(),
        api.authRules.list(),
      ])
      setRoutes(routesRes)
      setRules(rulesRes)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleDelete = async (rule: AuthRule) => {
    if (!confirm('Delete this auth rule?')) return
    try {
      await api.authRules.delete(rule.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleCreate = async (data: Partial<AuthRule>) => {
    try {
      await api.authRules.create(data)
      setShowForm(false)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleUpdate = async (data: Partial<AuthRule>) => {
    if (!editingRule) return
    try {
      await api.authRules.update(editingRule.id, data)
      setShowForm(false)
      setEditingRule(null)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
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
        if (row.type === 'apikey') return v.header_name ? `Header: ${v.header_name}` : 'API Key'
        if (row.type === 'bearer') return 'Bearer Token'
        if (row.type === 'basic') return v.username ? `User: ${v.username}` : 'Basic Auth'
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
      {error && <div className="mb-4 p-3 rounded-[var(--radius-md)] bg-[var(--error-light)] text-[var(--error)]">{error}</div>}
      <Card padding="none">
        {rules.length === 0 ? (
          <EmptyState icon={<Shield className="w-12 h-12" />} title="No auth rules configured" description="Add authentication rules to protect your routes" action={<Button onClick={() => setShowForm(true)}>Add Rule</Button>} />
        ) : (
          <DataTable columns={columns} data={rules} onEdit={(r) => { setEditingRule(r); setShowForm(true) }} onDelete={handleDelete} />
        )}
      </Card>

      <Modal
        open={showForm}
        onClose={() => { setShowForm(false); setEditingRule(null) }}
        title={editingRule ? 'Edit Auth Rule' : 'Add Auth Rule'}
      >
        <AuthRuleForm
          rule={editingRule}
          routes={routes}
          onSubmit={editingRule ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditingRule(null) }}
        />
      </Modal>
    </div>
  )
}
