import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HostProfileForm } from './index'
import { renderWithI18n } from '../../../test/render'

describe('HostProfileForm', () => {
  it('submits a trimmed name and description', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(<HostProfileForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('Profile name'), '  Dev  ')
    await user.type(screen.getByLabelText('Description'), '  dev hosts  ')
    await user.click(screen.getByRole('button', { name: 'Save Profile' }))

    expect(onSubmit).toHaveBeenCalledWith({ name: 'Dev', description: 'dev hosts' })
  })

  it('rejects an invalid profile name', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(<HostProfileForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('Profile name'), 'bad/name')
    await user.click(screen.getByRole('button', { name: 'Save Profile' }))

    expect(await screen.findByText(/Profile name is invalid/)).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('re-enables the submit button after an error', async () => {
    const onSubmit = vi.fn().mockRejectedValue(new Error('network'))
    const user = userEvent.setup()

    await renderWithI18n(<HostProfileForm onSubmit={onSubmit} onCancel={vi.fn()} />, { locale: 'en' })

    await user.type(screen.getByLabelText('Profile name'), 'dev')
    await user.click(screen.getByRole('button', { name: 'Save Profile' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Save Profile' })).not.toBeDisabled()
    })
  })
})
