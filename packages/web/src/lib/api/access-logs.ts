import { request } from './client'
import type { AccessLogEntry, AccessLogListResponse, AccessLogStats, AccessLogQueryParams } from './types'

export const accessLogsApi = {
  list: (params: AccessLogQueryParams): Promise<AccessLogListResponse> => {
    const searchParams = new URLSearchParams()
    if (params.client_ip) searchParams.append('client_ip', params.client_ip)
    if (params.path) searchParams.append('path', params.path)
    if (params.username) searchParams.append('username', params.username)
    if (params.auth_result) searchParams.append('auth_result', params.auth_result)
    if (params.route_id) searchParams.append('route_id', params.route_id)
    if (params.status_code) searchParams.append('status_code', params.status_code.toString())
    if (params.start_time) searchParams.append('start_time', params.start_time)
    if (params.end_time) searchParams.append('end_time', params.end_time)
    if (params.page) searchParams.append('page', params.page.toString())
    if (params.per_page) searchParams.append('per_page', params.per_page.toString())
    return request(`/access-logs?${searchParams.toString()}`)
  },
  stats: (duration: string = '1h'): Promise<AccessLogStats> => {
    return request(`/access-logs/stats?duration=${duration}`)
  },
}
