import { describe, expect, it } from 'vitest'
import { localDateTimeToISO, toLocalDateTime, presetTimeRange } from './access-log-time'

describe('log time bounds', () => {
  it('converts local input to the same instant sent to the API', () => {
    const local = new Date(2026, 8, 9, 14, 30)
    expect(localDateTimeToISO('2026-09-09 14:30')).toBe(local.toISOString())
    expect(toLocalDateTime(local.toISOString())).toBe('2026-09-09 14:30')
  })

  it('rejects impossible dates instead of rolling them to another month', () => {
    expect(localDateTimeToISO('2026-02-30 12:00')).toBeNull()
    expect(localDateTimeToISO('2026-09-09 25:00')).toBeNull()
    expect(localDateTimeToISO('09/09/2026')).toBeNull()
  })

  it('uses one fixed end instant for relative filtering across all views', () => {
    const now = new Date('2026-09-09T12:00:00Z')
    expect(presetTimeRange('24h', now)).toEqual({ start_time: '2026-09-08T12:00:00.000Z', end_time: now.toISOString() })
    expect(presetTimeRange('all', now)).toEqual({})
  })
})
