import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useAccessLogView } from './access-log-view'

afterEach(() => window.history.replaceState(null, '', '/'))

describe('log view URL state', () => {
  it('restores the same filters, times, grouping and pagination from a link', () => {
    window.history.replaceState(null, '', '/#/logs?host=api.example.test&range=custom&start_time=2026-09-01T00%3A00%3A00Z&end_time=2026-09-09T12%3A00%3A00Z&view=aggregate&group_by=client_ip&sort_by=errors&aggregate_page=2')
    const { result } = renderHook(() => useAccessLogView())
    expect(result.current[0]).toMatchObject({
      view: 'aggregate', groupBy: 'client_ip', sortBy: 'errors', aggregatePage: 2, preset: 'custom',
      filters: { host: 'api.example.test', start_time: '2026-09-01T00:00:00.000Z', end_time: '2026-09-09T12:00:00.000Z' },
    })
    act(() => result.current[1]((current) => ({ ...current, view: 'detail', filters: { ...current.filters, page: 3 } })))
    const query = new URLSearchParams(window.location.hash.split('?')[1])
    expect(query.get('page')).toBe('3')
    expect(query.get('host')).toBe('api.example.test')
    expect(query.get('start_time')).toBe('2026-09-01T00:00:00.000Z')
  })

  it('restores a changed history entry without retaining the previous domain', () => {
    window.history.replaceState(null, '', '/#/logs?host=old.example.test&range=all')
    const { result } = renderHook(() => useAccessLogView())
    act(() => {
      window.history.replaceState(null, '', '/#/logs?host=new.example.test&range=all')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    expect(result.current[0].filters.host).toBe('new.example.test')
    expect(result.current[0].filters.start_time).toBeUndefined()
  })
})
