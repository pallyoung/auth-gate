import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { RouteAuthPage } from './RouteAuthPage'
import { routeAuthApi, apiKeyApi } from '../lib/api/route-auth'
import { routesApi } from '../lib/api/routes'
import { renderWithI18n } from '../test/render'
import type { ApiKey, ApiKeyCreateResponse, Route, RouteAuthConfig } from '../lib/api/types'

vi.mock('../lib/api/routes', () => ({
  routesApi: {
    list: vi.fn(),
  },
}))

vi.mock('../lib/api/route-auth', () => ({
  routeAuthApi: {
    getConfig: vi.fn(),
    updateConfig: vi.fn(),
    deleteConfig: vi.fn(),
  },
  apiKeyApi: {
    list: vi.fn(),
    create: vi.fn(),
    updateName: vi.fn(),
    rotate: vi.fn(),
    expire: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => ({
    id: 'admin-1',
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

const route: Route = {
  id: 'route-1',
  name: 'Public API',
  host: 'api.example.com',
  path_prefix: '/api',
  backend: 'http://127.0.0.1:9000',
  strip_prefix: true,
  enabled: true,
  priority: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const secondRoute: Route = {
  ...route,
  id: 'route-2',
  name: 'Reports API',
  host: 'reports.example.com',
  path_prefix: '/reports',
}

const config: RouteAuthConfig = {
  route_id: route.id,
  api_key_enabled: true,
  api_key_header: 'X-API-Key',
  gateway_enabled: false,
  whitelist: ['127.0.0.1/32'],
  rate_limit: 10,
  burst: 20,
  cors_allowed_origins: 'https://app.example.com',
  cors_allowed_methods: 'GET',
  cors_allowed_headers: 'Authorization,Content-Type',
  cors_allow_credentials: true,
  cors_max_age: 86400,
}

const secondConfig: RouteAuthConfig = {
  ...config,
  route_id: secondRoute.id,
  cors_allowed_methods: 'POST',
}

const activeKey: ApiKey = {
  id: 'key-1',
  route_id: route.id,
  name: 'Primary',
  key_prefix: 'ag_test_',
  secret: 'ag_test_secret',
  status: 'active',
  created_at: '2026-01-01T00:00:00Z',
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

async function openConfigModal() {
  const user = userEvent.setup()
  await renderWithI18n(<RouteAuthPage />, { locale: 'en' })

  const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
  await user.click(editButtons[0])
  await screen.findByLabelText('Allowed Methods')
  return user
}

describe('RouteAuthPage configuration editing', () => {
  beforeEach(() => {
    vi.mocked(routesApi.list).mockReset()
    vi.mocked(routesApi.list).mockResolvedValue([route])
    vi.mocked(routeAuthApi.getConfig).mockReset()
    vi.mocked(routeAuthApi.getConfig).mockResolvedValue(config)
    vi.mocked(routeAuthApi.updateConfig).mockReset()
    vi.mocked(routeAuthApi.updateConfig).mockResolvedValue(config)
    vi.mocked(apiKeyApi.list).mockReset()
    vi.mocked(apiKeyApi.list).mockResolvedValue([])
    vi.mocked(apiKeyApi.create).mockReset()
    vi.mocked(apiKeyApi.rotate).mockReset()
    vi.mocked(apiKeyApi.expire).mockReset()
    vi.mocked(apiKeyApi.delete).mockReset()
  })

  it('changes locale in an open editor without reloading its draft or losing translated validation and dates', async () => {
    vi.mocked(apiKeyApi.list).mockResolvedValue([{ ...activeKey, expires_at: '2026-09-20T12:00:00Z' }])
    const user = userEvent.setup()
    const { i18n } = await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    await user.click((await screen.findAllByRole('button', { name: 'Edit' }))[0])
    const methods = await screen.findByLabelText('Allowed Methods')
    await user.clear(methods)
    await user.type(methods, 'GET,POST')
    const rate = screen.getByLabelText('Rate (req/s)')
    await user.clear(rate)
    await user.type(rate, '-1')
    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))
    expect(screen.getByText('Enter a whole number greater than or equal to 0.')).toBeInTheDocument()
    const calls = vi.mocked(routeAuthApi.getConfig).mock.calls.length
    await act(async () => { await i18n.changeLanguage('zh-CN') })
    expect(screen.getByLabelText('允许方法')).toHaveValue('GET,POST')
    expect(screen.getByText('请输入大于或等于 0 的整数。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '显示 API Key' })).toBeInTheDocument()
    expect(screen.getByText(/2026\/9\/20/)).toBeInTheDocument()
    expect(routeAuthApi.getConfig).toHaveBeenCalledTimes(calls)
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
  })

  it('keeps a comma-separated HTTP method draft intact without saving on each keystroke', async () => {
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')

    await user.clear(methodsInput)
    await user.type(methodsInput, 'GET,POST,OPTIONS')

    expect(methodsInput).toHaveValue('GET,POST,OPTIONS')
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
  })

  it('normalizes and saves the configuration in a single request', async () => {
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')

    await user.clear(methodsInput)
    await user.type(methodsInput, ' get, post, options ')
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))

    await waitFor(() => {
      expect(routeAuthApi.updateConfig).toHaveBeenCalledTimes(1)
    })
    expect(routeAuthApi.updateConfig).toHaveBeenCalledWith(
      route.id,
      expect.objectContaining({
        cors_allowed_methods: 'GET,POST,OPTIONS',
      })
    )
  })

  it('allows numeric configuration fields to stay empty while editing', async () => {
    const user = await openConfigModal()
    const maxAgeInput = screen.getByLabelText('Max Age (s)')

    await user.clear(maxAgeInput)

    expect(maxAgeInput).toHaveValue(null)
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
  })

  it('rejects invalid HTTP method tokens only when saving', async () => {
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')

    await user.clear(methodsInput)
    await user.type(methodsInput, 'GET,PO ST')

    expect(methodsInput).toHaveValue('GET,PO ST')
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))

    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
    expect(await screen.findByText('Use comma-separated HTTP method tokens, such as GET, POST, or OPTIONS.')).toBeInTheDocument()
  })

  it('blocks saving when the route auth configuration could not be loaded', async () => {
    vi.mocked(routeAuthApi.getConfig)
      .mockReset()
      .mockResolvedValueOnce(config)
      .mockRejectedValueOnce(new Error('storage unavailable'))

    const user = userEvent.setup()
    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])

    expect(await screen.findByText('The route authentication configuration could not be loaded. Close this dialog and try again.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save Configuration' })).not.toBeInTheDocument()
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
  })

  it('shows an unavailable list state and disables quick toggles when a route config fails to load', async () => {
    vi.mocked(routeAuthApi.getConfig).mockRejectedValue(new Error('storage unavailable'))

    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })

    expect(await screen.findByText('Unavailable')).toBeInTheDocument()
    expect(screen.queryByText('Public')).not.toBeInTheDocument()
    const quickToggles = screen.getAllByRole('checkbox')
    expect(quickToggles).toHaveLength(4)
    quickToggles.forEach((toggle) => expect(toggle).toBeDisabled())
    expect(routeAuthApi.updateConfig).not.toHaveBeenCalled()
  })

  it('keeps an unsaved configuration draft after creating an API key', async () => {
    vi.mocked(apiKeyApi.create).mockResolvedValue({ ...activeKey, secret: 'ag_new_secret' })
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')
    await user.clear(methodsInput)
    await user.type(methodsInput, 'PATCH')

    await user.click(screen.getByRole('button', { name: 'New Key' }))
    await user.click(screen.getByRole('button', { name: 'Create' }))

    await waitFor(() => expect(apiKeyApi.create).toHaveBeenCalledTimes(1))
    expect(methodsInput).toHaveValue('PATCH')
    expect(await screen.findByText('ag_new_secret')).toBeInTheDocument()
    expect(routeAuthApi.getConfig).toHaveBeenCalledTimes(2)
  })

  it.each([
    ['rotating', 'Rotate key'],
    ['revoking', 'Revoke'],
    ['deleting', 'Delete'],
  ])('keeps an unsaved configuration draft after %s an API key', async (operation, actionTitle) => {
    vi.mocked(apiKeyApi.list).mockResolvedValue([activeKey])
    vi.mocked(apiKeyApi.rotate).mockResolvedValue({ ...activeKey, secret: 'ag_rotated_secret' })
    vi.mocked(apiKeyApi.expire).mockResolvedValue(undefined)
    vi.mocked(apiKeyApi.delete).mockResolvedValue(undefined)
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')
    await user.clear(methodsInput)
    await user.type(methodsInput, 'PATCH')

    await user.click(screen.getByTitle(actionTitle))

    await waitFor(() => {
      if (actionTitle === 'Rotate key') expect(apiKeyApi.rotate).toHaveBeenCalledTimes(1)
      if (actionTitle === 'Revoke') expect(apiKeyApi.expire).toHaveBeenCalledTimes(1)
      if (actionTitle === 'Delete') expect(apiKeyApi.delete).toHaveBeenCalledTimes(1)
      expect(apiKeyApi.list).toHaveBeenCalledTimes(3)
    })
    await act(async () => {
      await Promise.resolve()
    })
    expect(methodsInput).toHaveValue('PATCH')
    expect(routeAuthApi.getConfig).toHaveBeenCalledTimes(2)
  })

  it('shows a newly created one-time secret even when the key list refresh fails', async () => {
    vi.mocked(apiKeyApi.list)
      .mockReset()
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([])
      .mockRejectedValueOnce(new Error('refresh unavailable'))
    vi.mocked(apiKeyApi.create).mockResolvedValue({ ...activeKey, secret: 'ag_new_secret' })
    const user = await openConfigModal()

    await user.click(screen.getByRole('button', { name: 'New Key' }))
    await user.click(screen.getByRole('button', { name: 'Create' }))

    expect(await screen.findByText('ag_new_secret')).toBeInTheDocument()
    expect(await screen.findByText('The API key changed, but the key list could not be refreshed. Try reopening this dialog.')).toBeInTheDocument()
  })

  it('shows a rotated one-time secret even when the key list refresh fails', async () => {
    vi.mocked(apiKeyApi.list)
      .mockReset()
      .mockResolvedValueOnce([activeKey])
      .mockResolvedValueOnce([activeKey])
      .mockRejectedValueOnce(new Error('refresh unavailable'))
    vi.mocked(apiKeyApi.rotate).mockResolvedValue({ ...activeKey, secret: 'ag_rotated_secret' })
    const user = await openConfigModal()

    await user.click(screen.getByTitle('Rotate key'))

    expect(await screen.findByText('ag_rotated_secret')).toBeInTheDocument()
    expect(await screen.findByText('The API key changed, but the key list could not be refreshed. Try reopening this dialog.')).toBeInTheDocument()
  })

  it('sends only one create request for two clicks in the same tick', async () => {
    const createRequest = deferred<ApiKeyCreateResponse>()
    vi.mocked(apiKeyApi.create).mockReturnValue(createRequest.promise)
    const user = await openConfigModal()
    await user.click(screen.getByRole('button', { name: 'New Key' }))
    const createButton = screen.getByRole('button', { name: 'Create' })

    fireEvent.click(createButton)
    fireEvent.click(createButton)

    expect(apiKeyApi.create).toHaveBeenCalledTimes(1)
    await act(async () => {
      createRequest.resolve({ ...activeKey, secret: 'ag_new_secret' })
      await createRequest.promise
    })
  })

  it('keeps the detail dialog open during key creation so the one-time secret is not lost', async () => {
    const createRequest = deferred<ApiKeyCreateResponse>()
    vi.mocked(apiKeyApi.create).mockReturnValue(createRequest.promise)
    const user = await openConfigModal()
    await user.click(screen.getByRole('button', { name: 'New Key' }))
    await user.click(screen.getByRole('button', { name: 'Create' }))
    const closeButton = screen.getByRole('button', { name: 'Close modal' })

    expect(closeButton).toBeDisabled()
    await user.keyboard('{Escape}')
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    await act(async () => {
      createRequest.resolve({ ...activeKey, secret: 'ag_new_secret' })
      await createRequest.promise
    })
    expect(await screen.findByText('ag_new_secret')).toBeInTheDocument()
  })

  it('does not show a key secret returned for a previously opened route', async () => {
    const firstCreate = deferred<ApiKeyCreateResponse>()
    vi.mocked(routesApi.list).mockResolvedValue([route, secondRoute])
    vi.mocked(routeAuthApi.getConfig).mockImplementation((routeId) =>
      Promise.resolve(routeId === route.id ? config : secondConfig)
    )
    vi.mocked(apiKeyApi.create).mockReturnValue(firstCreate.promise)
    const user = userEvent.setup()
    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])
    await screen.findByLabelText('Allowed Methods')
    await user.click(screen.getByRole('button', { name: 'New Key' }))
    await user.click(screen.getByRole('button', { name: 'Create' }))

    await user.click(editButtons[2])
    expect(await screen.findByRole('heading', { name: /Reports API/ })).toBeInTheDocument()
    await act(async () => {
      firstCreate.resolve({ ...activeKey, secret: 'ag_old_route_secret' })
      await firstCreate.promise
    })

    expect(screen.queryByText('ag_old_route_secret')).not.toBeInTheDocument()
  })

  it('does not let an older rotation overwrite the latest one-time secret', async () => {
    const firstRotate = deferred<ApiKeyCreateResponse>()
    const secondRotate = deferred<ApiKeyCreateResponse>()
    vi.mocked(routesApi.list).mockResolvedValue([route, secondRoute])
    vi.mocked(routeAuthApi.getConfig).mockImplementation((routeId) =>
      Promise.resolve(routeId === route.id ? config : secondConfig)
    )
    vi.mocked(apiKeyApi.list).mockResolvedValue([activeKey])
    vi.mocked(apiKeyApi.rotate)
      .mockReturnValueOnce(firstRotate.promise)
      .mockReturnValueOnce(secondRotate.promise)
    const user = userEvent.setup()
    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])
    await screen.findByLabelText('Allowed Methods')
    await user.click(screen.getByTitle('Rotate key'))

    await user.click(editButtons[2])
    await screen.findByRole('heading', { name: /Reports API/ })
    await user.click(screen.getByTitle('Rotate key'))
    await act(async () => {
      secondRotate.resolve({ ...activeKey, route_id: secondRoute.id, secret: 'ag_latest_secret' })
      await secondRotate.promise
    })
    expect(await screen.findByText('ag_latest_secret')).toBeInTheDocument()

    await act(async () => {
      firstRotate.resolve({ ...activeKey, secret: 'ag_older_secret' })
      await firstRotate.promise
    })
    expect(screen.getByText('ag_latest_secret')).toBeInTheDocument()
    expect(screen.queryByText('ag_older_secret')).not.toBeInTheDocument()
  })

  it('asks before discarding an unsaved modal draft', async () => {
    const user = await openConfigModal()
    await user.clear(screen.getByLabelText('Allowed Methods'))
    await user.type(screen.getByLabelText('Allowed Methods'), 'PATCH')

    await user.click(screen.getByRole('button', { name: 'Close modal' }))

    expect(await screen.findByText('Discard unsaved changes?')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Keep Editing' }))
    expect(screen.getByLabelText('Allowed Methods')).toHaveValue('PATCH')

    await user.click(screen.getByRole('button', { name: 'Close modal' }))
    await user.click(screen.getByRole('button', { name: 'Discard Changes' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('uses a separate discard dialog with Keep Editing as the initial action', async () => {
    const user = await openConfigModal()
    await user.clear(screen.getByLabelText('Allowed Methods'))
    await user.type(screen.getByLabelText('Allowed Methods'), 'PATCH')

    await user.click(screen.getByRole('button', { name: 'Close modal' }))

    const discardDialog = await screen.findByRole('alertdialog', { name: 'Discard unsaved changes?' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(within(discardDialog).getByRole('button', { name: 'Keep Editing' })).toHaveFocus()
    expect(within(discardDialog).getByRole('button', { name: 'Discard Changes' })).toBeInTheDocument()
  })

  it('ignores a stale detail response after another route is opened', async () => {
    const firstDetail = deferred<RouteAuthConfig>()
    const calls = new Map<string, number>()
    vi.mocked(routesApi.list).mockResolvedValue([route, secondRoute])
    vi.mocked(routeAuthApi.getConfig).mockImplementation((routeId) => {
      const call = (calls.get(routeId) ?? 0) + 1
      calls.set(routeId, call)
      if (call === 1) return Promise.resolve(routeId === route.id ? config : secondConfig)
      if (routeId === route.id) return firstDetail.promise
      return Promise.resolve(secondConfig)
    })

    const user = userEvent.setup()
    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])
    await user.click(screen.getByRole('button', { name: 'Close modal' }))
    await user.click(editButtons[2])
    expect(await screen.findByLabelText('Allowed Methods')).toHaveValue('POST')

    await act(async () => {
      firstDetail.resolve(config)
      await firstDetail.promise
    })

    expect(screen.getByLabelText('Allowed Methods')).toHaveValue('POST')
    expect(screen.getByRole('heading', { name: /Reports API/ })).toBeInTheDocument()
  })

  it('ignores a stale save completion after another route is opened', async () => {
    const firstSave = deferred<RouteAuthConfig>()
    vi.mocked(routesApi.list).mockResolvedValue([route, secondRoute])
    vi.mocked(routeAuthApi.getConfig).mockImplementation((routeId) =>
      Promise.resolve(routeId === route.id ? config : secondConfig)
    )
    vi.mocked(routeAuthApi.updateConfig).mockReturnValue(firstSave.promise)

    const user = userEvent.setup()
    await renderWithI18n(<RouteAuthPage />, { locale: 'en' })
    const editButtons = await screen.findAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])
    const methodsInput = await screen.findByLabelText('Allowed Methods')
    await user.clear(methodsInput)
    await user.type(methodsInput, 'PATCH')
    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))
    await user.click(editButtons[2])
    expect(await screen.findByLabelText('Allowed Methods')).toHaveValue('POST')

    await act(async () => {
      firstSave.resolve({ ...config, cors_allowed_methods: 'PATCH' })
      await firstSave.promise
    })

    expect(screen.getByLabelText('Allowed Methods')).toHaveValue('POST')
  })

  it('does not offer to discard changes while a save is already pending', async () => {
    const pendingSave = deferred<RouteAuthConfig>()
    vi.mocked(routeAuthApi.updateConfig).mockReturnValue(pendingSave.promise)
    const user = await openConfigModal()
    await user.clear(screen.getByLabelText('Allowed Methods'))
    await user.type(screen.getByLabelText('Allowed Methods'), 'PATCH')
    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))

    await user.click(screen.getByRole('button', { name: 'Close modal' }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.queryByText('Discard unsaved changes?')).not.toBeInTheDocument()

    await act(async () => {
      pendingSave.resolve({ ...config, cors_allowed_methods: 'PATCH' })
      await pendingSave.promise
    })
  })

  it('disables configuration controls while a save is pending', async () => {
    const pendingSave = deferred<RouteAuthConfig>()
    vi.mocked(routeAuthApi.updateConfig).mockReturnValue(pendingSave.promise)
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')
    await user.clear(methodsInput)
    await user.type(methodsInput, 'PATCH')

    await user.click(screen.getByRole('button', { name: 'Save Configuration' }))

    expect(methodsInput).toBeDisabled()
    expect(screen.getByLabelText('Max Age (s)')).toBeDisabled()
    await act(async () => {
      pendingSave.resolve({ ...config, cors_allowed_methods: 'PATCH' })
      await pendingSave.promise
    })
  })

  it('closes an existing discard prompt and prevents discarding once a save starts', async () => {
    const pendingSave = deferred<RouteAuthConfig>()
    vi.mocked(routeAuthApi.updateConfig).mockReturnValue(pendingSave.promise)
    const user = await openConfigModal()
    const methodsInput = screen.getByLabelText('Allowed Methods')
    const saveButton = screen.getByRole('button', { name: 'Save Configuration' })
    await user.clear(methodsInput)
    await user.type(methodsInput, 'PATCH')
    await user.click(screen.getByRole('button', { name: 'Close modal' }))
    await screen.findByText('Discard unsaved changes?')

    fireEvent.click(saveButton)

    expect(screen.queryByText('Discard unsaved changes?')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Discard Changes' })).not.toBeInTheDocument()
    expect(methodsInput).toBeDisabled()
    await act(async () => {
      pendingSave.resolve({ ...config, cors_allowed_methods: 'PATCH' })
      await pendingSave.promise
    })
  })

  it('sends only one save request for two clicks in the same tick', async () => {
    const pendingSave = deferred<RouteAuthConfig>()
    vi.mocked(routeAuthApi.updateConfig).mockReturnValue(pendingSave.promise)
    const user = await openConfigModal()
    const saveButton = screen.getByRole('button', { name: 'Save Configuration' })
    await user.clear(screen.getByLabelText('Allowed Methods'))
    await user.type(screen.getByLabelText('Allowed Methods'), 'PATCH')

    fireEvent.click(saveButton)
    fireEvent.click(saveButton)

    expect(routeAuthApi.updateConfig).toHaveBeenCalledTimes(1)
    await act(async () => {
      pendingSave.resolve({ ...config, cors_allowed_methods: 'PATCH' })
      await pendingSave.promise
    })
  })
})
