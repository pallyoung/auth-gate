import React from 'react'
import { useTranslation } from 'react-i18next'
import { ScrollText, Activity, AlertTriangle, Clock, UserX, BarChart3, List } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { MetricCard } from '../components/ui/MetricCard'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Select } from '../components/ui/Select'
import { Button } from '../components/ui/Button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table'
import { AccessLogFilters } from '../components/AccessLogFilters'
import { AccessLogDetail } from '../components/AccessLogDetail'
import { Pagination } from '../components/ui/Pagination'
import { accessLogsApi } from '../lib/api'
import type { AccessLogEntry, AccessLogListResponse, AccessLogStats, AccessLogQueryParams, AccessLogAggregateResponse, AccessLogAggregateGroup } from '../lib/api/types'

const AccessLogCharts = React.lazy(() =>
  import('../components/AccessLogCharts').then((m) => ({ default: m.AccessLogCharts }))
)

export function AccessLogsPage() {
  const { t } = useTranslation('accessLogs')

  // View mode: 'detail' or 'aggregate'
  const [viewMode, setViewMode] = React.useState<'detail' | 'aggregate'>('detail')

  // Detail view state
  const [entries, setEntries] = React.useState<AccessLogEntry[]>([])
  const [stats, setStats] = React.useState<AccessLogStats | null>(null)
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [selectedEntry, setSelectedEntry] = React.useState<AccessLogEntry | null>(null)
  const [filters, setFilters] = React.useState<AccessLogQueryParams>({
    page: 1,
    per_page: 20,
  })
  const [total, setTotal] = React.useState(0)
  const [totalPages, setTotalPages] = React.useState(0)

  // Aggregate view state
  const [aggData, setAggData] = React.useState<AccessLogAggregateResponse | null>(null)
  const [aggLoading, setAggLoading] = React.useState(false)
  const [groupBy, setGroupBy] = React.useState<string>('client_ip')
  const [sortBy, setSortBy] = React.useState<string>('count')
  const [duration, setDuration] = React.useState<string>('24h')
  const [aggPage, setAggPage] = React.useState(1)

  const fetchDetailData = React.useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const [listResult, statsResult] = await Promise.all([
        accessLogsApi.list(filters),
        accessLogsApi.stats('24h'),
      ])
      setEntries(listResult.entries)
      setTotal(listResult.total)
      setTotalPages(listResult.total_pages)
      setStats(statsResult)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch access logs')
    } finally {
      setLoading(false)
    }
  }, [filters])

  const fetchAggregateData = React.useCallback(async () => {
    try {
      setAggLoading(true)
      setError(null)
      const result = await accessLogsApi.aggregate({
        group_by: groupBy as any,
        duration,
        sort_by: sortBy as any,
        page: aggPage,
        per_page: 20,
      })
      setAggData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch aggregate data')
    } finally {
      setAggLoading(false)
    }
  }, [groupBy, sortBy, duration, aggPage])

  React.useEffect(() => {
    if (viewMode === 'detail') {
      fetchDetailData()
    } else {
      fetchAggregateData()
    }
  }, [viewMode, fetchDetailData, fetchAggregateData])

  const handlePageChange = (page: number) => {
    setFilters({ ...filters, page })
  }

  const handleFilterChange = (newFilters: AccessLogQueryParams) => {
    setFilters({ ...newFilters, page: 1 })
  }

  // Drill-down: switch to detail view with a pre-filled filter
  const handleDrillDown = (group: AccessLogAggregateGroup) => {
    const newFilters: AccessLogQueryParams = { page: 1, per_page: 20 }
    switch (groupBy) {
      case 'route_id':
        newFilters.route_id = group.key
        break
      case 'client_ip':
        newFilters.client_ip = group.key
        break
      case 'username':
        newFilters.username = group.key
        break
      case 'status_code':
        newFilters.status_code = parseInt(group.key, 10)
        break
      case 'auth_result':
        newFilters.auth_result = group.key
        break
    }
    setFilters(newFilters)
    setViewMode('detail')
  }

  const getStatusBadge = (statusCode: number) => {
    if (statusCode >= 200 && statusCode < 300) {
      return <Badge variant="success">{statusCode}</Badge>
    }
    if (statusCode >= 300 && statusCode < 400) {
      return <Badge variant="warning">{statusCode}</Badge>
    }
    if (statusCode >= 400) {
      return <Badge variant="error">{statusCode}</Badge>
    }
    return <Badge>{statusCode}</Badge>
  }

  const getAuthBadge = (authResult: string) => {
    switch (authResult) {
      case 'pass':
        return <Badge variant="success" className="normal-case tracking-normal">{t('filters.pass')}</Badge>
      case 'fail':
        return <Badge variant="error" className="normal-case tracking-normal">{t('filters.fail')}</Badge>
      case 'none':
        return <Badge className="normal-case tracking-normal">{t('filters.none')}</Badge>
      default:
        return <Badge className="normal-case tracking-normal">{authResult}</Badge>
    }
  }

  const getMethodBadge = (method: string) => {
    const colors: Record<string, 'primary' | 'success' | 'warning' | 'error' | 'default'> = {
      GET: 'primary',
      POST: 'success',
      PUT: 'warning',
      DELETE: 'error',
    }
    return <Badge variant={colors[method] || 'default'}>{method}</Badge>
  }

  const errorRate = stats ? (stats.error_count / stats.total_requests * 100).toFixed(1) : '0'
  const authFailures = stats ? stats.error_count : 0

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow={t('page.eyebrow')}
        title={t('page.title')}
        description={t('page.description')}
        meta={<Badge variant="success">{t('page.badge')}</Badge>}
      />

      {error && (
        <Alert variant="error" className="mb-6">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label={t('stats.totalRequests')}
          value={stats?.total_requests || 0}
          icon={<Activity className="h-4 w-4" />}
          tone="primary"
        />
        <MetricCard
          label={t('stats.errorRate')}
          value={`${errorRate}%`}
          icon={<AlertTriangle className="h-4 w-4" />}
          tone="warning"
        />
        <MetricCard
          label={t('stats.avgLatency')}
          value={`${stats?.avg_latency_ms?.toFixed(0) || 0} ms`}
          icon={<Clock className="h-4 w-4" />}
          tone="neutral"
        />
        <MetricCard
          label={t('stats.authFailures')}
          value={authFailures}
          icon={<UserX className="h-4 w-4" />}
          tone="error"
        />
      </div>

      <Card className="mb-6" padding="lg">
        <React.Suspense fallback={<div className="flex h-48 items-center justify-center"><div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" /></div>}>
          <AccessLogCharts stats={stats} />
        </React.Suspense>
      </Card>

      <Card padding="lg">
        <CardHeader>
          <div className="flex items-center justify-between gap-4">
            <CardTitle>{t('page.title')}</CardTitle>
            <div className="flex items-center gap-2">
              <Button
                variant={viewMode === 'detail' ? 'primary' : 'ghost'}
                size="sm"
                icon={<List className="h-4 w-4" />}
                onClick={() => setViewMode('detail')}
              >
                {t('view.detail')}
              </Button>
              <Button
                variant={viewMode === 'aggregate' ? 'primary' : 'ghost'}
                size="sm"
                icon={<BarChart3 className="h-4 w-4" />}
                onClick={() => setViewMode('aggregate')}
              >
                {t('view.aggregate')}
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent>
          {viewMode === 'aggregate' ? (
            /* ---- Aggregate View ---- */
            <>
              <div className="mb-8 grid grid-cols-1 gap-4 md:grid-cols-3">
                <Select
                  label={t('aggregate.groupBy')}
                  value={groupBy}
                  onChange={(e) => { setGroupBy(e.target.value); setAggPage(1) }}
                  options={[
                    { value: 'route_id', label: t('aggregate.groupByRoute') },
                    { value: 'client_ip', label: t('aggregate.groupByIP') },
                    { value: 'username', label: t('aggregate.groupByUsername') },
                    { value: 'status_code', label: t('aggregate.groupByStatus') },
                    { value: 'auth_result', label: t('aggregate.groupByAuth') },
                  ]}
                />
                <Select
                  label={t('aggregate.sortBy')}
                  value={sortBy}
                  onChange={(e) => { setSortBy(e.target.value); setAggPage(1) }}
                  options={[
                    { value: 'count', label: t('aggregate.sortByCount') },
                    { value: 'errors', label: t('aggregate.sortByErrors') },
                    { value: 'avg_latency', label: t('aggregate.sortByAvgLatency') },
                    { value: 'p95_latency', label: t('aggregate.sortByP95Latency') },
                  ]}
                />
                <Select
                  label={t('aggregate.duration')}
                  value={duration}
                  onChange={(e) => { setDuration(e.target.value); setAggPage(1) }}
                  options={[
                    { value: '1h', label: t('aggregate.duration1h') },
                    { value: '6h', label: t('aggregate.duration6h') },
                    { value: '24h', label: t('aggregate.duration24h') },
                    { value: '168h', label: t('aggregate.duration7d') },
                  ]}
                />
              </div>

              {aggLoading ? (
                <div className="flex h-32 items-center justify-center">
                  <div className="h-8 w-8 animate-spin rounded-full border-4 border-[var(--primary-500)] border-t-transparent" />
                </div>
              ) : !aggData || aggData.groups.length === 0 ? (
                <EmptyState
                  icon={<BarChart3 className="h-6 w-6" />}
                  title={t('aggregate.noData')}
                />
              ) : (
                <>
                  <div className="mb-3 text-sm text-[var(--text-muted)]">
                    {t('aggregate.totalGroups', { count: aggData.total_groups })}
                  </div>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('aggregate.colKey')}</TableHead>
                        <TableHead className="w-24 text-right">{t('aggregate.colCount')}</TableHead>
                        <TableHead className="w-28 text-right">{t('aggregate.colErrors')}</TableHead>
                        <TableHead className="w-24 text-right">{t('aggregate.colAuthFails')}</TableHead>
                        <TableHead className="w-28 text-right">{t('aggregate.colAvgLatency')}</TableHead>
                        <TableHead className="w-28 text-right">{t('aggregate.colP95Latency')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {aggData.groups.map((group) => (
                        <TableRow
                          key={group.key}
                          className="cursor-pointer hover:bg-[var(--bg-elevated)]"
                          onClick={() => handleDrillDown(group)}
                        >
                          <TableCell>
                            <div className="flex flex-col">
                              <span className="font-medium text-[var(--text-primary)]">
                                {group.route_name || group.key || '(empty)'}
                              </span>
                              {group.route_name && group.key !== group.route_name && (
                                <span className="font-mono text-xs text-[var(--text-muted)]">{group.key}</span>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-right font-mono">{group.count.toLocaleString()}</TableCell>
                          <TableCell className="text-right">
                            <div className="flex items-center justify-end gap-2">
                              <span className="font-mono">{group.errors}</span>
                              {group.error_rate > 0 && (
                                <Badge variant={group.error_rate > 10 ? 'error' : group.error_rate > 5 ? 'warning' : 'default'} badgeSize="sm">
                                  {group.error_rate.toFixed(1)}%
                                </Badge>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="text-right font-mono">
                            {group.auth_failures > 0 ? (
                              <Badge variant="error" badgeSize="sm">{group.auth_failures}</Badge>
                            ) : (
                              <span className="text-[var(--text-muted)]">0</span>
                            )}
                          </TableCell>
                          <TableCell className="text-right font-mono">{group.avg_latency_ms.toFixed(0)} ms</TableCell>
                          <TableCell className="text-right font-mono">{group.p95_latency_ms} ms</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>

                  {aggData.total_pages > 1 && (
                    <Pagination
                      page={aggData.page}
                      totalPages={aggData.total_pages}
                      onPageChange={(p) => setAggPage(p)}
                    />
                  )}
                </>
              )}
            </>
          ) : (
            /* ---- Detail View ---- */
            <>
              <AccessLogFilters filters={filters} onChange={handleFilterChange} />

              <div className="mt-6">
                {loading ? (
                  <div className="flex h-32 items-center justify-center">
                    <div className="h-8 w-8 animate-spin rounded-full border-4 border-[var(--primary-500)] border-t-transparent" />
                  </div>
                ) : entries.length === 0 ? (
                  <EmptyState
                    icon={<ScrollText className="h-6 w-6" />}
                    title={t('table.noData')}
                  />
                ) : (
                  <>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-32">{t('table.timestamp')}</TableHead>
                          <TableHead className="w-20">{t('table.method')}</TableHead>
                          <TableHead className="w-40">{t('table.route')}</TableHead>
                          <TableHead>{t('table.path')}</TableHead>
                          <TableHead className="w-20">{t('table.statusCode')}</TableHead>
                          <TableHead className="w-32">{t('table.clientIP')}</TableHead>
                          <TableHead className="w-24">{t('table.username')}</TableHead>
                          <TableHead className="w-28">{t('table.authResult')}</TableHead>
                          <TableHead className="w-24">{t('table.latency')}</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {entries.map((entry) => (
                          <TableRow
                            key={entry.request_id}
                            className="cursor-pointer hover:bg-[var(--bg-elevated)]"
                            onClick={() => setSelectedEntry(entry)}
                          >
                            <TableCell className="text-xs text-[var(--text-muted)]">
                              {new Date(entry.timestamp).toLocaleTimeString('zh-CN')}
                            </TableCell>
                            <TableCell>{getMethodBadge(entry.method)}</TableCell>
                            <TableCell className="text-sm truncate max-w-[160px]" title={entry.route_name || entry.route_id}>
                              {entry.route_name || entry.route_id}
                            </TableCell>
                            <TableCell className="font-mono text-sm">
                              {entry.path}
                            </TableCell>
                            <TableCell>{getStatusBadge(entry.status_code)}</TableCell>
                            <TableCell className="font-mono text-sm">
                              {entry.client_ip}
                            </TableCell>
                            <TableCell className="text-sm">
                              {entry.username || '-'}
                            </TableCell>
                            <TableCell>{getAuthBadge(entry.auth_result)}</TableCell>
                            <TableCell className="text-sm">
                              {entry.backend_latency_ms} ms
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>

                    {totalPages > 1 && (
                      <Pagination
                        page={filters.page || 1}
                        totalPages={totalPages}
                        onPageChange={handlePageChange}
                      />
                    )}
                  </>
                )}
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <AccessLogDetail
        entry={selectedEntry}
        onClose={() => setSelectedEntry(null)}
      />
    </div>
  )
}
