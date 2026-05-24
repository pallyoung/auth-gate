import { afterEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { SettingsPage } from './SettingsPage'
import { renderWithI18n } from '../test/render'
import { setSessionUser } from '../lib/session-store'

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
})
