import { ApiError } from './api/client'

export function accessLogErrorKey(error: unknown, fallback: 'load' | 'stats' | 'aggregate' | 'preview' | 'purge') {
  if (error instanceof ApiError) {
    if (error.status === 403) return 'errors.forbidden'
    if (error.status === 401) return 'errors.expired'
    if (error.code === 'retention_disabled') return 'errors.retentionDisabled'
    if (error.code === 'invalid_params') return 'errors.invalidParams'
    if (['invalid_start_time', 'invalid_end_time', 'invalid_time_range', 'invalid_range'].includes(error.code || '')) return 'errors.invalidRange'
  }
  return `errors.${fallback}`
}
