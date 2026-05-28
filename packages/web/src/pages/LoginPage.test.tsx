import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { LoginPage } from './LoginPage'
import { ApiError } from '../lib/api/client'
import { renderWithI18n } from '../test/render'

describe('LoginPage', () => {
  const onLogin = vi.fn()

  beforeEach(() => {
    onLogin.mockReset()
  })

  it('renders translated login copy in zh-CN', async () => {
    await renderWithI18n(<LoginPage onLogin={onLogin} />, { locale: 'zh-CN' })

    expect(screen.getByRole('heading', { name: '登录控制台' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
    expect(screen.getByLabelText('用户名')).toBeInTheDocument()
  })

  it('exposes username and password autocomplete hints', async () => {
    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    expect(screen.getByLabelText('Username')).toHaveAttribute('autocomplete', 'username')
    expect(screen.getByLabelText('Password')).toHaveAttribute('autocomplete', 'current-password')
  })

  it('shows a helpful invalid credentials message instead of the raw API error', async () => {
    onLogin.mockRejectedValue(
      new ApiError('invalid credentials', 401, 'invalid_credentials')
    )

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'admin')
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )
  })

  it('shows a disabled account message instead of the raw API error', async () => {
    onLogin.mockRejectedValue(new ApiError('user disabled', 401, 'user_disabled'))

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'disabled-admin')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This account has been disabled. Contact your administrator or try a different account.'
    )
  })

  it('shows an access guidance message when the account lacks control plane permission', async () => {
    onLogin.mockRejectedValue(
      new ApiError('control plane access denied', 403, 'control_plane_access_denied')
    )

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'route-only-user')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "This account doesn't have access to the control plane. Contact your administrator or try a different account."
    )
  })

  it('shows a generic session recovery message instead of the raw backend error', async () => {
    onLogin.mockRejectedValue(
      new ApiError('failed to load user', 500, 'session_store_failure')
    )

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'admin')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't start your control plane session right now. Try again in a moment."
    )
  })

  it('prevents duplicate sign-in submissions while a login request is pending', async () => {
    let resolveLogin: ((value: any) => void) | undefined
    onLogin.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'admin')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const submitButton = screen.getByRole('button', { name: 'Sign In' })
    await user.click(submitButton)
    await user.click(submitButton)

    expect(onLogin).toHaveBeenCalledTimes(1)

    resolveLogin?.({
      token: 'token',
      user: {
        id: 'admin-1',
        username: 'admin',
        role: 'admin',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })
  })

  it('prevents back-to-back native form submissions before the loading state re-renders', async () => {
    let resolveLogin: ((value: any) => void) | undefined
    onLogin.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveLogin = resolve
        })
    )

    const user = userEvent.setup()
    const { container } = await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), 'admin')
    await user.type(screen.getByLabelText('Password'), 'password123')

    const form = container.querySelector('form')
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(onLogin).toHaveBeenCalledTimes(1)

    resolveLogin?.({
      token: 'token',
      user: {
        id: 'admin-1',
        username: 'admin',
        role: 'admin',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })
  })

  it('submits a trimmed username when the input has surrounding whitespace', async () => {
    onLogin.mockResolvedValue({
      token: 'token',
      user: {
        id: 'admin-1',
        username: 'admin',
        role: 'admin',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />)

    await user.type(screen.getByLabelText('Username'), '  admin  ')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(onLogin).toHaveBeenCalledWith('admin', 'password123')
  })

  it('retranslates the current login error message when the language changes', async () => {
    onLogin.mockRejectedValue(
      new ApiError('invalid credentials', 401, 'invalid_credentials')
    )

    const user = userEvent.setup()

    await renderWithI18n(<LoginPage onLogin={onLogin} />, { locale: 'en' })

    await user.type(screen.getByLabelText('Username'), 'admin')
    await user.type(screen.getByLabelText('Password'), 'wrong-password')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '无法使用该用户名和密码登录。请检查凭据后重试。'
    )
    expect(screen.queryByText(
      "We couldn't sign you in with that username and password. Check your credentials and try again."
    )).not.toBeInTheDocument()
  })
})
