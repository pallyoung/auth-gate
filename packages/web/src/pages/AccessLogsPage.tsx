import React from 'react'
import { useTranslation } from 'react-i18next'
import { ScrollText, Activity, AlertTriangle, Clock, UserX } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { MetricCard } from '../components/ui/MetricCard'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Alert } from '../components/ui/Alert'
import { EmptyState } from '../components/ui/EmptyState'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/Table'
import { AccessLogFilters } from '../components/AccessLogFilters'
import { AccessLogDetail } from '../components/AccessLogDetail'
import { Pagination } from '../components/ui/Pagination'
import { accessLogsApi } from '../lib/api'
import type { AccessLogEntry, AccessLogListResponse, AccessLogStats, AccessLogQueryParams } from '../lib/api/types'

const AccessLogCharts = React.lazy(() =>
  import('../components/AccessLogCharts').then((m) => ({ default: m.AccessLogCharts }))
)

export function AccessLogsPage() {
  const { t } = useTranslation('accessLogs')

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

  const fetchData = React.useCallback(async () => {
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

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handlePageChange = (page: number) => {
    setFilters({ ...filters, page })
  }

  const handleFilterChange = (newFilters: AccessLogQueryParams) => {
    setFilters({ ...newFilters, page: 1 })
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
        return <Badge variant="success">{t('filters.pass')}</Badge>
      case 'fail':
        return <Badge variant="error">{t('filters.fail')}</Badge>
      case 'none':
        return <Badge>{t('filters.none')}</Badge>
      default:
        return <Badge>{authResult}</Badge>
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
          tone="default"
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
          <CardTitle>{t('page.title')}</CardTitle>
          <div className="text-sm text-[var(--text-muted)]">
            {total} {t('page.title').toLowerCase()}
          </div>
        </CardHeader>

        <CardContent>
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
                      <TableHead>{t('table.path')}</TableHead>
                      <TableHead className="w-20">{t('table.statusCode')}</TableHead>
                      <TableHead className="w-32">{t('table.clientIP')}</TableHead>
                      <TableHead className="w-24">{t('table.username')}</TableHead>
                      <TableHead className="w-24">{t('table.authResult')}</TableHead>
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
        </CardContent>
      </Card>

      <AccessLogDetail
        entry={selectedEntry}
        onClose={() => setSelectedEntry(null)}
      />
    </div>
  )
}
