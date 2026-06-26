import React from 'react'
import { Activity, Clock, Cpu, Database, Gauge, HardDrive, MemoryStick, Route as RouteIcon, Server } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { MetricCard } from '../components/ui/MetricCard'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/Card'
import { ProgressBar } from '../components/ui/ProgressBar'
import { systemApi } from '../lib/api'
import type { SystemStats } from '../lib/api/types'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatUptime(seconds: number, t: (key: string) => string): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d > 0) parts.push(`${d}${t('format.days')}`)
  if (h > 0) parts.push(`${h}${t('format.hours')}`)
  if (m > 0 || parts.length === 0) parts.push(`${m}${t('format.minutes')}`)
  return parts.join(' ')
}

export function DashboardPage() {
  const { t } = useTranslation('dashboard')
  const [stats, setStats] = React.useState<SystemStats | null>(null)
  const [error, setError] = React.useState<string | null>(null)

  const fetchStats = React.useCallback(async () => {
    try {
      const data = await systemApi.stats()
      setStats(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch system stats')
    }
  }, [])

  React.useEffect(() => {
    fetchStats()
    const timer = setInterval(fetchStats, 10_000)
    return () => clearInterval(timer)
  }, [fetchStats])

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow={t('page.eyebrow')}
        title={t('page.title')}
        description={t('page.description')}
      />

      {error && (
        <div className="mb-6 rounded-xl border border-[var(--error)] bg-[var(--error-light)] px-4 py-3 text-sm text-[var(--error)]">
          {error}
        </div>
      )}

      {/* Top metrics row */}
      <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <MetricCard
          label={t('metrics.cpu')}
          value={stats ? `${stats.cpu_usage_percent.toFixed(1)}%` : '-'}
          hint={stats ? `${stats.cpu_cores} cores` : undefined}
          icon={<Cpu className="h-4 w-4" />}
          tone={stats && stats.cpu_usage_percent >= 85 ? 'error' : stats && stats.cpu_usage_percent >= 60 ? 'warning' : 'primary'}
        />
        <MetricCard
          label={t('metrics.memory')}
          value={stats ? `${stats.mem_usage_percent.toFixed(1)}%` : '-'}
          hint={stats ? `${formatBytes(stats.mem_used_bytes)} / ${formatBytes(stats.mem_total_bytes)}` : undefined}
          icon={<MemoryStick className="h-4 w-4" />}
          tone={stats && stats.mem_usage_percent >= 85 ? 'error' : stats && stats.mem_usage_percent >= 60 ? 'warning' : 'accent'}
        />
        <MetricCard
          label={t('metrics.disk')}
          value={stats ? `${stats.disk_usage_percent.toFixed(1)}%` : '-'}
          hint={stats ? `${formatBytes(stats.disk_used_bytes)} / ${formatBytes(stats.disk_total_bytes)}` : undefined}
          icon={<HardDrive className="h-4 w-4" />}
          tone={stats && stats.disk_usage_percent >= 85 ? 'error' : stats && stats.disk_usage_percent >= 60 ? 'warning' : 'neutral'}
        />
        <MetricCard
          label={t('metrics.uptime')}
          value={stats ? formatUptime(stats.uptime_seconds, t) : '-'}
          icon={<Clock className="h-4 w-4" />}
        />
        <MetricCard
          label={t('metrics.goroutines')}
          value={stats?.goroutines ?? '-'}
          icon={<Gauge className="h-4 w-4" />}
        />
      </div>

      {/* Resource usage bars + info cards */}
      <div className="grid gap-5 lg:grid-cols-[1.1fr_0.9fr]">
        <div className="space-y-5">
          {/* Resource bars */}
          <Card padding="lg">
            <CardHeader>
              <CardTitle>{t('resources.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              {stats ? (
                <div className="space-y-5">
                  <ProgressBar value={stats.cpu_usage_percent} label={t('resources.cpu')} />
                  <ProgressBar value={stats.mem_usage_percent} label={t('resources.memory')} />
                  <ProgressBar value={stats.disk_usage_percent} label={t('resources.disk')} />
                </div>
              ) : (
                <div className="flex h-32 items-center justify-center">
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
                </div>
              )}
            </CardContent>
          </Card>

          {/* Go Runtime */}
          <Card padding="lg">
            <CardHeader>
              <CardTitle>{t('runtime.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              {stats ? (
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                  <InfoItem label={t('runtime.goroutines')} value={String(stats.goroutines)} />
                  <InfoItem label={t('runtime.heapAlloc')} value={formatBytes(stats.heap_alloc_bytes)} />
                  <InfoItem label={t('runtime.heapInuse')} value={formatBytes(stats.heap_inuse_bytes)} />
                  <InfoItem label={t('runtime.gcCycles')} value={String(stats.gc_count)} />
                  <InfoItem label={t('runtime.gcPause')} value={`${stats.gc_pause_total_ms.toFixed(1)} ms`} />
                </div>
              ) : null}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-5">
          {/* System info */}
          <Card padding="lg">
            <CardHeader>
              <CardTitle>{t('system.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              {stats ? (
                <div className="space-y-3">
                  <InfoRow icon={<Server className="h-4 w-4" />} label={t('system.hostname')} value={stats.hostname} />
                  <InfoRow icon={<Server className="h-4 w-4" />} label={t('system.os')} value={`${stats.platform || stats.os} ${stats.arch}`} />
                  <InfoRow icon={<Server className="h-4 w-4" />} label={t('system.kernel')} value={stats.kernel_version} />
                  <InfoRow icon={<Cpu className="h-4 w-4" />} label={t('system.goVersion')} value={stats.go_version} />
                  <InfoRow icon={<RouteIcon className="h-4 w-4" />} label={t('system.activeRoutes')} value={`${stats.active_routes} / ${stats.total_routes}`} />
                </div>
              ) : null}
            </CardContent>
          </Card>

          {/* Routes summary */}
          <Card padding="lg">
            <CardHeader>
              <CardTitle>{t('metrics.requests')}</CardTitle>
            </CardHeader>
            <CardContent>
              {stats ? (
                <div className="flex items-center gap-6">
                  <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-[var(--primary-500)] text-white">
                    <RouteIcon className="h-8 w-8" />
                  </div>
                  <div>
                    <div className="text-3xl font-bold tabular-nums text-[var(--text-primary)]">
                      {stats.active_routes}
                    </div>
                    <div className="text-sm text-[var(--text-muted)]">
                      {t('system.activeRoutes')} / {stats.total_routes} {t('system.totalRoutes')}
                    </div>
                  </div>
                </div>
              ) : null}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Auto-refresh hint */}
      <p className="mt-5 text-center text-xs text-[var(--text-muted)]">
        {t('page.autoRefresh')}
      </p>
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-[var(--text-muted)]">{label}</div>
      <div className="mt-0.5 font-mono text-sm text-[var(--text-primary)]">{value}</div>
    </div>
  )
}

function InfoRow({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-xl bg-[rgba(255,255,255,0.4)] px-3 py-2.5">
      <div className="flex items-center gap-2.5 text-sm text-[var(--text-muted)]">
        {icon}
        {label}
      </div>
      <span className="font-mono text-sm text-[var(--text-primary)]">{value}</span>
    </div>
  )
}
