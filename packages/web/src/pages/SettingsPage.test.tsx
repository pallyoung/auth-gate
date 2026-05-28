import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SettingsPage } from './SettingsPage'
import { configApi } from '../lib/api/config'
import { ApiError } from '../lib/api/client'
import { renderWithI18n } from '../test/render'
import { setSessionUser } from '../lib/session-store'

vi.mock('../lib/api/config', () => ({
  configApi: {
    reload: vi.fn(),
  },
}))

beforeEach(() => {
  vi.mocked(configApi.reload).mockReset()
  vi.mocked(configApi.reload).mockResolvedValue({ message: 'reloaded' })
})

afterEach(() => {
  setSessionUser(null)
})

describe('SettingsPage i18n', () => {
  it('translates the active role label', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    await renderWithI18n(<SettingsPage />, { locale: 'zh-CN' })

    expect(screen.getByText('当前角色：管理员')).toBeInTheDocument()
  })

  it('shows a localized success message instead of the raw backend reload response', async () => {
    setSessionUser({
      id: 'editor-1',
      username: 'editor',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
    })

    const user = userEvent.setup()

    await renderWithI18n(<SettingsPage />, { locale: 'en' })

    await user.click(screen.getByRole('button', { name: 'Reload Config' }))

    expect(await screen.findByText('The running gateway configuration has been reloaded.')).toBeInTheDocument()
    expect(screen.queryByText('reloaded')).not.toBeInTheDocument()
  })

  it('shows permission guidance instead of the raw backend error when reload is rejected', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    vi.mocked(configApi.reload).mockRejectedValue(
      new ApiError('insufficient permissions', 403, 'insufficient_permissions')
    )

    const user = userEvent.setup()

    await renderWithI18n(<SettingsPage />, { locale: 'en' })

    await user.click(screen.getByRole('button', { name: 'Reload Config' }))

    expect(
      await screen.findByText('Your account cannot reload the runtime configuration. Ask an editor or administrator to apply the change.')
    ).toBeInTheDocument()
    expect(screen.queryByText('insufficient permissions')).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw invalid token error when reload is rejected', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    vi.mocked(configApi.reload).mockRejectedValue(
      new ApiError('invalid token', 401, 'invalid_token')
    )

    const user = userEvent.setup()

    await renderWithI18n(<SettingsPage />, { locale: 'en' })

    await user.click(screen.getByRole('button', { name: 'Reload Config' }))

    expect(
      await screen.findByText('Your session has expired. Sign in again before reloading the runtime configuration.')
    ).toBeInTheDocument()
    expect(screen.queryByText('invalid token')).not.toBeInTheDocument()
  })

  it('prevents duplicate config reloads while a reload is pending', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    let resolveReload: ((value: { message: string }) => void) | undefined
    vi.mocked(configApi.reload).mockImplementation(
      () =>
        new Promise<{ message: string }>((resolve) => {
          resolveReload = resolve
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<SettingsPage />, { locale: 'en' })

    const reloadButton = screen.getByRole('button', { name: 'Reload Config' })
    await user.click(reloadButton)
    await user.click(reloadButton)

    expect(configApi.reload).toHaveBeenCalledTimes(1)

    resolveReload?.({ message: 'reloaded' })
  })

  it('prevents back-to-back native reload clicks before the reloading state re-renders', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    let resolveReload: ((value: { message: string }) => void) | undefined
    vi.mocked(configApi.reload).mockImplementation(
      () =>
        new Promise<{ message: string }>((resolve) => {
          resolveReload = resolve
        })
    )

    await renderWithI18n(<SettingsPage />, { locale: 'en' })

    const reloadButton = screen.getByRole('button', { name: 'Reload Config' })

    await act(async () => {
      reloadButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      reloadButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(configApi.reload).toHaveBeenCalledTimes(1)

    resolveReload?.({ message: 'reloaded' })
  })

  it('retranslates the current reload success message when the language changes', async () => {
    setSessionUser({
      id: 'editor-1',
      username: 'editor',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
    })

    const user = userEvent.setup()
    const view = await renderWithI18n(<SettingsPage />, { locale: 'en' })

    await user.click(screen.getByRole('button', { name: 'Reload Config' }))

    expect(await screen.findByText('The running gateway configuration has been reloaded.')).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('正在运行的网关配置已重新加载。')).toBeInTheDocument()
    expect(screen.queryByText('The running gateway configuration has been reloaded.')).not.toBeInTheDocument()
  })

  it('retranslates the current reload error message when the language changes', async () => {
    setSessionUser({
      id: 'admin-1',
      username: 'admin',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true,
      },
    })

    vi.mocked(configApi.reload).mockRejectedValue(
      new ApiError('invalid token', 401, 'invalid_token')
    )

    const user = userEvent.setup()
    const view = await renderWithI18n(<SettingsPage />, { locale: 'en' })

    await user.click(screen.getByRole('button', { name: 'Reload Config' }))

    expect(
      await screen.findByText('Your session has expired. Sign in again before reloading the runtime configuration.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('你的会话已失效。请重新登录后再重载运行时配置。')).toBeInTheDocument()
    expect(
      screen.queryByText('Your session has expired. Sign in again before reloading the runtime configuration.')
    ).not.toBeInTheDocument()
  })
})
