import { fireEvent, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AccessLogTimeRange } from './AccessLogTimeRange'
import { renderWithI18n } from '../test/render'

describe('AccessLogTimeRange', () => {
  it('keeps invalid custom drafts out of the query and focuses the invalid field', async () => {
    const onApply = vi.fn()
    await renderWithI18n(<AccessLogTimeRange filters={{}} preset="custom" onApply={onApply} />, { locale: 'en' })
    fireEvent.change(screen.getByLabelText('Start time'), { target: { value: '2026-02-30 12:00' } })
    fireEvent.change(screen.getByLabelText('End time'), { target: { value: '2026-09-09 12:00' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply range' }))
    expect(onApply).not.toHaveBeenCalled()
    expect(screen.getByLabelText('Start time')).toHaveFocus()
    expect(screen.getByRole('alert')).toHaveTextContent('valid date and time')
    fireEvent.change(screen.getByLabelText('Start time'), { target: { value: '2026-09-10 12:00' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply range' }))
    expect(onApply).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('on or after')
  })
})
