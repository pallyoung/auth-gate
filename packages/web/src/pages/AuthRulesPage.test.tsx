import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AuthRulesPage } from './AuthRulesPage'
import { ApiError } from '../lib/api/client'
import { authRulesApi } from '../lib/api/auth-rules'
import { routesApi } from '../lib/api/routes'
import { renderWithI18n } from '../test/render'

const sessionUser = {
  id: 'viewer-1',
  username: 'viewer-debug',
  role: 'viewer',
  permissions: {
    can_manage_routes: false,
    can_manage_auth: false,
    can_manage_users: false,
    can_view_logs: false,
  },
}

vi.mock('../lib/api/routes', () => ({
  routesApi: {
    list: vi.fn(),
  },
}))

vi.mock('../lib/api/auth-rules', () => ({
  authRulesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => sessionUser,
}))

describe('AuthRulesPage permissions', () => {
  beforeEach(async () => {
    sessionUser.id = 'viewer-1'
    sessionUser.username = 'viewer-debug'
    sessionUser.role = 'viewer'
    sessionUser.permissions.can_manage_routes = false
    sessionUser.permissions.can_manage_auth = false
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = false

    vi.mocked(routesApi.list).mockResolvedValue([])
    vi.mocked(authRulesApi.list).mockResolvedValue([])
    vi.mocked(authRulesApi.create).mockReset()
    vi.mocked(authRulesApi.update).mockReset()
    vi.mocked(authRulesApi.delete).mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows read-only empty state guidance for viewer accounts', async () => {
    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByText('No auth rules configured')).toBeInTheDocument()
    })

    expect(
      screen.getByText('Your account can review auth rules here, but only editors or administrators can add or update policies.')
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'Add a rule to require API keys, bearer tokens, basic auth, or gateway-managed login before requests are forwarded.'
      )
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Rule' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add Rule' })).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw backend error when loading auth rules fails', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(authRulesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing auth rules.')
    ).toBeInTheDocument()
    expect(screen.queryByText('unauthorized')).not.toBeInTheDocument()
  })

  it('does not fall back to the empty auth rule state when the auth rule directory fails to load', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText(
        'The current auth rule list could not be loaded. Resolve the current error before reviewing or editing auth rules.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('No auth rules configured')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add Rule' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Rule' })).not.toBeInTheDocument()
  })

  it('does not show normal directory counts or metrics when the auth rule directory is unavailable', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await screen.findByText(
      'The current auth rule list could not be loaded. Resolve the current error before reviewing or editing auth rules.'
    )

    expect(screen.queryByText('0 active rule definitions')).not.toBeInTheDocument()
    expect(screen.queryByText('Protected Routes')).not.toBeInTheDocument()
    expect(screen.queryByText('API Key Rules')).not.toBeInTheDocument()
    expect(screen.queryByText('Bearer + Basic')).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw invalid token error when loading auth rules fails', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(authRulesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing auth rules.')
    ).toBeInTheDocument()
    expect(screen.queryByText('invalid token')).not.toBeInTheDocument()
  })

  it('shows route loading guidance instead of the raw backend error when loading available routes fails', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )
    vi.mocked(authRulesApi.list).mockResolvedValue([])

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('The route list could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Available routes could not be loaded. Resolve the route loading error before creating or updating auth rules.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'Create a route before adding auth rules so the policy has a protected destination to attach to.'
      )
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Route' })).not.toBeInTheDocument()
    expect(screen.queryByText('failed to list routes')).not.toBeInTheDocument()
  })

  it('keeps available routes marked unavailable when route loading fails for other errors', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))
    vi.mocked(authRulesApi.list).mockResolvedValue([])

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing auth rules.')
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Available routes could not be loaded. Resolve the route loading error before creating or updating auth rules.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'Create a route before adding auth rules so the policy has a protected destination to attach to.'
      )
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Route' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add Rule' })).not.toBeInTheDocument()
  })

  it('shows a directory loading message instead of save-oriented copy when loading auth rules fails with a store error', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.list).mockRejectedValue(
      new ApiError('failed to list auth rules', 500, 'auth_rule_store_failure')
    )

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('The auth rule directory could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(screen.queryByText('The auth rule change could not be saved. Try again in a moment.')).not.toBeInTheDocument()
    expect(screen.queryByText('failed to list auth rules')).not.toBeInTheDocument()
  })

  it('keeps loaded auth rules visible when available routes fail to load', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )
    vi.mocked(authRulesApi.list).mockResolvedValue([
      {
        id: 'rule-1',
        route_id: 'route-1',
        type: 'bearer',
        config: {},
        created_at: '',
        updated_at: '',
      },
    ])

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(await screen.findByText('1 active rule definitions')).toBeInTheDocument()
    expect(screen.queryByText('No auth rules configured')).not.toBeInTheDocument()
  })

  it('summarizes runtime policy fields in the auth rule directory', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.list).mockResolvedValue([
      {
        id: 'rule-1',
        route_id: 'route-1',
        type: 'bearer',
        config: {},
        whitelist: ['127.0.0.1/32', '10.0.0.0/8'],
        rate_limit: 15,
        burst: 30,
        cors_allowed_origins: 'https://app.example.com,.example.com',
        cors_allowed_methods: 'GET,POST,OPTIONS',
        cors_allowed_headers: 'Authorization,Content-Type',
        cors_allow_credentials: true,
        cors_max_age: 7200,
        created_at: '',
        updated_at: '',
      },
    ])

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(await screen.findByText('1 active rule definitions')).toBeInTheDocument()
    expect(screen.getAllByText('15 rps / burst 30').length).toBeGreaterThan(0)
    expect(screen.getAllByText('2 whitelist entries').length).toBeGreaterThan(0)
    expect(screen.getAllByText('CORS https://app.example.com,.example.com').length).toBeGreaterThan(0)
  })

  it('does not offer edit actions when available routes fail to load', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )
    vi.mocked(authRulesApi.list).mockResolvedValue([
      {
        id: 'rule-1',
        route_id: 'route-1',
        type: 'bearer',
        config: {},
        created_at: '',
        updated_at: '',
      },
    ])

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(await screen.findByText('1 active rule definitions')).toBeInTheDocument()
    expect(screen.queryAllByLabelText('Edit')).toHaveLength(0)
    expect(screen.getAllByLabelText('Delete').length).toBeGreaterThan(0)
  })

  it('guides editors to create a route before adding auth rules when no routes exist', async () => {
    const user = userEvent.setup()
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    window.location.hash = '#/auth'

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByText('No auth rules configured')).toBeInTheDocument()
    })

    expect(
      screen.getByText('Create a route before adding auth rules so the policy has a protected destination to attach to.')
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create First Route' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Rule' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add Rule' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Create First Route' }))

    expect(window.location.hash).toBe('#/')
  })

  it('shows a duplicate rule guidance message instead of the raw API error', async () => {
    const user = userEvent.setup()

    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.create).mockRejectedValue(
      new ApiError('route already has an auth rule', 400, 'duplicate_route_auth_rule')
    )

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Rule' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Rule' }))
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'This route already has an auth rule. Edit the existing policy instead of creating another one.'
      )
    ).toBeInTheDocument()
  })

  it('shows route refresh guidance instead of the raw API error', async () => {
    const user = userEvent.setup()

    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.create).mockRejectedValue(
      new ApiError('route not found', 400, 'route_not_found')
    )

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Rule' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Rule' }))
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'The selected route is no longer available. Refresh the route list and try again.'
      )
    ).toBeInTheDocument()
  })

  it('shows permission guidance instead of the raw backend error when auth rule changes are rejected', async () => {
    const user = userEvent.setup()

    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.create).mockRejectedValue(
      new ApiError('insufficient permissions', 403, 'insufficient_permissions')
    )

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Rule' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Rule' }))
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'Your account cannot manage auth rules. Ask an editor or administrator to apply the change.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('insufficient permissions')).not.toBeInTheDocument()
  })

  it('retranslates the current auth rule list error when the language changes', async () => {
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    vi.mocked(authRulesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    const view = await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing auth rules.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('你的会话已失效。请重新登录后再管理鉴权规则。')).toBeInTheDocument()
    expect(
      screen.queryByText('Your session has expired. Sign in again before managing auth rules.')
    ).not.toBeInTheDocument()
  })

  it('retranslates the current auth rule form error when the language changes', async () => {
    const user = userEvent.setup()

    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

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
    vi.mocked(authRulesApi.create).mockRejectedValue(
      new ApiError('route already has an auth rule', 400, 'duplicate_route_auth_rule')
    )

    const view = await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Rule' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Rule' }))
    await user.click(screen.getByRole('button', { name: 'Create Rule' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'This route already has an auth rule. Edit the existing policy instead of creating another one.'
      )
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(
      within(dialog).getByText('该路由已经配置了鉴权规则。请改为编辑现有策略，而不是再创建一条。')
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'This route already has an auth rule. Edit the existing policy instead of creating another one.'
      )
    ).not.toBeInTheDocument()
  })

  it('keeps the newest auth rule refresh results when an older delete refresh resolves later', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    sessionUser.id = 'editor-1'
    sessionUser.username = 'editor-debug'
    sessionUser.role = 'editor'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = true

    const deleteResolvers = new Map<string, () => void>()
    const routesListResolvers: Array<(value: Awaited<ReturnType<typeof routesApi.list>>) => void> = []
    const rulesListResolvers: Array<(value: Awaited<ReturnType<typeof authRulesApi.list>>) => void> = []

    vi.mocked(routesApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          routesListResolvers.push(resolve)
        })
    )
    vi.mocked(authRulesApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          rulesListResolvers.push(resolve)
        })
    )
    vi.mocked(authRulesApi.delete).mockImplementation(
      (id: string) =>
        new Promise<void>((resolve) => {
          deleteResolvers.set(id, resolve)
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<AuthRulesPage />, { locale: 'en' })

    await act(async () => {
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
      rulesListResolvers[0]?.([
        {
          id: 'rule-1',
          route_id: 'route-1',
          type: 'apikey',
          config: { header_name: 'X-Billing-Key' },
          created_at: '',
          updated_at: '',
        },
        {
          id: 'rule-2',
          route_id: 'route-2',
          type: 'bearer',
          config: {},
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    const table = await screen.findByRole('table')
    const billingRuleRow = within(table).getByText('Billing Route').closest('tr')
    const reportsRuleRow = within(table).getByText('Reports Route').closest('tr')

    expect(billingRuleRow).not.toBeNull()
    expect(reportsRuleRow).not.toBeNull()

    await user.click(within(billingRuleRow as HTMLTableRowElement).getByLabelText('Delete'))
    await user.click(within(reportsRuleRow as HTMLTableRowElement).getByLabelText('Delete'))

    await act(async () => {
      deleteResolvers.get('rule-1')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(routesListResolvers).toHaveLength(2)
      expect(rulesListResolvers).toHaveLength(2)
    })

    await act(async () => {
      deleteResolvers.get('rule-2')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(routesListResolvers).toHaveLength(3)
      expect(rulesListResolvers).toHaveLength(3)
    })

    await act(async () => {
      routesListResolvers[2]?.([
        {
          id: 'route-fresh',
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
      rulesListResolvers[2]?.([
        {
          id: 'rule-fresh',
          route_id: 'route-fresh',
          type: 'gateway',
          config: {},
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(await within(table).findByText('Fresh Route')).toBeInTheDocument()

    await act(async () => {
      routesListResolvers[1]?.([
        {
          id: 'route-stale',
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
      rulesListResolvers[1]?.([
        {
          id: 'rule-stale',
          route_id: 'route-stale',
          type: 'basic',
          config: { username: 'legacy' },
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(within(table).getByText('Fresh Route')).toBeInTheDocument()
    expect(within(table).queryByText('Stale Route')).not.toBeInTheDocument()
    expect(within(table).queryByText('Billing Route')).not.toBeInTheDocument()
    expect(within(table).queryByText('Reports Route')).not.toBeInTheDocument()
  })
})
