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
})
