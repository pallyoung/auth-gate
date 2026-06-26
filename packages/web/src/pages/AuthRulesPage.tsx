import React from 'react'
import { ChevronDown, ChevronRight, KeyRound, Pencil, Plus, Shield, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AuthRuleForm } from '../components/AuthRuleForm'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { authRulesApi } from '../lib/api/auth-rules'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'

interface RouteGroup {
  routeId: string
  routeName: string
  rules: AuthRule[]
}

export function AuthRulesPage() {
  const { t } = useTranslation(['authRules', 'routes'])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [rules, setRules] = React.useState<AuthRule[]>([])
  const [loading, setLoading] = React.useState(true)
  const [showForm, setShowForm] = React.useState(false)
  const [editingRule, setEditingRule] = React.useState<AuthRule | null>(null)
  const [defaultRouteId, setDefaultRouteId] = React.useState<string | undefined>()
  const [expandedRoutes, setExpandedRoutes] = React.useState<Set<string>>(new Set())
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
      setDefaultRouteId(undefined)
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

  const openCreateForm = (routeId?: string) => {
    setEditingRule(null)
    setDefaultRouteId(routeId)
    setShowForm(true)
  }

  const openEditForm = (rule: AuthRule) => {
    setEditingRule(rule)
    setDefaultRouteId(undefined)
    setShowForm(true)
  }

  const closeForm = () => {
    setShowForm(false)
    setEditingRule(null)
    setDefaultRouteId(undefined)
  }

  const toggleRoute = (routeId: string) => {
    setExpandedRoutes((prev) => {
      const next = new Set(prev)
      if (next.has(routeId)) {
        next.delete(routeId)
      } else {
        next.add(routeId)
      }
      return next
    })
  }

  // 按路由分组（包含无规则的路由）
  const routeGroups: RouteGroup[] = React.useMemo(() => {
    const ruleMap = new Map<string, AuthRule[]>()
    for (const rule of rules) {
      const existing = ruleMap.get(rule.route_id) || []
      existing.push(rule)
      ruleMap.set(rule.route_id, existing)
    }
    const groups: RouteGroup[] = []
    for (const route of routes) {
      groups.push({
        routeId: route.id,
        routeName: route.name || route.path_prefix,
        rules: ruleMap.get(route.id) || [],
      })
    }
    return groups
  }, [routes, rules])

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
      case 'gateway':
        return t('types.gateway')
      default:
        return value
    }
  }

  const getCredentialMapping = (rule: AuthRule) => {
    if (rule.type === 'apikey') {
      return rule.config?.header_name ? `${t('form.headerName')} ${rule.config.header_name}` : t('page.headerBasedKey')
    }
    if (rule.type === 'gateway') return t('page.gatewayLogin')
    return t('page.noCredentials')
  }

  const getRuntimePolicySummary = (rule: AuthRule) => {
    const summaries = [
      rule.rate_limit || rule.burst ? t('page.rateLimitSummary', { rate: rule.rate_limit || 0, burst: rule.burst || 0 }) : null,
      rule.whitelist?.length ? t('page.whitelistSummary', { count: rule.whitelist.length }) : null,
      rule.cors_allowed_origins ? t('page.corsSummary', { origins: rule.cors_allowed_origins }) : null,
    ].filter(Boolean)
    return summaries.length > 0 ? summaries.join(' · ') : t('page.noRuntimePolicy')
  }

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
              onClick={() => openCreateForm()}
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
            value={routeGroups.filter((g) => g.rules.length > 0).length}
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

        {routeGroups.length === 0 ? (
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
                  <Button onClick={() => openCreateForm()}>{t('page.createFirst')}</Button>
                ) : (
                  <Button onClick={() => { window.location.hash = '/' }}>
                    {t('routes:page.createFirst')}
                  </Button>
                )
              ) : undefined
            }
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10" />
                <TableHead>{t('table.protectedRoute')}</TableHead>
                <TableHead>{t('table.authType')}</TableHead>
                <TableHead className="hidden md:table-cell">{t('table.runtimePolicy')}</TableHead>
                <TableHead className="w-24" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {routeGroups.map((group) => {
                const isExpanded = expandedRoutes.has(group.routeId)
                const hasRules = group.rules.length > 0

                return (
                  <React.Fragment key={group.routeId}>
                    {/* 路由主行 */}
                    <TableRow
                      className="cursor-pointer"
                      onClick={() => hasRules && toggleRoute(group.routeId)}
                    >
                      <TableCell className="w-10">
                        {hasRules ? (
                          isExpanded ? (
                            <ChevronDown className="h-4 w-4 text-[var(--text-muted)]" />
                          ) : (
                            <ChevronRight className="h-4 w-4 text-[var(--text-muted)]" />
                          )
                        ) : null}
                      </TableCell>
                      <TableCell>
                        <span className="font-semibold text-[var(--text-primary)]">{group.routeName}</span>
                      </TableCell>
                      <TableCell>
                        {hasRules ? (
                          <div className="flex flex-wrap gap-1">
                            {group.rules.map((rule) => (
                              <Badge key={rule.id} variant="primary" badgeSize="sm">
                                {typeLabel(rule.type)}
                              </Badge>
                            ))}
                          </div>
                        ) : (
                          <span className="text-sm text-[var(--text-muted)]">{t('page.noAuthMethods')}</span>
                        )}
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        {hasRules ? (
                          <span className="text-sm text-[var(--text-muted)]">
                            {t('page.ruleCount', { count: group.rules.length })}
                          </span>
                        ) : null}
                      </TableCell>
                      <TableCell className="w-24">
                        {canManageAuth && !routeListUnavailable ? (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              openCreateForm(group.routeId)
                            }}
                            title={t('page.addRuleForRoute')}
                          >
                            <Plus className="h-4 w-4" />
                          </Button>
                        ) : null}
                      </TableCell>
                    </TableRow>

                    {/* 展开的规则子行 */}
                    {isExpanded && group.rules.map((rule) => (
                      <TableRow key={rule.id} className="bg-[var(--bg-card-soft)]/50">
                        <TableCell />
                        <TableCell className="pl-10">
                          <div className="flex items-center gap-2">
                            <Badge variant="primary" badgeSize="sm">{typeLabel(rule.type)}</Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-[var(--text-muted)]">{getCredentialMapping(rule)}</span>
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          <span className="text-sm text-[var(--text-muted)]">{getRuntimePolicySummary(rule)}</span>
                        </TableCell>
                        <TableCell>
                          {canManageAuth ? (
                            <div className="flex items-center gap-1">
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => openEditForm(rule)}
                                title={t('common:actions.edit', 'Edit')}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleDelete(rule)}
                                className="text-[var(--text-error)]"
                                title={t('common:actions.delete', 'Delete')}
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                            </div>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    ))}
                  </React.Fragment>
                )
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      <Modal
        open={canManageAuth && showForm}
        onClose={closeForm}
        title={editingRule ? t('page.editModalTitle') : t('page.addModalTitle')}
      >
        <AuthRuleForm
          rule={editingRule}
          routes={routes}
          defaultRouteId={defaultRouteId}
          onSubmit={editingRule ? handleUpdate : handleCreate}
          onCancel={closeForm}
        />
      </Modal>
    </div>
  )
}
