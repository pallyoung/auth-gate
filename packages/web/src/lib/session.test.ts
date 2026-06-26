import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('session integration', () => {
  beforeEach(async () => {
    localStorage.clear()
    vi.resetModules()
  })

  it('refreshes the current user from /auth/me instead of local cached user data', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        id: 'user-1',
        username: 'fresh-user',
        role: 'editor',
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }))

    vi.stubGlobal('fetch', fetchMock)

    const { refreshSessionUser } = await import('./session')
    const { getSessionUser } = await import('./session-store')

    const user = await refreshSessionUser()

    expect(fetchMock).toHaveBeenCalledWith('/api/auth/me', expect.objectContaining({
      headers: expect.any(Headers),
    }))
    expect(user).toEqual({
      id: 'user-1',
      username: 'fresh-user',
      role: 'editor',
    })
    expect(getSessionUser()).toEqual(user)
  })

  it('keeps the existing session when refreshing /auth/me fails for a non-auth error', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: {
          code: 'session_store_failure',
          message: 'failed to load user',
        },
      }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      }))

    vi.stubGlobal('fetch', fetchMock)

    const { refreshSessionUser } = await import('./session')
    const { getSessionToken, getSessionUser, setSession } = await import('./session-store')

    setSession({
      token: 'token-123',
      user: {
        id: 'user-1',
        username: 'cached-user',
        role: 'editor',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    await expect(refreshSessionUser()).rejects.toMatchObject({
      message: 'failed to load user',
      status: 500,
      code: 'session_store_failure',
    })

    expect(getSessionToken()).toBe('token-123')
    expect(localStorage.getItem('token')).toBe('token-123')
    expect(getSessionUser()).toEqual({
      id: 'user-1',
      username: 'cached-user',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })
  })

  it('clears the session when an authenticated request returns 401', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: 'invalid_token', message: 'invalid token' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }))

    vi.stubGlobal('fetch', fetchMock)

    const { routesApi } = await import('./api/routes')
    const { getSessionToken } = await import('./session-store')

    await expect(routesApi.list()).rejects.toMatchObject({
      message: 'invalid token',
      status: 401,
      code: 'invalid_token',
    })
    expect(getSessionToken()).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('does not clear a newer session when an earlier authenticated request returns 401', async () => {
    const deferredResponse = {} as {
      resolve: (response: Response) => void
      promise: Promise<Response>
    }
    deferredResponse.promise = new Promise<Response>((resolve) => {
      deferredResponse.resolve = resolve
    })

    vi.stubGlobal('fetch', vi.fn(() => deferredResponse.promise))

    const { routesApi } = await import('./api/routes')
    const { getSessionToken, getSessionUser, setSession } = await import('./session-store')

    setSession({
      token: 'token-old',
      user: {
        id: 'user-old',
        username: 'old-user',
        role: 'editor',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    const listPromise = routesApi.list()

    setSession({
      token: 'token-new',
      user: {
        id: 'user-new',
        username: 'new-user',
        role: 'admin',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    deferredResponse.resolve(new Response(JSON.stringify({
      error: { code: 'invalid_token', message: 'invalid token' },
    }), {
      status: 401,
      headers: { 'Content-Type': 'application/json' },
    }))

    await expect(listPromise).rejects.toMatchObject({
      message: 'invalid token',
      status: 401,
      code: 'invalid_token',
    })

    expect(getSessionToken()).toBe('token-new')
    expect(localStorage.getItem('token')).toBe('token-new')
    expect(getSessionUser()).toEqual({
      id: 'user-new',
      username: 'new-user',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true, can_manage_hosts: true,
      },
    })
  })

  it('does not clear the control-plane session when route access login returns 401', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: 'route_access_denied', message: 'route access denied' },
      }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }))

    vi.stubGlobal('fetch', fetchMock)

    const { authApi } = await import('./api/auth')
    const { getSessionToken } = await import('./session-store')

    await expect(authApi.accessLogin({
      route_id: 'route-1',
      username: 'member',
      password: 'bad-password',
      next: '/cloud',
    })).rejects.toMatchObject({
      message: 'route access denied',
      status: 401,
      code: 'route_access_denied',
    })

    expect(getSessionToken()).toBe('token-123')
    expect(localStorage.getItem('token')).toBe('token-123')
  })

  it('does not overwrite a newer session user when an earlier /auth/me refresh resolves late', async () => {
    const deferredResponse = {} as {
      resolve: (response: Response) => void
      promise: Promise<Response>
    }
    deferredResponse.promise = new Promise<Response>((resolve) => {
      deferredResponse.resolve = resolve
    })

    vi.stubGlobal('fetch', vi.fn(() => deferredResponse.promise))

    const { refreshSessionUser } = await import('./session')
    const { getSessionToken, getSessionUser, setSession } = await import('./session-store')

    setSession({
      token: 'token-old',
      user: {
        id: 'user-old',
        username: 'old-user',
        role: 'editor',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    const refreshPromise = refreshSessionUser()

    setSession({
      token: 'token-new',
      user: {
        id: 'user-new',
        username: 'new-user',
        role: 'admin',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    deferredResponse.resolve(new Response(JSON.stringify({
      id: 'user-old',
      username: 'old-user-from-server',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }))

    await expect(refreshPromise).resolves.toEqual({
      id: 'user-old',
      username: 'old-user-from-server',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    expect(getSessionToken()).toBe('token-new')
    expect(getSessionUser()).toEqual({
      id: 'user-new',
      username: 'new-user',
      role: 'admin',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: true,
        can_view_logs: true, can_manage_hosts: true,
      },
    })
  })

  it('persists the last known session user so bootstrap refresh failures can recover the control-plane shell', async () => {
    const { getSessionUser, setSession } = await import('./session-store')

    setSession({
      token: 'token-123',
      user: {
        id: 'user-1',
        username: 'cached-user',
        role: 'editor',
        features: {
          certificates: false,
        },
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    expect(JSON.parse(localStorage.getItem('auth-gate.session-user') || 'null')).toEqual({
      id: 'user-1',
      username: 'cached-user',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
      features: {
        certificates: false,
      },
    })

    vi.resetModules()

    const reloadedStore = await import('./session-store')

    expect(reloadedStore.getSessionUser()).toEqual(getSessionUser())

    reloadedStore.clearSession()

    expect(localStorage.getItem('auth-gate.session-user')).toBeNull()
  })

  it('syncs session state from storage events so other tabs can log the current tab out', async () => {
    const { getSessionNotice, getSessionToken, getSessionUser, setSession, subscribeSession } = await import('./session-store')

    setSession({
      token: 'token-123',
      user: {
        id: 'user-1',
        username: 'cached-user',
        role: 'editor',
      },
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true, can_manage_hosts: true,
      },
    })

    const listener = vi.fn()
    const unsubscribe = subscribeSession(listener)

    localStorage.removeItem('token')
    localStorage.removeItem('auth-gate.session-user')
    localStorage.setItem('auth-gate.session-notice', 'expired')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'token',
      oldValue: 'token-123',
      newValue: null,
    }))

    expect(listener).toHaveBeenCalledTimes(1)
    expect(getSessionToken()).toBeNull()
    expect(getSessionUser()).toBeNull()
    expect(getSessionNotice()).toBe('expired')

    unsubscribe()
  })
})
