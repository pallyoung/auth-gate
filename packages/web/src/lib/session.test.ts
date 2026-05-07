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

    expect(fetchMock).toHaveBeenCalledWith('/_authgate/api/auth/me', expect.objectContaining({
      headers: expect.any(Headers),
    }))
    expect(user).toEqual({
      id: 'user-1',
      username: 'fresh-user',
      role: 'editor',
    })
    expect(getSessionUser()).toEqual(user)
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
})
