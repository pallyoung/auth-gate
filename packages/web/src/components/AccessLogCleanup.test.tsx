import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AccessLogCleanup } from './AccessLogCleanup'
import { accessLogsApi } from '../lib/api/access-logs'
import { renderWithI18n } from '../test/render'
import { selectComboboxOption } from '../test/selectComboboxOption'
import type { AccessLogPurgeResult } from '../lib/api/types'

vi.mock('../lib/api/access-logs', () => ({ accessLogsApi: { purge: vi.fn() } }))

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { resolve, promise }
}

beforeEach(() => vi.mocked(accessLogsApi.purge).mockReset())

describe('AccessLogCleanup', () => {
  it('previews all matching pages and deletes only after confirmation using the frozen cutoff', async () => {
    const deletion = deferred<AccessLogPurgeResult>()
    const cutoff = '2026-09-09T12:00:00Z'
    vi.mocked(accessLogsApi.purge).mockResolvedValueOnce({ matched: 4, cutoff_time: cutoff }).mockReturnValueOnce(deletion.promise)
    const onPurged = vi.fn()
    await renderWithI18n(<AccessLogCleanup filters={{ host: 'api.example.com', page: 3, per_page: 20 }} onClose={vi.fn()} onPurged={onPurged} />, { locale: 'en' })
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus()
    const confirm = await screen.findByRole('button', { name: 'Delete 4 logs' })
    expect(accessLogsApi.purge).toHaveBeenCalledTimes(1)
    expect(screen.getByText('api.example.com')).toBeInTheDocument()
    fireEvent.click(confirm)
    fireEvent.click(confirm)
    expect(accessLogsApi.purge).toHaveBeenCalledTimes(2)
    expect(accessLogsApi.purge).toHaveBeenLastCalledWith({ mode: 'filtered', filters: { host: 'api.example.com' }, preview: false, cutoff_time: cutoff })
    expect(confirm).toBeDisabled()
    await act(async () => deletion.resolve({ removed: 4, cutoff_time: cutoff }))
    expect(onPurged).toHaveBeenCalledWith(4)
  })

  it('cannot reuse an older preview after changing the cleanup scope', async () => {
    const old = deferred<AccessLogPurgeResult>()
    vi.mocked(accessLogsApi.purge).mockReturnValueOnce(old.promise).mockResolvedValueOnce({ matched: 8, cutoff_time: '2026-09-09T12:00:00Z' })
    const user = userEvent.setup()
    await renderWithI18n(<AccessLogCleanup filters={{ host: 'api.example.com' }} onClose={vi.fn()} onPurged={vi.fn()} />, { locale: 'en' })
    await selectComboboxOption(user, 'Cleanup scope', 'All retained logs')
    await screen.findByRole('button', { name: 'Delete 8 logs' })
    await act(async () => old.resolve({ matched: 2, cutoff_time: '2026-09-09T10:00:00Z' }))
    expect(screen.getByRole('button', { name: 'Delete 8 logs' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Delete 2 logs' })).not.toBeInTheDocument()
  })

  it('requires explicit all-log scope when there are no filters', async () => {
    await renderWithI18n(<AccessLogCleanup filters={{ page: 1 }} onClose={vi.fn()} onPurged={vi.fn()} />, { locale: 'en' })
    expect(accessLogsApi.purge).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Delete 0 logs' })).toBeDisabled()
    expect(screen.getByText(/Apply a filter first/)).toBeInTheDocument()
  })

  it('keeps the dialog open on deletion failure and translates its error on language changes', async () => {
    vi.mocked(accessLogsApi.purge).mockResolvedValueOnce({ matched: 1, cutoff_time: '2026-09-09T12:00:00Z' }).mockRejectedValueOnce(new Error('disk failed'))
    const onPurged = vi.fn()
    const view = await renderWithI18n(<AccessLogCleanup filters={{ host: 'api.example.com' }} onClose={vi.fn()} onPurged={onPurged} />, { locale: 'en' })
    fireEvent.click(await screen.findByRole('button', { name: 'Delete 1 logs' }))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('Could not clear logs'))
    expect(onPurged).not.toHaveBeenCalled()
    await act(async () => view.i18n.changeLanguage('zh-CN'))
    expect(screen.getByRole('alert')).toHaveTextContent('清理日志失败')
    expect(screen.getByRole('button', { name: '删除 1 条日志' })).toBeEnabled()
  })
})
