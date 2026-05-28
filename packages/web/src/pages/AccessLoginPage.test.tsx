import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AccessLoginPage } from './AccessLoginPage'
import { authApi } from '../lib/api/auth'
import { ApiError } from '../lib/api/client'
import { renderWithI18n } from '../test/render'

vi.mock('../lib/api/auth', () => ({
  authApi: {
    accessLogin: vi.fn(),
  },
}))

describe('AccessLoginPage', () => {
  beforeEach(() => {
    vi.mocked(authApi.accessLogin).mockReset()
    window.history.replaceState({}, '', '/_authgate/')
  })

  it('exposes username and password autocomplete hints', async () => {
    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
        })}
      />
    )

    expect(screen.getByLabelText('Username')).toHaveAttribute('autocomplete', 'username')
    expect(screen.getByLabelText('Password')).toHaveAttribute('autocomplete', 'current-password')
  })

  it('shows a route context error when the login link is missing a route id', async () => {
    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_name: 'Protected App',
          path_prefix: '/protected',
        })}
      />
    )

    expect(screen.getByRole('alert')).toHaveTextContent(
      'This access link is missing route details. Return to the protected app and try again.'
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeDisabled()
  })

  it('shows a friendly blocking error when the protected route no longer exists', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('route not found', 404, 'route_not_found')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This protected route is no longer available. Return to the app and request a fresh access link.'
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeDisabled()
  })

  it('clears the stale route unavailable state when the access link changes', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValueOnce(
      new ApiError('route not found', 404, 'route_not_found')
    )

    const user = userEvent.setup()

    const rendered = await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This protected route is no longer available. Return to the app and request a fresh access link.'
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeDisabled()

    rendered.rerender(
      <I18nextProvider i18n={rendered.i18n}>
        <AccessLoginPage
          searchParams={new URLSearchParams({
            route_id: 'route-2',
            route_name: 'Fresh Protected App',
            path_prefix: '/fresh-protected',
            next: '/fresh-protected/report',
          })}
        />
      </I18nextProvider>
    )

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('ignores stale access login failures after the access link changes', async () => {
    let rejectPendingLogin: ((reason?: unknown) => void) | undefined
    vi.mocked(authApi.accessLogin).mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          rejectPendingLogin = reject
        })
    )

    const user = userEvent.setup()

    const rendered = await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    rendered.rerender(
      <I18nextProvider i18n={rendered.i18n}>
        <AccessLoginPage
          searchParams={new URLSearchParams({
            route_id: 'route-2',
            route_name: 'Fresh Protected App',
            path_prefix: '/fresh-protected',
            next: '/fresh-protected/report',
          })}
        />
      </I18nextProvider>
    )

    await act(async () => {
      rejectPendingLogin?.(new ApiError('route not found', 404, 'route_not_found'))
      await Promise.resolve()
    })

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('shows a helpful invalid credentials message instead of the raw API error', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('invalid credentials', 401, 'invalid_credentials')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('shows an access guidance message when the account lacks route permission', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('route access denied', 401, 'route_access_denied')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "This account doesn't have access to this protected route. Contact your administrator or try a different account."
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('shows a disabled account message instead of the raw API error', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('user disabled', 401, 'user_disabled')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This account has been disabled. Contact your administrator or try a different account.'
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('shows a generic session recovery message instead of the raw backend error', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('failed to generate token', 500, 'token_generation_failed')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't start your route access session right now. Try again in a moment."
    )
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeEnabled()
  })

  it('prevents duplicate access-login submissions while a login request is pending', async () => {
    let resolveLogin: ((value: any) => void) | undefined
    vi.mocked(authApi.accessLogin).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const submitButton = screen.getByRole('button', { name: 'Continue to Route' })
    await user.click(submitButton)
    await user.click(submitButton)

    expect(authApi.accessLogin).toHaveBeenCalledTimes(1)

    resolveLogin?.({ next: '/protected/report', user: { id: 'user-1', username: 'member', role: 'member' } })
  })

  it('prevents back-to-back native access-login submissions before the loading state re-renders', async () => {
    let resolveLogin: ((value: any) => void) | undefined
    vi.mocked(authApi.accessLogin).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve
        })
    )

    const user = userEvent.setup()
    const { container } = await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const form = container.querySelector('form')
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(authApi.accessLogin).toHaveBeenCalledTimes(1)

    resolveLogin?.({
      next: '/protected/report',
      user: { id: 'user-1', username: 'member', role: 'member' },
    })
  })

  it('submits a trimmed username when the input has surrounding whitespace', async () => {
    vi.mocked(authApi.accessLogin).mockResolvedValue({
      next: '/protected/report',
      user: { id: 'user-1', username: 'member', role: 'member' },
    })

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), '  member  ')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(authApi.accessLogin).toHaveBeenCalledWith({
      route_id: 'route-1',
      username: 'member',
      password: 'password123',
      next: '/protected/report',
    })
  })

  it('updates the current route after a successful access login without triggering document-navigation errors', async () => {
    vi.mocked(authApi.accessLogin).mockResolvedValue({
      next: '/protected/report',
      user: { id: 'user-1', username: 'member', role: 'member' },
    })
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(window.location.pathname).toBe('/protected/report')
    expect(consoleErrorSpy).not.toHaveBeenCalled()
  })

  it('retranslates the current access login error message when the language changes', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('invalid credentials', 401, 'invalid_credentials')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          route_name: 'Protected App',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '该用户名和密码无法登录。请检查凭据后重试。'
    )
    expect(
      screen.queryByText(
        "We couldn't sign you in with that username and password. Check your credentials and try again."
      )
    ).not.toBeInTheDocument()
  })

  it('keeps the current access login error visible when the language changes and the route name falls back to a translation', async () => {
    vi.mocked(authApi.accessLogin).mockRejectedValue(
      new ApiError('invalid credentials', 401, 'invalid_credentials')
    )

    const user = userEvent.setup()

    await renderWithI18n(
      <AccessLoginPage
        searchParams={new URLSearchParams({
          route_id: 'route-1',
          path_prefix: '/protected',
          next: '/protected/report',
        })}
      />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Username'), 'member')
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Continue to Route' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '该用户名和密码无法登录。请检查凭据后重试。'
    )
    expect(
      screen.queryByText(
        "We couldn't sign you in with that username and password. Check your credentials and try again."
      )
    ).not.toBeInTheDocument()
  })
})
