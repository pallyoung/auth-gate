import type { AccessLogQueryParams } from './api/types'

export type LogTimePreset = '1h' | '6h' | '24h' | '7d' | 'all' | 'custom'

export function toLocalDateTime(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function localDateTimeToISO(value: string): string | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})$/.exec(value.trim())
  if (!match) return null
  const [, year, month, day, hour, minute] = match.map(Number)
  if (year < 1000 || year > 9999) return null
  const date = new Date(year, month - 1, day, hour, minute)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day || date.getHours() !== hour || date.getMinutes() !== minute) return null
  return date.toISOString()
}

export function presetTimeRange(preset: Exclude<LogTimePreset, 'custom'>, now = new Date()): Pick<AccessLogQueryParams, 'start_time' | 'end_time'> {
  if (preset === 'all') return {}
  const hours = { '1h': 1, '6h': 6, '24h': 24, '7d': 168 }[preset]
  return { start_time: new Date(now.getTime() - hours * 3600000).toISOString(), end_time: now.toISOString() }
}
