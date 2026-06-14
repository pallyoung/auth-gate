import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HostEntryForm } from './index'
import { renderWithI18n } from '../../../test/render'

describe('HostEntryForm', () => {
  it('submits a normalized payload', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(<HostEntryForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('IP address'), '127.0.0.1')
    await user.type(screen.getByLabelText('Hostnames (space-separated)'), '  api.local db.local  ')
    await user.type(screen.getByLabelText('Comment (optional)'), 'dev box')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    expect(onSubmit).toHaveBeenCalledWith({
      ip: '127.0.0.1',
      hostnames: ['api.local', 'db.local'],
      comment: 'dev box',
      enabled: true,
    })
  })

  it('rejects an invalid IP', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(<HostEntryForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('IP address'), 'not-an-ip')
    await user.type(screen.getByLabelText('Hostnames (space-separated)'), 'api.local')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    expect(await screen.findByText('IP address is not valid. Use a valid IPv4 or IPv6 address.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('rejects an empty hostname list', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(<HostEntryForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('IP address'), '127.0.0.1')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    expect(await screen.findByText('One or more hostnames are invalid. Use letters, numbers, dots, and dashes only.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('re-enables the submit button after an error', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new Error('network'))
    const user = userEvent.setup()

    await renderWithI18n(<HostEntryForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('IP address'), '127.0.0.1')
    await user.type(screen.getByLabelText('Hostnames (space-separated)'), 'api.local')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Save Entry' })).not.toBeDisabled()
    })
  })
})
