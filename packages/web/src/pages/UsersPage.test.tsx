import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { UsersPage } from './UsersPage'
import { ApiError } from '../lib/api/client'
import { routesApi } from '../lib/api/routes'
import { usersApi } from '../lib/api/users'
import { renderWithI18n } from '../test/render'

const sessionUser = {
  id: 'editor-1',
  username: 'editor-debug',
  role: 'editor',
  permissions: {
    can_manage_routes: true,
    can_manage_auth: true,
    can_manage_users: false,
    can_view_logs: true,
  },
}

vi.mock('../lib/api/users', () => ({
  usersApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../lib/api/routes', () => ({
  routesApi: {
    list: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => sessionUser,
}))

describe('UsersPage permissions', () => {
  beforeEach(async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(usersApi.list).mockReset()
    vi.mocked(usersApi.create).mockReset()
    vi.mocked(usersApi.update).mockReset()
    vi.mocked(usersApi.delete).mockReset()
    vi.mocked(routesApi.list).mockReset()

    vi.mocked(usersApi.list).mockResolvedValue([])
    vi.mocked(routesApi.list).mockResolvedValue([])
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('does not request protected user management data when the account cannot manage users', async () => {
    const { usersApi } = await import('../lib/api/users')
    const { routesApi } = await import('../lib/api/routes')

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(screen.getByText('User management is unavailable for your account.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add User' })).not.toBeInTheDocument()
    expect(vi.mocked(usersApi.list)).not.toHaveBeenCalled()
    expect(vi.mocked(routesApi.list)).not.toHaveBeenCalled()
  })

  it('shows session expiry guidance instead of the raw backend error when loading users fails', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing users.')
    ).toBeInTheDocument()
    expect(screen.queryByText('unauthorized')).not.toBeInTheDocument()
  })

  it('does not fall back to the empty user state when the user directory fails to load', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText(
        'The current user list could not be loaded. Resolve the current error before reviewing or editing users.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('No users configured')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First User' })).not.toBeInTheDocument()
  })

  it('does not show normal directory counts or metrics when the user directory is unavailable', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await screen.findByText(
      'The current user list could not be loaded. Resolve the current error before reviewing or editing users.'
    )

    expect(screen.queryByText('0 managed users')).not.toBeInTheDocument()
    expect(screen.queryByText('Enabled Users')).not.toBeInTheDocument()
    expect(screen.queryByText('Route Access Users')).not.toBeInTheDocument()
    expect(screen.queryByText('Operators')).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw invalid token error when loading users fails', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing users.')
    ).toBeInTheDocument()
    expect(screen.queryByText('invalid token')).not.toBeInTheDocument()
  })

  it('shows route loading guidance instead of the raw backend error when loading route assignments fails', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([])
    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('The route list could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(screen.queryByText('failed to list routes')).not.toBeInTheDocument()
  })

  it('keeps route assignments marked unavailable in the add user form when route loading fails for other errors', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([])
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing users.')
    ).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))

    expect(
      screen.getByText(
        'Route assignments are temporarily unavailable because the route list could not be loaded. You can still create accounts now and assign routes later.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('No routes available yet. Create routes before assigning access.')
    ).not.toBeInTheDocument()
  })

  it('shows a directory loading message instead of save-oriented copy when loading users fails with a store error', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(
      new ApiError('failed to list users', 500, 'user_store_failure')
    )

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('The user directory could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(screen.queryByText('The user change could not be saved. Try again in a moment.')).not.toBeInTheDocument()
    expect(screen.queryByText('failed to list users')).not.toBeInTheDocument()
  })

  it('keeps loaded users visible when route assignments fail to load', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([
      {
        id: 'user-1',
        username: 'alice',
        role: 'viewer',
        enabled: true,
        route_ids: [],
        created_at: '',
        updated_at: '',
      },
    ])
    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect((await screen.findAllByText('alice')).length).toBeGreaterThan(0)
    expect(screen.queryByText('No users configured')).not.toBeInTheDocument()
  })

  it('shows assigned route counts for operator accounts instead of implying global route access', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([
      {
        id: 'user-1',
        username: 'operator-a',
        role: 'editor',
        enabled: true,
        route_ids: ['route-1'],
        created_at: '',
        updated_at: '',
      },
    ])
    vi.mocked(routesApi.list).mockResolvedValue([
      {
        id: 'route-1',
        name: 'Billing Route',
        host: '',
        path_prefix: '/billing',
        backend: 'http://example.com',
        strip_prefix: true,
        enabled: true,
        priority: 0,
        created_at: '',
        updated_at: '',
      },
    ])

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect((await screen.findAllByText('operator-a')).length).toBeGreaterThan(0)
    expect(screen.getAllByText('1 route').length).toBeGreaterThan(0)
    expect(screen.queryByText('All routes')).not.toBeInTheDocument()
  })

  it('counts every account with assigned route access instead of only member roles', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([
      {
        id: 'user-1',
        username: 'editor-with-route',
        role: 'editor',
        enabled: true,
        route_ids: ['route-1'],
        created_at: '',
        updated_at: '',
      },
      {
        id: 'user-2',
        username: 'member-without-route',
        role: 'member',
        enabled: true,
        route_ids: [],
        created_at: '',
        updated_at: '',
      },
      {
        id: 'user-3',
        username: 'viewer-with-route',
        role: 'viewer',
        enabled: true,
        route_ids: ['route-2'],
        created_at: '',
        updated_at: '',
      },
    ])
    vi.mocked(routesApi.list).mockResolvedValue([
      {
        id: 'route-1',
        name: 'Billing Route',
        host: '',
        path_prefix: '/billing',
        backend: 'http://example.com',
        strip_prefix: true,
        enabled: true,
        priority: 0,
        created_at: '',
        updated_at: '',
      },
      {
        id: 'route-2',
        name: 'Reports Route',
        host: '',
        path_prefix: '/reports',
        backend: 'http://example.com',
        strip_prefix: true,
        enabled: true,
        priority: 0,
        created_at: '',
        updated_at: '',
      },
    ])

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(await screen.findByText('3 managed users')).toBeInTheDocument()

    const routeAccessMetric = screen.getByText('Route Access Users').parentElement
    if (!routeAccessMetric) {
      throw new Error('route access metric container not found')
    }

    expect(within(routeAccessMetric).getByText('2')).toBeInTheDocument()
    expect(
      within(routeAccessMetric).getByText('Accounts with assigned gateway-managed route access.')
    ).toBeInTheDocument()
  })

  it('shows route availability guidance in the add user form when route loading fails', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockResolvedValue([])
    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))

    expect(
      screen.getByText(
        'Route assignments are temporarily unavailable because the route list could not be loaded. You can still create accounts now and assign routes later.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('No routes available yet. Create routes before assigning access.')
    ).not.toBeInTheDocument()
  })

  it('shows a helpful duplicate username message instead of the raw API error', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.create).mockRejectedValue(
      new ApiError('username already exists', 400, 'duplicate_user')
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))
    await user.type(screen.getByLabelText('Username'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'That username is already in use. Choose a different username and try again.'
      )
    ).toBeInTheDocument()
  })

  it('shows a helpful invalid username message instead of the raw API error', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.create).mockRejectedValue(
      new ApiError('username required', 400, 'invalid_username')
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))
    await user.type(screen.getByLabelText('Username'), '   ')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'Enter a username before saving this user.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('username required')).not.toBeInTheDocument()
  })

  it('shows route assignment guidance instead of the raw API error', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(routesApi.list).mockResolvedValue([
      {
        id: 'route-1',
        name: 'Billing Route',
        host: '',
        path_prefix: '/billing',
        backend: 'http://example.com',
        strip_prefix: true,
        enabled: true,
        priority: 0,
        created_at: '',
        updated_at: '',
      },
    ])
    vi.mocked(usersApi.create).mockRejectedValue(
      new ApiError('route not found', 400, 'route_not_found')
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))
    await user.type(screen.getByLabelText('Username'), 'member-a')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('checkbox', { name: /Billing Route/i }))
    await user.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'One of the selected routes is no longer available. Refresh the route list and try again.'
      )
    ).toBeInTheDocument()
  })

  it('shows permission guidance instead of the raw backend error when user changes are rejected', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.create).mockRejectedValue(
      new ApiError('insufficient permissions', 403, 'insufficient_permissions')
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))
    await user.type(screen.getByLabelText('Username'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'Your account cannot manage users. Ask an administrator to apply the change.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('insufficient permissions')).not.toBeInTheDocument()
  })

  it('retranslates the current user list error when the language changes', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    const view = await renderWithI18n(<UsersPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing users.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('你的会话已失效。请重新登录后再管理用户。')).toBeInTheDocument()
    expect(
      screen.queryByText('Your session has expired. Sign in again before managing users.')
    ).not.toBeInTheDocument()
  })

  it('retranslates the current user form error when the language changes', async () => {
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    vi.mocked(usersApi.create).mockRejectedValue(
      new ApiError('username already exists', 400, 'duplicate_user')
    )

    const user = userEvent.setup()
    const view = await renderWithI18n(<UsersPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add User' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add User' }))
    await user.type(screen.getByLabelText('Username'), 'alice')
    await user.type(screen.getByLabelText('Password'), 'password123')
    await user.click(screen.getByRole('button', { name: 'Create User' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'That username is already in use. Choose a different username and try again.'
      )
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(within(dialog).getByText('该用户名已被使用。请更换其他用户名后重试。')).toBeInTheDocument()
    expect(
      screen.queryByText('That username is already in use. Choose a different username and try again.')
    ).not.toBeInTheDocument()
  })

  it('keeps the newest user refresh results when an older delete refresh resolves later', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    sessionUser.id = 'admin-1'
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_users = true

    const deleteResolvers = new Map<string, () => void>()
    const usersListResolvers: Array<(value: Awaited<ReturnType<typeof usersApi.list>>) => void> = []
    const routesListResolvers: Array<(value: Awaited<ReturnType<typeof routesApi.list>>) => void> = []

    vi.mocked(usersApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          usersListResolvers.push(resolve)
        })
    )
    vi.mocked(routesApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          routesListResolvers.push(resolve)
        })
    )
    vi.mocked(usersApi.delete).mockImplementation(
      (id: string) =>
        new Promise<void>((resolve) => {
          deleteResolvers.set(id, resolve)
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<UsersPage />, { locale: 'en' })

    await act(async () => {
      usersListResolvers[0]?.([
        {
          id: 'user-1',
          username: 'alice',
          role: 'viewer',
          enabled: true,
          route_ids: ['route-1'],
          created_at: '',
          updated_at: '',
        },
        {
          id: 'user-2',
          username: 'bob',
          role: 'member',
          enabled: true,
          route_ids: ['route-2'],
          created_at: '',
          updated_at: '',
        },
      ])
      routesListResolvers[0]?.([
        {
          id: 'route-1',
          name: 'Billing Route',
          host: '',
          path_prefix: '/billing',
          backend: 'http://billing.internal',
          strip_prefix: true,
          enabled: true,
          priority: 0,
          created_at: '',
          updated_at: '',
        },
        {
          id: 'route-2',
          name: 'Reports Route',
          host: '',
          path_prefix: '/reports',
          backend: 'http://reports.internal',
          strip_prefix: true,
          enabled: true,
          priority: 0,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    const table = await screen.findByRole('table')
    const aliceRow = within(table).getByText('alice').closest('tr')
    const bobRow = within(table).getByText('bob').closest('tr')

    expect(aliceRow).not.toBeNull()
    expect(bobRow).not.toBeNull()

    await user.click(within(aliceRow as HTMLTableRowElement).getByLabelText('Delete'))
    await user.click(within(bobRow as HTMLTableRowElement).getByLabelText('Delete'))

    await act(async () => {
      deleteResolvers.get('user-1')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(usersListResolvers).toHaveLength(2)
      expect(routesListResolvers).toHaveLength(2)
    })

    await act(async () => {
      deleteResolvers.get('user-2')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(usersListResolvers).toHaveLength(3)
      expect(routesListResolvers).toHaveLength(3)
    })

    await act(async () => {
      usersListResolvers[2]?.([
        {
          id: 'user-fresh',
          username: 'fresh-user',
          role: 'editor',
          enabled: true,
          route_ids: ['route-3'],
          created_at: '',
          updated_at: '',
        },
      ])
      routesListResolvers[2]?.([
        {
          id: 'route-3',
          name: 'Fresh Route',
          host: '',
          path_prefix: '/fresh',
          backend: 'http://fresh.internal',
          strip_prefix: true,
          enabled: true,
          priority: 0,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(await within(table).findByText('fresh-user')).toBeInTheDocument()

    await act(async () => {
      usersListResolvers[1]?.([
        {
          id: 'user-stale',
          username: 'stale-user',
          role: 'viewer',
          enabled: true,
          route_ids: ['route-4'],
          created_at: '',
          updated_at: '',
        },
      ])
      routesListResolvers[1]?.([
        {
          id: 'route-4',
          name: 'Stale Route',
          host: '',
          path_prefix: '/stale',
          backend: 'http://stale.internal',
          strip_prefix: true,
          enabled: true,
          priority: 0,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(within(table).getByText('fresh-user')).toBeInTheDocument()
    expect(within(table).queryByText('stale-user')).not.toBeInTheDocument()
    expect(within(table).queryByText('alice')).not.toBeInTheDocument()
    expect(within(table).queryByText('bob')).not.toBeInTheDocument()
  })
})
