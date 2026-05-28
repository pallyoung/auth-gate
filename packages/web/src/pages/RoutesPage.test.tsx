import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { RoutesPage } from './RoutesPage'
import { ApiError } from '../lib/api/client'
import { routesApi } from '../lib/api/routes'
import { renderWithI18n } from '../test/render'

const sessionUser = {
  username: 'admin',
  role: 'admin',
  permissions: {
    can_manage_routes: true,
    can_manage_auth: true,
    can_manage_users: true,
    can_view_logs: true,
  },
}

vi.mock('../lib/api/routes', () => ({
  routesApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => sessionUser,
}))

describe('RoutesPage i18n', () => {
  beforeEach(async () => {
    sessionUser.username = 'admin'
    sessionUser.role = 'admin'
    sessionUser.permissions.can_manage_routes = true
    sessionUser.permissions.can_manage_auth = true
    sessionUser.permissions.can_manage_users = true
    sessionUser.permissions.can_view_logs = true

    vi.mocked(routesApi.list).mockResolvedValue([])
    vi.mocked(routesApi.create).mockReset()
    vi.mocked(routesApi.update).mockReset()
    vi.mocked(routesApi.delete).mockReset()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders translated page and empty state copy in zh-CN', async () => {
    await renderWithI18n(<RoutesPage />, { locale: 'zh-CN' })

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '路由' })).toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: '新增路由' })).toBeInTheDocument()
    expect(screen.getByText('尚未配置路由')).toBeInTheDocument()
  })

  it('shows read-only empty state guidance for viewer accounts', async () => {
    sessionUser.username = 'viewer-debug'
    sessionUser.role = 'viewer'
    sessionUser.permissions.can_manage_routes = false
    sessionUser.permissions.can_manage_auth = false
    sessionUser.permissions.can_manage_users = false
    sessionUser.permissions.can_view_logs = false

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByText('No routes configured')).toBeInTheDocument()
    })

    expect(
      screen.getByText('Your account can review routes here, but only editors or administrators can add forwarding rules.')
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Create your first route to start forwarding traffic into protected backend services.')
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Route' })).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw backend error when loading routes fails', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing routes.')
    ).toBeInTheDocument()
    expect(screen.queryByText('unauthorized')).not.toBeInTheDocument()
  })

  it('does not fall back to the empty route state when the route directory fails to load', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    expect(
      await screen.findByText(
        'The current route list could not be loaded. Resolve the current error before reviewing or editing routes.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('No routes configured')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add Route' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create First Route' })).not.toBeInTheDocument()
  })

  it('does not show normal directory counts or metrics when the route directory is unavailable', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await screen.findByText(
      'The current route list could not be loaded. Resolve the current error before reviewing or editing routes.'
    )

    expect(screen.queryByText('0 configured route entries')).not.toBeInTheDocument()
    expect(screen.queryByText('Total Routes')).not.toBeInTheDocument()
    expect(screen.queryByText('Active Routes')).not.toBeInTheDocument()
    expect(screen.queryByText('Host Coverage')).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw invalid token error when loading routes fails', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing routes.')
    ).toBeInTheDocument()
    expect(screen.queryByText('invalid token')).not.toBeInTheDocument()
  })

  it('shows a directory loading message instead of save-oriented copy when loading routes fails with a store error', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(
      new ApiError('failed to list routes', 500, 'route_store_failure')
    )

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    expect(
      await screen.findByText('The route directory could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'The route directory could not be loaded. Resolve the loading error before reviewing or editing routes.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'The current route list could not be loaded. Resolve the current error before reviewing or editing routes.'
      )
    ).not.toBeInTheDocument()
    expect(screen.queryByText('The route change could not be saved. Try again in a moment.')).not.toBeInTheDocument()
    expect(screen.queryByText('failed to list routes')).not.toBeInTheDocument()
  })

  it('shows a helpful backend validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('backend must be a valid http or https URL', 400, 'invalid_route_backend')
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'not-a-url')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    const dialog = await screen.findByRole('dialog')

    expect(await within(dialog).findByText('Enter a valid http:// or https:// backend target.')).toBeInTheDocument()
    expect(screen.queryByText('backend must be a valid http or https URL')).not.toBeInTheDocument()
  })

  it('shows a helpful backend weight validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('backend weight must be greater than 0', 400, 'invalid_route_backend_weight')
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    const dialog = await screen.findByRole('dialog')

    expect(await within(dialog).findByText('Use a backend weight greater than 0 for every configured backend.')).toBeInTheDocument()
    expect(screen.queryByText('backend weight must be greater than 0')).not.toBeInTheDocument()
  })

  it('shows pooled backend and tls summaries in the route table', async () => {
    vi.mocked(routesApi.list).mockResolvedValue([
      {
        id: 'route-lb',
        name: 'API Edge',
        host: 'api.example.com',
        path_prefix: '/api',
        backend: '',
        backends: [
          {
            url: 'http://backend-a.example.com',
            weight: 2,
            dial_timeout_ms: 1500,
            read_timeout_ms: 2500,
            write_timeout_ms: 3500,
            max_idle_conns: 8,
          },
          { url: 'https://backend-b.example.com', weight: 1 },
        ],
        strip_prefix: true,
        enabled: true,
        priority: 50,
        tls_enabled: true,
        tls_cert: '/etc/ssl/certs/api.pem',
        tls_key: '/etc/ssl/private/api.key',
        timeout_ms: 4500,
        retry_attempts: 3,
        created_at: '',
        updated_at: '',
      } as any,
    ])

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    const table = await screen.findByRole('table')
    expect(within(table).getByText('2 backends')).toBeInTheDocument()
    expect(within(table).getByText('TLS enabled')).toBeInTheDocument()
    expect(within(table).getByText('Timeout 4500ms')).toBeInTheDocument()
    expect(within(table).getByText('Retries 3')).toBeInTheDocument()
    expect(within(table).getByText('Backend 1 dial 1500ms')).toBeInTheDocument()
    expect(within(table).getByText('Backend 1 read 2500ms')).toBeInTheDocument()
    expect(within(table).getByText('Backend 1 write 3500ms')).toBeInTheDocument()
    expect(within(table).getByText('Backend 1 idle 8')).toBeInTheDocument()
    expect(within(table).getByText('http://backend-a.example.com')).toBeInTheDocument()
  })

  it('shows a helpful redirect code validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('redirect_code must be 0, 301, or 302', 400, 'invalid_route_redirect_code')
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText('Use only 301 or 302 for redirects, or 0 to disable redirection.')
    ).toBeInTheDocument()
    expect(screen.queryByText('redirect_code must be 0, 301, or 302')).not.toBeInTheDocument()
  })

  it('shows a helpful host validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError(
        'host must be a hostname or IP address without scheme, port, or path',
        400,
        'invalid_route_host'
      )
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Host'), 'https://api.example.com')
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText(
        'Enter a hostname or IP address only. Do not include a scheme, port, or path.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('host must be a hostname or IP address without scheme, port, or path')
    ).not.toBeInTheDocument()
  })

  it('explains which path match modes require a leading slash when the backend rejects the route prefix', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('path_prefix must start with /', 400, 'invalid_route_path_prefix')
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), 'billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    expect(
      await within(await screen.findByRole('dialog')).findByText(
        'For plain-prefix, exact, and prefix-stop matches, start the path prefix with /. Regex modes can use a raw pattern.'
      )
    ).toBeInTheDocument()
  })

  it('shows a helpful path match mode validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError(
        'path_match_mode must be one of prefix, exact, stop, regex, or regex_i',
        400,
        'invalid_route_path_match_mode'
      )
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    expect(
      await within(await screen.findByRole('dialog')).findByText(
        'Choose a supported path match mode before saving the route.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('path_match_mode must be one of prefix, exact, stop, regex, or regex_i')
    ).not.toBeInTheDocument()
  })

  it('shows a helpful regex validation message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError(
        'path_prefix must be a valid regular expression for the selected path match mode',
        400,
        'invalid_route_path_regex'
      )
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.click(screen.getByLabelText('Path Prefix'))
    await user.paste('[')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    expect(
      await within(await screen.findByRole('dialog')).findByText(
        'Enter a valid regular expression when using a regex path match mode.'
      )
    ).toBeInTheDocument()
    expect(
      screen.queryByText('path_prefix must be a valid regular expression for the selected path match mode')
    ).not.toBeInTheDocument()
  })

  it('shows a reserved path guidance message instead of the raw API error', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError(
        'path_prefix conflicts with reserved control-plane paths',
        400,
        'reserved_route_path_prefix'
      )
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/_authgate/internal')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    expect(
      await within(await screen.findByRole('dialog')).findByText(
        'This path prefix is reserved for the control plane. Choose a different route path.'
      )
    ).toBeInTheDocument()
  })

  it('shows permission guidance instead of the raw backend error when route changes are rejected', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('insufficient permissions', 403, 'insufficient_permissions')
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    expect(
      await within(await screen.findByRole('dialog')).findByText(
        'Your account cannot manage routes. Ask an editor or administrator to apply the change.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('insufficient permissions')).not.toBeInTheDocument()
  })

  it('retranslates the current route list error when the language changes', async () => {
    vi.mocked(routesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    const view = await renderWithI18n(<RoutesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing routes.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('你的会话已失效。请重新登录后再管理路由。')).toBeInTheDocument()
    expect(
      screen.queryByText('Your session has expired. Sign in again before managing routes.')
    ).not.toBeInTheDocument()
  })

  it('retranslates the current route form error when the language changes', async () => {
    vi.mocked(routesApi.create).mockRejectedValue(
      new ApiError('backend must be a valid http or https URL', 400, 'invalid_route_backend')
    )

    const user = userEvent.setup()
    const view = await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'not-a-url')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    const dialog = await screen.findByRole('dialog')

    expect(
      await within(dialog).findByText('Enter a valid http:// or https:// backend target.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(within(dialog).getByText('请输入有效的 http:// 或 https:// 后端地址。')).toBeInTheDocument()
    expect(
      screen.queryByText('Enter a valid http:// or https:// backend target.')
    ).not.toBeInTheDocument()
  })

  it('creates a pooled-backend tls route without a legacy backend target', async () => {
    vi.mocked(routesApi.create).mockResolvedValue({
      id: 'route-created',
      name: 'API Edge',
      host: 'api.example.com',
      path_prefix: '/api',
      backend: '',
      backends: [
        { url: 'http://backend-a.example.com', weight: 2 },
      ],
      strip_prefix: true,
      enabled: true,
      priority: 0,
      tls_enabled: true,
      tls_cert: '/etc/ssl/certs/api.pem',
      tls_key: '/etc/ssl/private/api.key',
      created_at: '',
      updated_at: '',
    } as any)

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Add Route' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Add Route' }))
    await user.type(screen.getByLabelText('Path Prefix'), '/api')
    await user.click(screen.getByRole('button', { name: 'Add Backend' }))
    await user.type(screen.getByLabelText('Backend URL 1'), 'http://backend-a.example.com')
    await user.clear(screen.getByLabelText('Weight 1'))
    await user.type(screen.getByLabelText('Weight 1'), '2')
    await user.click(screen.getByLabelText('TLS Termination'))
    await user.type(screen.getByLabelText('Certificate Path'), '/etc/ssl/certs/api.pem')
    await user.type(screen.getByLabelText('Private Key Path'), '/etc/ssl/private/api.key')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    await waitFor(() => {
      expect(routesApi.create).toHaveBeenCalledWith(
        expect.objectContaining({
          backend: '',
          backends: [{ url: 'http://backend-a.example.com', weight: 2 }],
          tls_enabled: true,
          tls_cert: '/etc/ssl/certs/api.pem',
          tls_key: '/etc/ssl/private/api.key',
        })
      )
    })
  })

  it('keeps the newest route refresh results when an older delete refresh resolves later', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))

    const deleteResolvers = new Map<string, () => void>()
    const listResolvers: Array<(value: Awaited<ReturnType<typeof routesApi.list>>) => void> = []

    vi.mocked(routesApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          listResolvers.push(resolve)
        })
    )
    vi.mocked(routesApi.delete).mockImplementation(
      (id: string) =>
        new Promise<void>((resolve) => {
          deleteResolvers.set(id, resolve)
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<RoutesPage />, { locale: 'en' })

    await act(async () => {
      listResolvers[0]?.([
        {
          id: 'route-1',
          name: 'Billing',
          host: 'billing.example.com',
          path_prefix: '/billing',
          backend: 'http://billing.internal',
          strip_prefix: true,
          enabled: true,
          priority: 10,
          created_at: '',
          updated_at: '',
        },
        {
          id: 'route-2',
          name: 'Reports',
          host: 'reports.example.com',
          path_prefix: '/reports',
          backend: 'http://reports.internal',
          strip_prefix: true,
          enabled: true,
          priority: 20,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    const table = await screen.findByRole('table')
    const billingRow = within(table).getByText('Billing').closest('tr')
    const reportsRow = within(table).getByText('Reports').closest('tr')

    expect(billingRow).not.toBeNull()
    expect(reportsRow).not.toBeNull()

    await user.click(within(billingRow as HTMLTableRowElement).getByLabelText('Delete'))
    await user.click(within(reportsRow as HTMLTableRowElement).getByLabelText('Delete'))

    await act(async () => {
      deleteResolvers.get('route-1')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(listResolvers).toHaveLength(2)
    })

    await act(async () => {
      deleteResolvers.get('route-2')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(listResolvers).toHaveLength(3)
    })

    await act(async () => {
      listResolvers[2]?.([
        {
          id: 'route-fresh',
          name: 'Fresh Snapshot',
          host: 'fresh.example.com',
          path_prefix: '/fresh',
          backend: 'http://fresh.internal',
          strip_prefix: true,
          enabled: true,
          priority: 30,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(await within(table).findByText('Fresh Snapshot')).toBeInTheDocument()

    await act(async () => {
      listResolvers[1]?.([
        {
          id: 'route-stale',
          name: 'Stale Snapshot',
          host: 'stale.example.com',
          path_prefix: '/stale',
          backend: 'http://stale.internal',
          strip_prefix: true,
          enabled: true,
          priority: 40,
          created_at: '',
          updated_at: '',
        },
      ])
      await Promise.resolve()
    })

    expect(within(table).getByText('Fresh Snapshot')).toBeInTheDocument()
    expect(within(table).queryByText('Stale Snapshot')).not.toBeInTheDocument()
    expect(within(table).queryByText('Reports')).not.toBeInTheDocument()
    expect(within(table).queryByText('Billing')).not.toBeInTheDocument()
  })
})
