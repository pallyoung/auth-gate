import React from 'react'
import { useTranslation } from 'react-i18next'
import { ScrollText, Activity, AlertTriangle, Clock, UserX, BarChart3, List, RefreshCw, Trash2, X } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { MetricCard } from '../components/ui/MetricCard'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Select } from '../components/ui/Select'
import { Button } from '../components/ui/Button'
import { Pagination } from '../components/ui/Pagination'
import { AccessLogFilters } from '../components/AccessLogFilters'
import { AccessLogTimeRange } from '../components/AccessLogTimeRange'
import { AccessLogDetail } from '../components/AccessLogDetail'
import { AccessLogTable } from '../components/AccessLogTable'
import { AccessLogAggregateTable } from '../components/AccessLogAggregateTable'
import { AccessLogCleanup } from '../components/AccessLogCleanup'
import { accessLogsApi } from '../lib/api'
import { getSessionUser, subscribeSession } from '../lib/session-store'
import { useAccessLogView } from '../lib/access-log-view'
import { presetTimeRange } from '../lib/access-log-time'
import { accessLogErrorKey } from '../lib/access-log-errors'
import type { AccessLogEntry, AccessLogListResponse, AccessLogStats, AccessLogQueryParams, AccessLogAggregateResponse, AccessLogAggregateGroup, AccessLogAggregateParams } from '../lib/api/types'

const AccessLogCharts = React.lazy(() => import('../components/AccessLogCharts').then((m) => ({ default: m.AccessLogCharts })))
const getRole = () => getSessionUser()?.role || ''

export function AccessLogsPage() {
  const { t, i18n } = useTranslation('accessLogs')
  const [state, setState] = useAccessLogView()
  const { filters, view, groupBy, sortBy, aggregatePage, preset } = state
  const isAdmin = React.useSyncExternalStore(subscribeSession, getRole, getRole) === 'admin'
  const [list, setList] = React.useState<AccessLogListResponse | null>(null)
  const [stats, setStats] = React.useState<AccessLogStats | null>(null)
  const [aggregate, setAggregate] = React.useState<AccessLogAggregateResponse | null>(null)
  const [loading, setLoading] = React.useState(true)
  const [statsLoading, setStatsLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [statsError, setStatsError] = React.useState<string | null>(null)
  const [selected, setSelected] = React.useState<AccessLogEntry | null>(null)
  const [cleanupFilters, setCleanupFilters] = React.useState<AccessLogQueryParams | null>(null)
  const [filterDraftPending, setFilterDraftPending] = React.useState(false)
  const [removed, setRemoved] = React.useState<number | null>(null)
  const [refresh, setRefresh] = React.useState(0)
  const generation = React.useRef(0)

  React.useEffect(() => {
    const id = ++generation.current
    setLoading(true)
    setStatsLoading(true)
    setError(null)
    setStatsError(null)
    setStats(null)
    const isCurrent = () => generation.current === id
    const statsRequest = accessLogsApi.stats(filters)
      .then((result) => { if (isCurrent()) setStats(result) })
      .catch((err) => { if (isCurrent()) setStatsError(accessLogErrorKey(err, 'stats')) })
      .finally(() => { if (isCurrent()) setStatsLoading(false) })
    const dataRequest = view === 'detail'
      ? accessLogsApi.list(filters).then((result) => {
        if (!isCurrent()) return
        if (result.total_pages > 0 && (filters.page || 1) > result.total_pages) {
          setState((current) => ({ ...current, filters: { ...current.filters, page: result.total_pages } }))
          return
        }
        setList(result)
      })
      : accessLogsApi.aggregate({ ...filters, group_by: groupBy, sort_by: sortBy, page: aggregatePage, per_page: 20 }).then((result) => {
        if (!isCurrent()) return
        if (result.total_pages > 0 && aggregatePage > result.total_pages) {
          setState((current) => ({ ...current, aggregatePage: result.total_pages }))
          return
        }
        setAggregate(result)
      })
    void Promise.allSettled([statsRequest, dataRequest
      .catch((err) => { if (isCurrent()) setError(accessLogErrorKey(err, view === 'detail' ? 'load' : 'aggregate')) })
      .finally(() => { if (isCurrent()) setLoading(false) })])
    return () => { generation.current += 1 }
  }, [filters, view, groupBy, sortBy, aggregatePage, refresh, setState])

  const updateFilters = (next: AccessLogQueryParams) => {
    setState((current) => ({ ...current, filters: { ...next, page: 1 }, aggregatePage: 1 }))
    setSelected(null)
    setRemoved(null)
  }
  const refreshData = () => {
    if (preset !== 'custom') {
      setState((current) => ({ ...current, filters: { ...current.filters, start_time: undefined, end_time: undefined, ...presetTimeRange(preset) } }))
    } else setRefresh((n) => n + 1)
  }
  const drillDown = (group: AccessLogAggregateGroup) => {
    if (!group.key) return
    setState((current) => ({ ...current, view: 'detail', filters: { ...current.filters, page: 1, [groupBy]: groupBy === 'status_code' ? Number(group.key) : group.key } }))
  }
  const loadingIndicator = <div role="status" className="flex min-h-48 items-center justify-center gap-3 text-sm text-[var(--text-muted)]"><span className="h-5 w-5 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" aria-hidden="true" />{t('page.loading')}</div>
  const errorAlert = (key: string) => <div role="alert"><Alert variant="error">{t(key)} <Button variant="ghost" size="sm" onClick={refreshData}>{t('page.retry')}</Button></Alert></div>
  const number = (value: number) => value.toLocaleString(i18n.resolvedLanguage)
  const errorRate = stats && stats.total_requests > 0 ? stats.error_count / stats.total_requests * 100 : 0
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const date = (value: string) => new Date(value).toLocaleString(i18n.resolvedLanguage)
  const appliedRange = filters.start_time && filters.end_time
    ? t('time.active', { start: date(filters.start_time), end: date(filters.end_time), zone })
    : filters.start_time ? t('time.from', { start: date(filters.start_time) })
      : filters.end_time ? t('time.until', { end: date(filters.end_time) }) : t('time.all')

  return <div className="animate-rise-in space-y-6">
    <PageHeader eyebrow={t('page.eyebrow')} title={t('page.title')} description={t('page.description')} meta={<Badge>{t('page.badge')}</Badge>} />
    <Card padding="lg" className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div><h2 className="font-semibold text-[var(--text-primary)]">{t('filters.title')}</h2><p className="mt-1 text-sm text-[var(--text-muted)]">{t('filters.description')}</p></div>
        <div className="flex flex-wrap gap-2">
          <Button variant="ghost" size="sm" onClick={() => { setState((current) => ({ ...current, filters: { page: 1, per_page: 20, ...presetTimeRange('24h') }, preset: '24h', aggregatePage: 1 })); setSelected(null); setRemoved(null) }}>{t('filters.reset')}</Button>
          <Button variant="secondary" size="sm" icon={<RefreshCw className="h-4 w-4" />} onClick={refreshData} disabled={loading || statsLoading}>{t('page.refresh')}</Button>
          {isAdmin && <Button variant="secondary" size="sm" icon={<Trash2 className="h-4 w-4" />} disabled={filterDraftPending} onClick={() => setCleanupFilters({ ...filters })}>{t('cleanup.action')}</Button>}
        </div>
      </div>
      <AccessLogFilters filters={filters} onChange={updateFilters} onPendingChange={setFilterDraftPending} />
      <AccessLogTimeRange filters={filters} preset={preset} onApply={(range, nextPreset) => {
        setState((current) => ({ ...current, filters: { ...current.filters, start_time: undefined, end_time: undefined, ...range, page: 1 }, preset: nextPreset, aggregatePage: 1 }))
        setSelected(null)
        setRemoved(null)
      }} />
      <p className="text-xs text-[var(--text-muted)]">{appliedRange}</p>
      {(filters.route_id || filters.status_code) && <div className="flex flex-wrap gap-2">{(['route_id', 'status_code'] as const).map((key) => filters[key] ? <Button key={key} size="sm" variant="ghost" icon={<X className="h-3 w-3" />} aria-label={t('filters.clearField', { field: t(key === 'route_id' ? 'filters.route' : 'filters.statusCode') })} onClick={() => { const next = { ...filters }; delete next[key]; updateFilters(next) }}>{t(key === 'route_id' ? 'filters.route' : 'filters.statusCode')}: {filters[key]}</Button> : null)}</div>}
    </Card>
    {removed !== null && <div role="status"><Alert variant="success">{t('cleanup.success', { count: removed })}</Alert></div>}
    {statsError && errorAlert(statsError)}
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-busy={statsLoading}>
      <MetricCard label={t('stats.totalRequests')} value={stats ? number(stats.total_requests) : '—'} icon={<Activity className="h-4 w-4" />} tone="primary" />
      <MetricCard label={t('stats.errorRate')} value={stats ? `${errorRate.toLocaleString(i18n.resolvedLanguage, { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%` : '—'} icon={<AlertTriangle className="h-4 w-4" />} tone="warning" />
      <MetricCard label={t('stats.avgLatency')} value={stats ? `${number(Math.round(stats.avg_latency_ms || 0))} ms` : '—'} icon={<Clock className="h-4 w-4" />} tone="neutral" />
      <MetricCard label={t('stats.authFailures')} value={stats ? number(stats.auth_failures ?? 0) : '—'} icon={<UserX className="h-4 w-4" />} tone="error" />
    </div>
    <Card padding="lg">{statsLoading ? loadingIndicator : <React.Suspense fallback={loadingIndicator}><AccessLogCharts stats={stats} /></React.Suspense>}</Card>
    <Card padding="lg">
      <CardHeader><div className="flex flex-wrap items-center justify-between gap-4">
        <CardTitle>{t('page.title')}</CardTitle>
        <div className="flex gap-2">
          <Button variant={view === 'detail' ? 'primary' : 'ghost'} aria-pressed={view === 'detail'} size="sm" icon={<List className="h-4 w-4" />} onClick={() => setState((current) => ({ ...current, view: 'detail' }))}>{t('view.detail')}</Button>
          <Button variant={view === 'aggregate' ? 'primary' : 'ghost'} aria-pressed={view === 'aggregate'} size="sm" icon={<BarChart3 className="h-4 w-4" />} onClick={() => setState((current) => ({ ...current, view: 'aggregate' }))}>{t('view.aggregate')}</Button>
        </div>
      </div></CardHeader>
      <CardContent>
        {view === 'aggregate' && <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <Select label={t('aggregate.groupBy')} value={groupBy} onChange={(event) => setState((current) => ({ ...current, groupBy: event.target.value as AccessLogAggregateParams['group_by'], aggregatePage: 1 }))} options={[
            { value: 'host', label: t('aggregate.groupByHost') }, { value: 'route_id', label: t('aggregate.groupByRoute') },
            { value: 'client_ip', label: t('aggregate.groupByIP') }, { value: 'username', label: t('aggregate.groupByUsername') },
            { value: 'status_code', label: t('aggregate.groupByStatus') }, { value: 'auth_result', label: t('aggregate.groupByAuth') },
          ]} />
          <Select label={t('aggregate.sortBy')} value={sortBy} onChange={(event) => setState((current) => ({ ...current, sortBy: event.target.value as NonNullable<AccessLogAggregateParams['sort_by']>, aggregatePage: 1 }))} options={[
            { value: 'count', label: t('aggregate.sortByCount') }, { value: 'errors', label: t('aggregate.sortByErrors') },
            { value: 'avg_latency', label: t('aggregate.sortByAvgLatency') }, { value: 'p95_latency', label: t('aggregate.sortByP95Latency') },
          ]} />
        </div>}
        {error ? errorAlert(error) : loading ? loadingIndicator : view === 'detail' ? <>
          <p role="status" className="mb-3 text-sm text-[var(--text-muted)]">{t('page.results', { count: list?.total ?? 0 })}</p>
          {!list?.entries.length ? <EmptyState icon={<ScrollText className="h-6 w-6" />} title={t('table.noData')} /> : <AccessLogTable entries={list.entries} onSelect={setSelected} />}
          {list && list.total_pages > 1 && <Pagination page={list.page} totalPages={list.total_pages} onPageChange={(page) => setState((current) => ({ ...current, filters: { ...current.filters, page } }))} />}
        </> : <>
          <p role="status" className="mb-3 text-sm text-[var(--text-muted)]">{t('aggregate.totalGroups', { count: aggregate?.total_groups ?? 0 })}</p>
          {!aggregate?.groups.length ? <EmptyState icon={<BarChart3 className="h-6 w-6" />} title={t('aggregate.noData')} /> : <AccessLogAggregateTable groups={aggregate.groups} groupBy={groupBy} onSelect={drillDown} />}
          {aggregate && aggregate.total_pages > 1 && <Pagination page={aggregate.page} totalPages={aggregate.total_pages} onPageChange={(page) => setState((current) => ({ ...current, aggregatePage: page }))} />}
        </>}
      </CardContent>
    </Card>
    <p className="text-xs text-[var(--text-muted)]">{t('page.retentionHint')}</p>
    <AccessLogDetail entry={selected} onClose={() => setSelected(null)} />
    {isAdmin && cleanupFilters && <AccessLogCleanup filters={cleanupFilters} onClose={() => setCleanupFilters(null)} onPurged={(count) => {
      setCleanupFilters(null)
      setSelected(null)
      setRemoved(count)
      setState((current) => ({ ...current, filters: { ...current.filters, page: 1 }, aggregatePage: 1 }))
      setRefresh((n) => n + 1)
    }} />}
  </div>
}
