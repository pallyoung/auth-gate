import { screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { RoutesPage } from './RoutesPage'
import { renderWithI18n } from '../test/render'

vi.mock('../lib/api/routes', () => ({
  routesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => ({
    username: 'admin',
    role: 'admin',
    permissions: {
      can_manage_routes: true,
      can_manage_auth: true,
      can_manage_users: true,
      can_view_logs: true,
    },
  }),
}))

describe('RoutesPage i18n', () => {
  beforeEach(async () => {
    const { routesApi } = await import('../lib/api/routes')
    vi.mocked(routesApi.list).mockResolvedValue([])
  })

  it('renders translated page and empty state copy in zh-CN', async () => {
    await renderWithI18n(<RoutesPage />, { locale: 'zh-CN' })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '路由' })).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: '新增路由' })).toBeInTheDocument()
    expect(screen.getByText('尚未配置路由')).toBeInTheDocument()
  })
})
