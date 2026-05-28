import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'
import { UserForm } from './UserForm'
import { renderWithI18n } from '../test/render'
import type { Route, User } from '../lib/api/types'

const routes: Route[] = [
  {
    id: 'route-1',
    name: 'Billing Route',
    host: '',
    path_prefix: '/billing',
    backend: 'http://127.0.0.1:9000',
    strip_prefix: true,
    enabled: true,
    priority: 100,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
]

const aliceUser: User = {
  id: 'user-1',
  username: 'alice',
  role: 'member',
  enabled: true,
  route_ids: ['route-1'],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const bobUser: User = {
  id: 'user-2',
  username: 'bob',
  role: 'admin',
  enabled: false,
  route_ids: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('UserForm', () => {
  it('prevents duplicate submissions while a user save is pending', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <UserForm user={null} routes={routes} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Username'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const submitButton = screen.getByRole('button', { name: 'Create User' })
    await user.click(submitButton)
    await user.click(submitButton)

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('prevents back-to-back native submissions before the submitting state re-renders', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()
    const { container } = await renderWithI18n(
      <UserForm user={null} routes={routes} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Username'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const form = container.querySelector('form')
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('refreshes form fields when the edited user changes while the form stays mounted', async () => {
    const view = await renderWithI18n(
      <UserForm user={aliceUser} routes={routes} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    expect(screen.getByLabelText('Username')).toHaveValue('alice')
    expect(screen.getByRole('combobox', { name: 'Role' })).toHaveValue('member')
    expect(screen.getByRole('checkbox', { name: /enabled/i, hidden: true })).toBeChecked()

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <UserForm user={bobUser} routes={routes} onSubmit={vi.fn()} onCancel={vi.fn()} />
      </I18nextProvider>
    )

    expect(screen.getByLabelText('Username')).toHaveValue('bob')
    expect(screen.getByRole('combobox', { name: 'Role' })).toHaveValue('admin')
    expect(screen.getByRole('checkbox', { name: /enabled/i, hidden: true })).not.toBeChecked()
  })

  it('refreshes form fields when the same user receives updated values while the form stays mounted', async () => {
    const view = await renderWithI18n(
      <UserForm user={aliceUser} routes={routes} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <UserForm
          user={{
            ...aliceUser,
            username: 'alice-updated',
            role: 'admin',
            enabled: false,
            route_ids: [],
          }}
          routes={routes}
          onSubmit={vi.fn()}
          onCancel={vi.fn()}
        />
      </I18nextProvider>
    )

    expect(screen.getByLabelText('Username')).toHaveValue('alice-updated')
    expect(screen.getByRole('combobox', { name: 'Role' })).toHaveValue('admin')
    expect(screen.getByRole('checkbox', { name: /enabled/i, hidden: true })).not.toBeChecked()
  })
})
