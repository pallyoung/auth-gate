import { act, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AccessLogFilters } from './AccessLogFilters'
import { renderWithI18n } from '../test/render'

describe('AccessLogFilters', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('updates visible text immediately and commits remote filters after 300ms', async () => {
    vi.useFakeTimers()
    const onChange = vi.fn()
    await renderWithI18n(
      <AccessLogFilters filters={{ page: 1, per_page: 20 }} onChange={onChange} />,
      { locale: 'en' }
    )
    const usernameInput = screen.getByPlaceholderText('admin')

    fireEvent.change(usernameInput, { target: { value: 'alice' } })

    expect(usernameInput).toHaveValue('alice')
    expect(onChange).not.toHaveBeenCalled()

    act(() => vi.advanceTimersByTime(299))
    expect(onChange).not.toHaveBeenCalled()

    act(() => vi.advanceTimersByTime(1))
    expect(onChange).toHaveBeenCalledWith({ page: 1, per_page: 20, username: 'alice' })
  })

  it('does not commit a filter while an IME composition is active', async () => {
    vi.useFakeTimers()
    const onChange = vi.fn()
    await renderWithI18n(
      <AccessLogFilters filters={{ page: 1, per_page: 20 }} onChange={onChange} />,
      { locale: 'en' }
    )
    const usernameInput = screen.getByPlaceholderText('admin')

    fireEvent.compositionStart(usernameInput)
    fireEvent.change(usernameInput, { target: { value: '张' } })
    act(() => vi.advanceTimersByTime(600))

    expect(onChange).not.toHaveBeenCalled()

    fireEvent.compositionEnd(usernameInput)
    act(() => vi.advanceTimersByTime(300))

    expect(onChange).toHaveBeenCalledWith({ page: 1, per_page: 20, username: '张' })
  })

  it('clears a text filter immediately, cancels pending work, and restores input focus', async () => {
    vi.useFakeTimers()
    const onChange = vi.fn()
    await renderWithI18n(
      <AccessLogFilters
        filters={{ page: 1, per_page: 20, username: 'alice' }}
        onChange={onChange}
      />,
      { locale: 'en' }
    )
    const usernameInput = screen.getByPlaceholderText('admin')

    fireEvent.change(usernameInput, { target: { value: 'alice-2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Clear Username' }))

    expect(usernameInput).toHaveValue('')
    expect(usernameInput).toHaveFocus()
    expect(onChange).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith({ page: 1, per_page: 20 })

    act(() => vi.advanceTimersByTime(300))
    expect(onChange).toHaveBeenCalledTimes(1)
  })
})
