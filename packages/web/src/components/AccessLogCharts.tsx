import React from 'react'
import { useTranslation } from 'react-i18next'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  AreaChart, Area, BarChart, Bar,
} from 'recharts'
import type { AccessLogStats } from '../lib/api/types'

interface AccessLogChartsProps {
  stats: AccessLogStats | null
}

export const AccessLogCharts: React.FC<AccessLogChartsProps> = ({ stats }) => {
  const { t, i18n } = useTranslation('accessLogs')

  if (!stats) {
    return null
  }

  const formatTime = (time: string) => new Date(time).toLocaleString(i18n.resolvedLanguage, { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  const requestsData = (stats.requests_per_minute ?? []).map((item) => ({
    time: formatTime(item.time),
    count: item.count,
  }))

  const latencyData = (stats.latency_per_hour ?? []).map((item) => ({
    time: formatTime(item.time),
    avg: item.avg_ms,
    p95: item.p95_ms,
  }))

  const errorData = (stats.error_rate_per_hour ?? []).map((item) => ({
    time: formatTime(item.time),
    count: item.count,
  }))

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div className="rounded-lg border border-[var(--border-default)] bg-[var(--bg-card)] p-4">
        <h3 className="mb-4 text-sm font-medium text-[var(--text-secondary)]">
          {t('charts.requestsPerMinute')}
        </h3>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={requestsData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-default)" />
            <XAxis
              dataKey="time"
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <YAxis
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--bg-elevated)',
                border: '1px solid var(--border-default)',
                borderRadius: '8px',
              }}
            />
            <Line
              type="monotone"
              dataKey="count"
              name={t('charts.requests')}
              stroke="var(--primary-500)"
              strokeWidth={2}
              dot={false}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="rounded-lg border border-[var(--border-default)] bg-[var(--bg-card)] p-4">
        <h3 className="mb-4 text-sm font-medium text-[var(--text-secondary)]">
          {t('charts.responseLatency')}
        </h3>
        <ResponsiveContainer width="100%" height={200}>
          <AreaChart data={latencyData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-default)" />
            <XAxis
              dataKey="time"
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <YAxis
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--bg-elevated)',
                border: '1px solid var(--border-default)',
                borderRadius: '8px',
              }}
            />
            <Area
              type="monotone"
              dataKey="avg"
              stroke="var(--primary-500)"
              fill="var(--primary-500)"
              fillOpacity={0.2}
              name={t('charts.avgLatency')}
            />
            <Area
              type="monotone"
              dataKey="p95"
              stroke="var(--warning)"
              fill="var(--warning)"
              fillOpacity={0.2}
              name={t('charts.p95Latency')}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="rounded-lg border border-[var(--border-default)] bg-[var(--bg-card)] p-4">
        <h3 className="mb-4 text-sm font-medium text-[var(--text-secondary)]">
          {t('charts.errorRate')}
        </h3>
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={errorData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border-default)" />
            <XAxis
              dataKey="time"
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <YAxis
              tick={{ fill: 'var(--text-muted)', fontSize: 12 }}
              axisLine={{ stroke: 'var(--border-default)' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--bg-elevated)',
                border: '1px solid var(--border-default)',
                borderRadius: '8px',
              }}
            />
            <Bar
              dataKey="count"
              name={t('charts.errors')}
              fill="var(--error)"
              radius={[4, 4, 0, 0]}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
