import React from 'react'
import { KeyRound, LockKeyhole, Plus, Shield } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AuthRuleForm } from '../components/AuthRuleForm'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { authRulesApi } from '../lib/api/auth-rules'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'

export function AuthRulesPage() {
  const { t } = useTranslation(['authRules', 'routes'])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [rules, setRules] = React.useState<AuthRule[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<AuthRule | null>(null)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [authRuleListUnavailable, setAuthRuleListUnavailable] = React.useState(false)
  const [authRuleDirectoryUnavailable, setAuthRuleDirectoryUnavailable] = React.useState(false)
  const [routeListUnavailable, setRouteListUnavailable] = React.useState(false)
  const canManageAuth = getSessionUser()?.permissions?.can_manage_auth ?? false
  const requestGenerationRef = React.useRef(0)
  const hasAvailableRoutes = routes.length > 0
  const showDirectoryMetrics = !authRuleListUnavailable
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
      case 'auth_rule_not_found':
        return { translationKey: 'errors.authRuleNotFound' }
      case 'route_not_found':
        return { translationKey: 'errors.routeNotFound' }
      case 'route_id_required':
        return { translationKey: 'errors.routeIDRequired' }
      case 'invalid_auth_rule_type':
        return { translationKey: 'errors.invalidAuthRuleType' }
      case 'missing_apikey_secret':
        return { translationKey: 'errors.missingAPIKeySecret' }
      case 'missing_bearer_secret':
        return { translationKey: 'errors.missingBearerSecret' }
      case 'missing_basic_credentials':
        return { translationKey: 'errors.missingBasicCredentials' }
      case 'duplicate_route_auth_rule':
        return { translationKey: 'errors.duplicateRouteAuthRule' }
      case 'route_store_failure':
        return { translationKey: 'errors.routeStoreFailure' }
      case 'auth_rule_store_failure':
        return { translationKey: 'errors.authRuleStoreFailure' }
      default:
        return { message: err.message }
    }
  }, [])

  const getListErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (err instanceof ApiError && err.code === 'auth_rule_store_failure') {
      return { translationKey: 'errors.authRuleDirectoryUnavailable' }
    }

    return getErrorState(err)
  }, [getErrorState])

  const fetchData = React.useCallback(async () => {
    const requestGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = requestGeneration

    try {
      setError(null)
      setAuthRuleListUnavailable(false)
      setAuthRuleDirectoryUnavailable(false)
      setRouteListUnavailable(false)
      const [routesResult, rulesResult] = await Promise.allSettled([routesApi.list(), authRulesApi.list()])

      if (requestGenerationRef.current !== requestGeneration) {
        return
      }

      if (routesResult.status === 'fulfilled') {
        setRoutes(routesResult.value)
      } else {
        setRoutes([])
        setRouteListUnavailable(true)
      }

      if (rulesResult.status === 'fulfilled') {
        setRules(rulesResult.value)
      } else {
        setRules([])
        setAuthRuleListUnavailable(true)
        setAuthRuleDirectoryUnavailable(
          rulesResult.reason instanceof ApiError && rulesResult.reason.code === 'auth_rule_store_failure'
        )
      }

      if (rulesResult.status === 'rejected') {
        setError(getListErrorState(rulesResult.reason))
        return
      }

      if (routesResult.status === 'rejected') {
        setError(getErrorState(routesResult.reason))
      }
    } finally {
      if (requestGenerationRef.current === requestGeneration) {
        setLoading(false)
      }
    }
  }, [getErrorState, getListErrorState])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleDelete = async (rule: AuthRule) => {
    if (!confirm(t('page.deleteConfirm'))) return
    try {
      await authRulesApi.delete(rule.id)
      await fetchData()
    } catch (err) {
      setError(getErrorState(err))
    }
  }

  const handleCreate = async (data: AuthRuleInput) => {
    try {
      await authRulesApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleUpdate = async (data: AuthRuleInput) => {
    if (!editingRule) return
    try {
      await authRulesApi.update(editingRule.id, data)
      setShowForm(false)
      setEditingRule(null)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
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
    {
      key: 'id',
      header: t('table.runtimePolicy'),
      render: (_value: string, row: AuthRule) => {
        const summaries = [
          row.rate_limit || row.burst ? t('page.rateLimitSummary', { rate: row.rate_limit || 0, burst: row.burst || 0 }) : null,
          row.whitelist?.length ? t('page.whitelistSummary', { count: row.whitelist.length }) : null,
          row.cors_allowed_origins ? t('page.corsSummary', { origins: row.cors_allowed_origins }) : null,
        ].filter(Boolean)

        if (summaries.length === 0) {
          return <span className="text-sm text-[var(--text-muted)]">{t('page.noRuntimePolicy')}</span>
        }

        return (
          <div className="space-y-1 text-sm text-[var(--text-primary)]">
            {summaries.map((summary) => (
              <div key={summary}>{summary}</div>
            ))}
          </div>
        )
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
            {showDirectoryMetrics ? (
              <span className="text-sm text-[var(--text-muted)]">
                {t('page.activeRuleDefinitions', { count: rules.length })}
              </span>
            ) : null}
          </>
        }
        action={
          canManageAuth && hasAvailableRoutes && !authRuleListUnavailable ? (
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

      {errorMessage && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {errorMessage}
        </Alert>
      )}

      {showDirectoryMetrics ? (
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
      ) : null}

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
            title={
              authRuleDirectoryUnavailable
                ? t('page.directoryUnavailableTitle')
                : authRuleListUnavailable
                ? t('page.listUnavailableTitle')
                : t('page.emptyTitle')
            }
            description={
              authRuleDirectoryUnavailable
                ? t('page.directoryUnavailableDescription')
                : authRuleListUnavailable
                ? t('page.listUnavailableDescription')
                : canManageAuth
                ? routeListUnavailable
                  ? t('page.routeLoadUnavailableDescription')
                  : hasAvailableRoutes
                  ? t('page.emptyDescription')
                  : t('page.noRoutesDescription')
                : t('page.readOnlyEmptyDescription')
            }
            action={
              canManageAuth && !authRuleListUnavailable ? (
                routeListUnavailable ? undefined : hasAvailableRoutes ? (
                  <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button>
                ) : (
                  <Button onClick={() => { window.location.hash = '/' }}>
                    {t('routes:page.createFirst')}
                  </Button>
                )
              ) : undefined
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={rules}
            onEdit={
              canManageAuth && !routeListUnavailable
                ? (rule) => {
                    setEditingRule(rule)
                    setShowForm(true)
                  }
                : undefined
            }
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
