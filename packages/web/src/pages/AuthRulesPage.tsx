import React from 'react'
import { KeyRound, LockKeyhole, Plus, Shield } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AuthRuleForm } from '../components/AuthRuleForm'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { authRulesApi } from '../lib/api/auth-rules'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'

export function AuthRulesPage() {
  const { t } = useTranslation('authRules')
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
    if (!confirm(t('page.deleteConfirm'))) return
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

  const typeLabel = (value: string) => {
    switch (value) {
      case 'none':
        return t('types.none')
      case 'apikey':
        return t('types.apikey')
      case 'bearer':
        return t('types.bearer')
      case 'basic':
        return t('types.basic')
      case 'gateway':
        return t('types.gateway')
      default:
        return value
    }
  }

  const columns = [
    {
      key: 'route_id',
      header: t('table.protectedRoute'),
      render: (value: string) => <span className="font-semibold text-[var(--text-primary)]">{getRouteName(value)}</span>,
    },
    {
      key: 'type',
      header: t('table.authType'),
      className: 'w-40',
      render: (value: string) => <Badge variant="primary" badgeSize="sm">{typeLabel(value)}</Badge>,
    },
    {
      key: 'config',
      header: t('table.credentialMapping'),
      render: (value: any, row: AuthRule) => {
        if (row.type === 'apikey') {
          return value.header_name ? `${t('form.headerName')} ${value.header_name}` : t('page.headerBasedKey')
        }
        if (row.type === 'bearer') return t('page.bearerValidation')
        if (row.type === 'basic') {
          return value.username ? `${t('form.username')} ${value.username}` : t('page.basicCredentials')
        }
        if (row.type === 'gateway') return t('page.gatewayLogin')
        return t('page.noCredentials')
      },
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
              {t('page.activeRuleDefinitions', { count: rules.length })}
            </span>
          </>
        }
        action={
          canManageAuth ? (
            <Button
              icon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingRule(null)
                setShowForm(true)
              }}
            >
              {t('page.addRule')}
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
          label={t('page.protectedRoutes')}
          value={new Set(rules.map((rule) => rule.route_id)).size}
          hint={t('page.protectedRoutesHint')}
          icon={<Shield className="h-5 w-5" />}
          tone="primary"
        />
        <MetricCard
          label={t('page.apiKeyRules')}
          value={typeCounts.apikey || 0}
          hint={t('page.apiKeyRulesHint')}
          icon={<KeyRound className="h-5 w-5" />}
          tone="accent"
        />
        <MetricCard
          label={t('page.bearerAndBasic')}
          value={(typeCounts.bearer || 0) + (typeCounts.basic || 0)}
          hint={t('page.bearerAndBasicHint')}
          icon={<LockKeyhole className="h-5 w-5" />}
        />
      </div>

      <Card padding="lg" className="space-y-5">
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

        {rules.length === 0 ? (
          <EmptyState
            icon={<Shield className="h-8 w-8" />}
            title={t('page.emptyTitle')}
            description={t('page.emptyDescription')}
            action={canManageAuth ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button> : undefined}
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
        onClose={() => {
          setShowForm(false)
          setEditingRule(null)
        }}
        title={editingRule ? t('page.editModalTitle') : t('page.addModalTitle')}
      >
        <AuthRuleForm
          rule={editingRule}
          routes={routes}
          onSubmit={editingRule ? handleUpdate : handleCreate}
          onCancel={() => {
            setShowForm(false)
            setEditingRule(null)
          }}
        />
      </Modal>
    </div>
  )
}
