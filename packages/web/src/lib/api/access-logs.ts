import { request } from './client'
import type { AccessLogListResponse, AccessLogStats, AccessLogQueryParams, AccessLogAggregateParams, AccessLogAggregateResponse, AccessLogPurgeRequest, AccessLogPurgeResult } from './types'

export function accessLogSearchParams(params: AccessLogQueryParams): URLSearchParams {
  const search = new URLSearchParams()
  for (const key of ['host', 'client_ip', 'path', 'username', 'auth_result', 'route_id', 'status_code', 'start_time', 'end_time', 'page', 'per_page'] as const) {
    const value = params[key]
    if (value !== undefined && value !== '') search.set(key, String(value))
  }
  return search
}

export const accessLogsApi = {
  list: (params: AccessLogQueryParams): Promise<AccessLogListResponse> =>
    request(`/access-logs?${accessLogSearchParams(params)}`),
  stats: (params: AccessLogQueryParams | string = '1h'): Promise<AccessLogStats> => {
    const search = typeof params === 'string'
      ? new URLSearchParams({ duration: params })
      : accessLogSearchParams(params)
    search.delete('page')
    search.delete('per_page')
    if (typeof params !== 'string') search.set('duration', 'all')
    return request(`/access-logs/stats?${search}`)
  },
  aggregate: (params: AccessLogAggregateParams): Promise<AccessLogAggregateResponse> => {
    const search = accessLogSearchParams(params)
    search.set('group_by', params.group_by)
    search.set('duration', params.duration || 'all')
    for (const key of ['duration', 'sort_by', 'sort_order'] as const) {
      if (params[key]) search.set(key, params[key])
    }
    return request(`/access-logs/aggregate?${search}`)
  },
  purge: (body: AccessLogPurgeRequest): Promise<AccessLogPurgeResult> =>
    request('/access-logs/purge', { method: 'POST', body: JSON.stringify(body) }),
}
