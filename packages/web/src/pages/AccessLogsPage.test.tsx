import { act, fireEvent, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AccessLogsPage } from './AccessLogsPage'
import { accessLogsApi } from '../lib/api'
import { renderWithI18n } from '../test/render'
import type { AccessLogEntry, AccessLogListResponse, AccessLogStats } from '../lib/api/types'

vi.mock('../lib/api', () => ({
  accessLogsApi: {
    list: vi.fn(),
    stats: vi.fn(),
    aggregate: vi.fn(),
  },
}))

vi.mock('../components/AccessLogCharts', () => ({
  AccessLogCharts: () => <div>charts</div>,
}))

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve
  })
  return { promise, resolve }
}

function makeEntry(requestId: string, path: string): AccessLogEntry {
  return {
    request_id: requestId,
    route_id: 'route-1',
    route_name: 'Public API',
    method: 'GET',
    path,
    backend_url: 'http://127.0.0.1:9000',
    backend_latency_ms: 12,
    status_code: 200,
    client_ip: '127.0.0.1',
    user_agent: 'vitest',
    username: 'alice',
    auth_result: 'pass',
    timestamp: '2026-01-01T00:00:00Z',
  }
}

function makeList(entry: AccessLogEntry): AccessLogListResponse {
  return {
    entries: [entry],
    total: 1,
    page: 1,
    per_page: 20,
    total_pages: 1,
  }
}

const stats: AccessLogStats = {
  total_requests: 1,
  success_count: 1,
  error_count: 0,
  avg_latency_ms: 12,
  p95_latency_ms: 12,
  requests_per_minute: [],
  error_rate_per_hour: [],
  latency_per_hour: [],
  top_paths: [],
  top_ips: [],
}

describe('AccessLogsPage request ordering', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(accessLogsApi.list).mockReset()
    vi.mocked(accessLogsApi.stats).mockReset()
    vi.mocked(accessLogsApi.stats).mockResolvedValue(stats)
    vi.mocked(accessLogsApi.aggregate).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('does not let an older filter response replace newer results', async () => {
    const firstRequest = deferred<AccessLogListResponse>()
    const secondRequest = deferred<AccessLogListResponse>()
    vi.mocked(accessLogsApi.list)
      .mockReturnValueOnce(firstRequest.promise)
      .mockReturnValueOnce(secondRequest.promise)

    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    expect(accessLogsApi.list).toHaveBeenCalledTimes(1)

    fireEvent.change(screen.getByPlaceholderText('/api/users'), {
      target: { value: '/new' },
    })
    act(() => vi.advanceTimersByTime(300))
    await act(async () => {
      await Promise.resolve()
    })
    expect(accessLogsApi.list).toHaveBeenCalledTimes(2)

    await act(async () => {
      secondRequest.resolve(makeList(makeEntry('new-request', '/new')))
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(screen.getByText('/new')).toBeInTheDocument()

    await act(async () => {
      firstRequest.resolve(makeList(makeEntry('old-request', '/old')))
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(screen.getByText('/new')).toBeInTheDocument()
    expect(screen.queryByText('/old')).not.toBeInTheDocument()
  })
})
