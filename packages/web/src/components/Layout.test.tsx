import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Layout } from './Layout'
import { renderWithI18n } from '../test/render'

describe('Layout language switching', () => {
  it('switches navigation labels when the language toggle changes', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <Layout
        currentPath="/"
        user={{
          username: 'admin',
          role: 'admin',
          permissions: {
            can_manage_routes: true,
            can_manage_auth: true,
            can_manage_users: true,
            can_view_logs: true,
          },
        }}
        onLogout={vi.fn()}
      >
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    expect(screen.getAllByText('Routes')[0]).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Language switcher' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(screen.getAllByText('路由')[0]).toBeInTheDocument()
    expect(screen.getByText('管理员')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: '语言切换' })).toBeInTheDocument()
    expect(localStorage.getItem('auth-gate.locale')).toBe('zh-CN')
  })
})
