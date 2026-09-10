import { act, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AccessLogsPage } from './AccessLogsPage'
import { accessLogsApi } from '../lib/api'
import { accessLogsApi as cleanupApi } from '../lib/api/access-logs'
import { setSessionUser } from '../lib/session-store'
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

vi.mock('../lib/api/access-logs', () => ({ accessLogsApi: { purge: vi.fn() } }))

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
    vi.mocked(cleanupApi.purge).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it.each([
    ['Domain', 'host', 'api.example.com'],
    ['IP Address', 'client_ip', '192.0.2.1'],
    ['Request Path', 'path', '/private'],
    ['Username', 'username', 'alice'],
  ])('waits for the visible %s filter before allowing cleanup', async (label, key, value) => {
    setSessionUser({ id: 'admin', username: 'admin', role: 'admin' })
    vi.mocked(accessLogsApi.list).mockResolvedValue(makeList(makeEntry('one', '/users')))
    vi.mocked(cleanupApi.purge).mockResolvedValue({ matched: 1, cutoff_time: '2026-09-09T12:00:00Z' })
    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    const cleanup = screen.getByRole('button', { name: 'Clear logs' })
    expect(cleanup).toBeEnabled()

    fireEvent.change(screen.getByRole('textbox', { name: label }), { target: { value } })
    expect(cleanup).toBeDisabled()
    fireEvent.click(cleanup)
    expect(cleanupApi.purge).not.toHaveBeenCalled()

    await act(async () => { vi.advanceTimersByTime(300) })
    expect(cleanup).toBeEnabled()
    fireEvent.click(cleanup)
    await act(async () => { await Promise.resolve() })
    expect(cleanupApi.purge).toHaveBeenCalledWith(expect.objectContaining({
      mode: 'filtered', preview: true, filters: expect.objectContaining({ [key]: value }),
    }))
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

  it('keeps cleanup disabled through IME composition and re-enables it after a reset', async () => {
    setSessionUser({ id: 'admin', username: 'admin', role: 'admin' })
    vi.mocked(accessLogsApi.list).mockResolvedValue(makeList(makeEntry('one', '/users')))
    vi.mocked(cleanupApi.purge).mockResolvedValue({ matched: 1, cutoff_time: '2026-09-09T12:00:00Z' })
    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    const username = screen.getByRole('textbox', { name: 'Username' })
    const cleanup = screen.getByRole('button', { name: 'Clear logs' })

    fireEvent.compositionStart(username)
    fireEvent.change(username, { target: { value: '张三' } })
    await act(async () => { vi.advanceTimersByTime(600) })
    expect(cleanup).toBeDisabled()
    fireEvent.compositionEnd(username)
    await act(async () => { vi.advanceTimersByTime(300) })
    expect(cleanup).toBeEnabled()

    fireEvent.change(username, { target: { value: 'pending' } })
    expect(cleanup).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Reset filters' }))
    expect(username).toHaveValue('')
    expect(cleanup).toBeEnabled()
    fireEvent.click(cleanup)
    await act(async () => { vi.advanceTimersByTime(300) })
    expect(cleanupApi.purge).toHaveBeenCalledTimes(1)
    expect(vi.mocked(cleanupApi.purge).mock.calls[0][0].filters).not.toHaveProperty('username')
  })

  it('uses the same domain and period for summaries and grouped logs', async () => {
    vi.mocked(accessLogsApi.list).mockResolvedValue(makeList(makeEntry('one', '/users')))
    vi.mocked(accessLogsApi.aggregate).mockResolvedValue({ group_by: 'host', duration: 'all', total_groups: 0, page: 1, per_page: 20, total_pages: 1, groups: [] })
    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    fireEvent.change(screen.getByRole('textbox', { name: 'Domain' }), { target: { value: 'api.example.com' } })
    await act(async () => { vi.advanceTimersByTime(300) })
    const calls = vi.mocked(accessLogsApi.list).mock.calls
    const listParams = calls[calls.length - 1][0]
    expect(accessLogsApi.stats).toHaveBeenLastCalledWith(expect.objectContaining({ host: listParams.host, start_time: listParams.start_time, end_time: listParams.end_time }))
    fireEvent.click(screen.getByRole('button', { name: 'Aggregate' }))
    await act(async () => { await Promise.resolve() })
    expect(accessLogsApi.aggregate).toHaveBeenLastCalledWith(expect.objectContaining({ host: listParams.host, start_time: listParams.start_time, end_time: listParams.end_time }))
    expect(screen.getByRole('textbox', { name: 'Domain' })).toHaveValue('api.example.com')
  })

  it('renders zero errors without NaN and uses real authentication failures', async () => {
    vi.mocked(accessLogsApi.list).mockResolvedValue({ entries: [], total: 0, page: 1, per_page: 20, total_pages: 1 })
    vi.mocked(accessLogsApi.stats).mockResolvedValue({ ...stats, total_requests: 0, auth_failures: 0 })
    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    await act(async () => { await Promise.resolve() })
    expect(screen.queryByText('NaN%')).not.toBeInTheDocument()
    expect(screen.getByText('0.0%')).toBeInTheDocument()
  })

  it('does not count backend HTTP errors as authentication failures', async () => {
    vi.mocked(accessLogsApi.list).mockResolvedValue(makeList(makeEntry('one', '/users')))
    vi.mocked(accessLogsApi.stats).mockResolvedValue({ ...stats, total_requests: 10, error_count: 7, auth_failures: 2 })
    await renderWithI18n(<AccessLogsPage />, { locale: 'en' })
    await act(async () => { await Promise.resolve() })
    expect(within(screen.getByText('Auth Failures').parentElement!).getByText('2')).toBeInTheDocument()
    expect(screen.getByText('70.0%')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Clear logs' })).not.toBeInTheDocument()
  })

  it('translates table headers, unknown hosts and full timestamps when the language changes', async () => {
    const entry = makeEntry('locale-request', '/locale')
    vi.mocked(accessLogsApi.list).mockResolvedValue(makeList(entry))
    const view = await renderWithI18n(<AccessLogsPage />, { locale: 'zh-CN' })
    await act(async () => { await Promise.resolve() })
    expect(screen.getByRole('columnheader', { name: '域名' })).toBeInTheDocument()
    expect(screen.getByText('未记录')).toBeInTheDocument()
    expect(screen.getByText(new Date(entry.timestamp).toLocaleString('zh-CN'))).toBeInTheDocument()
    await act(async () => view.i18n.changeLanguage('en'))
    expect(screen.getByRole('columnheader', { name: 'Domain' })).toBeInTheDocument()
    expect(screen.getByText('Not recorded')).toBeInTheDocument()
    expect(screen.getByText(new Date(entry.timestamp).toLocaleString('en'))).toBeInTheDocument()
    expect(accessLogsApi.list).toHaveBeenCalledTimes(1)
  })
})
