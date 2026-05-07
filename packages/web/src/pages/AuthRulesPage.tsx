import React from 'react'
import { KeyRound, LockKeyhole, Plus, Shield } from 'lucide-react'
import { AuthRuleForm } from '../components/AuthRuleForm'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { authRulesApi } from '../lib/api/auth-rules'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'

export function AuthRulesPage() {
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [rules, setRules] = React.useState<AuthRule[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<AuthRule | null>(null)
  const [error, setError] = React.useState('')
  const canManageAuth = getSessionUser()?.permissions?.can_manage_auth ?? false

  const fetchData = React.useCallback(async () => {
    try {
      setError('')
      const [routesRes, rulesRes] = await Promise.all([routesApi.list(), authRulesApi.list()])
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
      await authRulesApi.delete(rule.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleCreate = async (data: AuthRuleInput) => {
    try {
      await authRulesApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleUpdate = async (data: AuthRuleInput) => {
    if (!editingRule) return
    try {
      await authRulesApi.update(editingRule.id, data)
      setShowForm(false)
      setEditingRule(null)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const getRouteName = (routeId: string) => {
    const route = routes.find((item) => item.id === routeId)
    return route?.name || route?.path_prefix || routeId
  }

  const typeCounts = rules.reduce<Record<string, number>>((acc, rule) => {
    acc[rule.type] = (acc[rule.type] || 0) + 1
    return acc
  }, {})

  const columns = [
    {
      key: 'route_id',
      header: 'Protected Route',
      render: (value: string) => <span className="font-semibold text-[var(--text-primary)]">{getRouteName(value)}</span>,
    },
    {
      key: 'type',
      header: 'Auth Type',
      className: 'w-40',
      render: (value: string) => <Badge variant="primary" badgeSize="sm">{value}</Badge>,
    },
    {
      key: 'config',
      header: 'Credential Mapping',
      render: (value: any, row: AuthRule) => {
        if (row.type === 'apikey') return value.header_name ? `Header ${value.header_name}` : 'Header based key'
        if (row.type === 'bearer') return 'Bearer token validation'
        if (row.type === 'basic') return value.username ? `User ${value.username}` : 'Basic credentials'
        return 'No credential requirement'
      },
    },
  ]

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          Loading auth rules...
        </div>
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="Policy Enforcement"
        title="Auth Rules"
        description="Attach route-level credentials and keep sensitive paths behind explicit verification rules."
        meta={
          <>
            <Badge variant="primary">Security Policies</Badge>
            <span className="text-sm text-[var(--text-muted)]">{rules.length} active rule definitions</span>
          </>
        }
        action={
          canManageAuth ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => { setEditingRule(null); setShowForm(true) }}>
              Add Rule
            </Button>
          ) : null
        }
      />

      {error && (
        <Alert variant="error" title="Auth rule operation failed" className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard label="Protected Routes" value={new Set(rules.map((rule) => rule.route_id)).size} hint="Unique routes with attached auth policies." icon={<Shield className="h-5 w-5" />} tone="primary" />
        <MetricCard label="API Key Rules" value={typeCounts.apikey || 0} hint="Header-based key validation rules." icon={<KeyRound className="h-5 w-5" />} tone="accent" />
        <MetricCard label="Bearer + Basic" value={(typeCounts.bearer || 0) + (typeCounts.basic || 0)} hint="Token and credential verification rules." icon={<LockKeyhole className="h-5 w-5" />} />
      </div>

      <Card padding="lg" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Policy Directory
          </div>
          <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
            Authentication matrix
          </h2>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Audit which routes are protected and how credentials are validated before traffic reaches upstream services.
          </p>
        </div>

        {rules.length === 0 ? (
          <EmptyState
            icon={<Shield className="h-8 w-8" />}
            title="No auth rules configured"
            description="Add a rule to require API keys, bearer tokens, or basic auth before requests are forwarded."
            action={canManageAuth ? <Button onClick={() => setShowForm(true)}>Create First Rule</Button> : undefined}
          />
        ) : (
          <DataTable
            columns={columns}
            data={rules}
            onEdit={canManageAuth ? (rule) => { setEditingRule(rule); setShowForm(true) } : undefined}
            onDelete={canManageAuth ? handleDelete : undefined}
          />
        )}
      </Card>

      <Modal
        open={canManageAuth && showForm}
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
