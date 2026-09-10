import React from 'react'
import { accessLogSearchParams } from './api/access-logs'
import type { AccessLogAggregateParams, AccessLogQueryParams } from './api/types'
import { presetTimeRange, type LogTimePreset } from './access-log-time'

export interface AccessLogView {
  filters: AccessLogQueryParams
  preset: LogTimePreset
  view: 'detail' | 'aggregate'
  groupBy: AccessLogAggregateParams['group_by']
  sortBy: NonNullable<AccessLogAggregateParams['sort_by']>
  aggregatePage: number
}

function readView(): AccessLogView {
  const [route, query = ''] = window.location.hash.slice(1).split('?')
  const search = new URLSearchParams(route === '/logs' ? query : '')
  const presetValue = search.get('range')
  const preset = (['1h', '6h', '24h', '7d', 'all', 'custom'].includes(presetValue || '') ? presetValue : '24h') as LogTimePreset
  const filters: AccessLogQueryParams = { ...presetTimeRange(preset === 'custom' ? '24h' : preset), page: 1, per_page: 20 }
  for (const key of ['host', 'client_ip', 'path', 'username', 'auth_result', 'route_id'] as const) {
    const value = search.get(key)
    if (value) filters[key] = value
  }
  for (const key of ['start_time', 'end_time'] as const) {
    const value = search.get(key)
    if (value && !Number.isNaN(Date.parse(value))) filters[key] = new Date(value).toISOString()
  }
  if (filters.start_time && filters.end_time && filters.start_time > filters.end_time) Object.assign(filters, presetTimeRange('24h'))
  const page = Number(search.get('page'))
  if (Number.isInteger(page) && page > 0) filters.page = page
  const status = Number(search.get('status_code'))
  if (Number.isInteger(status) && status >= 100 && status <= 599) filters.status_code = status
  const groups = ['host', 'client_ip', 'route_id', 'username', 'status_code', 'auth_result']
  const sorts = ['count', 'errors', 'avg_latency', 'p95_latency']
  const aggregatePage = Number(search.get('aggregate_page'))
  return {
    filters, preset,
    view: search.get('view') === 'aggregate' ? 'aggregate' : 'detail',
    groupBy: (groups.includes(search.get('group_by') || '') ? search.get('group_by') : 'host') as AccessLogView['groupBy'],
    sortBy: (sorts.includes(search.get('sort_by') || '') ? search.get('sort_by') : 'count') as AccessLogView['sortBy'],
    aggregatePage: Number.isInteger(aggregatePage) && aggregatePage > 0 ? aggregatePage : 1,
  }
}

export function useAccessLogView() {
  const [state, setState] = React.useState(readView)
  React.useEffect(() => {
    const restore = () => {
      if (window.location.hash.split('?')[0] === '#/logs') setState(readView())
    }
    window.addEventListener('hashchange', restore)
    window.addEventListener('popstate', restore)
    return () => { window.removeEventListener('hashchange', restore); window.removeEventListener('popstate', restore) }
  }, [])
  React.useEffect(() => {
    if (window.location.hash.split('?')[0] !== '#/logs') return
    const search = accessLogSearchParams(state.filters)
    search.set('range', state.preset)
    search.set('view', state.view)
    search.set('group_by', state.groupBy)
    search.set('sort_by', state.sortBy)
    search.set('aggregate_page', String(state.aggregatePage))
    window.history.replaceState(window.history.state, '', `${window.location.pathname}${window.location.search}#/logs?${search}`)
  }, [state])
  return [state, setState] as const
}
