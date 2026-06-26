import { request } from './client'
import type { SystemStats } from './types'

export const systemApi = {
  stats: () => request<SystemStats>('/system/stats'),
}
