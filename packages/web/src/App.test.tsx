import { act, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithI18n } from './test/render'

describe('App', () => {
  beforeEach(() => {
    localStorage.clear()
    window.location.hash = '/'
    vi.resetModules()
  })

  it('shows a session expired message on the login page when bootstrap auth refresh returns invalid_token', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: {
          code: 'invalid_token',
          message: 'invalid token',
        },
      }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }))

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Sign in to the control plane' })).toBeInTheDocument()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Your session has expired. Sign in again to continue.'
    )
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('shows a session recovery message when bootstrap auth refresh fails without a cached user', async () => {
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

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Sign in to the control plane' })).toBeInTheDocument()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't start your control plane session right now. Try again in a moment."
    )
    expect(localStorage.getItem('token')).toBe('token-123')
  })

  it('keeps the last known control-plane session visible when bootstrap auth refresh fails with a non-auth error', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-user',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Response(JSON.stringify({
          error: {
            code: 'session_store_failure',
            message: 'failed to load user',
          },
        }), {
          status: 500,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      if (url === '/_authgate/api/routes') {
        return new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Routes' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Sign in to the control plane' })).not.toBeInTheDocument()
    expect(localStorage.getItem('token')).toBe('token-123')
  })

  it('keeps the last known control-plane session visible while bootstrap auth refresh is still pending', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-user',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Promise<Response>(() => {
          // Keep the bootstrap refresh pending to verify cached sessions stay usable.
        })
      }

      if (url === '/_authgate/api/routes') {
        return Promise.resolve(new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }))
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Routes' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/_authgate/api/routes', expect.anything())
    expect(screen.queryByRole('heading', { name: 'Sign in to the control plane' })).not.toBeInTheDocument()
  })

  it('redirects direct users-page visits back to routes when the session cannot manage users', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-editor',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))
    window.location.hash = '/users'

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Response(JSON.stringify({
          id: 'user-1',
          username: 'cached-editor',
          role: 'editor',
          permissions: {
            can_manage_routes: true,
            can_manage_auth: true,
            can_manage_users: false,
            can_view_logs: true,
          },
          features: {
            certificates: false,
          },
        }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      if (url === '/_authgate/api/routes') {
        return new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Routes' })).toBeInTheDocument()
    expect(window.location.hash).toBe('#/')
    expect(screen.queryByText('User management is unavailable for your account.')).not.toBeInTheDocument()
  })

  it('keeps direct users-page visits on /users while bootstrap refresh upgrades cached permissions', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-editor',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))
    window.location.hash = '/users'

    let resolveAuthMe: ((response: Response) => void) | undefined

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Promise<Response>((resolve) => {
          resolveAuthMe = resolve
        })
      }

      if (url === '/_authgate/api/users') {
        return Promise.resolve(new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }))
      }

      if (url === '/_authgate/api/routes') {
        return Promise.resolve(new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }))
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(window.location.hash).toBe('#/users')

    await act(async () => {
      resolveAuthMe?.(new Response(JSON.stringify({
        id: 'user-1',
        username: 'fresh-admin',
        role: 'admin',
        permissions: {
          can_manage_routes: true,
          can_manage_auth: true,
          can_manage_users: true,
          can_view_logs: true,
        },
        features: {
          certificates: false,
        },
      }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }))
    })

    expect(await screen.findByRole('button', { name: 'Add User' })).toBeInTheDocument()
    expect(window.location.hash).toBe('#/users')
  })

  it('normalizes unknown control-plane routes back to routes', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-editor',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))
    window.location.hash = '/does-not-exist'

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Response(JSON.stringify({
          id: 'user-1',
          username: 'cached-editor',
          role: 'editor',
          permissions: {
            can_manage_routes: true,
            can_manage_auth: true,
            can_manage_users: false,
            can_view_logs: true,
          },
          features: {
            certificates: false,
          },
        }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      if (url === '/_authgate/api/routes') {
        return new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Routes' })).toBeInTheDocument()
    expect(window.location.hash).toBe('#/')
  })

  it('returns to the login page when another tab clears the active session', async () => {
    localStorage.setItem('token', 'token-123')
    localStorage.setItem('auth-gate.session-user', JSON.stringify({
      id: 'user-1',
      username: 'cached-user',
      role: 'editor',
      permissions: {
        can_manage_routes: true,
        can_manage_auth: true,
        can_manage_users: false,
        can_view_logs: true,
      },
      features: {
        certificates: false,
      },
    }))

    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Response(JSON.stringify({
          id: 'user-1',
          username: 'cached-user',
          role: 'editor',
          permissions: {
            can_manage_routes: true,
            can_manage_auth: true,
            can_manage_users: false,
            can_view_logs: true,
          },
          features: {
            certificates: false,
          },
        }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      if (url === '/_authgate/api/routes') {
        return new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(await screen.findByRole('heading', { name: 'Routes' })).toBeInTheDocument()

    localStorage.removeItem('token')
    localStorage.removeItem('auth-gate.session-user')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'token',
      oldValue: 'token-123',
      newValue: null,
    }))

    expect(await screen.findByRole('heading', { name: 'Sign in to the control plane' })).toBeInTheDocument()
  })

  it('returns to the login page immediately when another tab clears the session during bootstrap refresh', async () => {
    localStorage.setItem('token', 'token-123')

    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input.toString()

      if (url === '/_authgate/api/auth/me') {
        return new Promise<Response>(() => {
          // Keep bootstrap refresh pending to verify storage sync can still leave loading state.
        })
      }

      throw new Error(`Unhandled fetch request: ${url}`)
    })

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    localStorage.removeItem('token')
    localStorage.removeItem('auth-gate.session-user')
    localStorage.setItem('auth-gate.session-notice', 'expired')
    window.dispatchEvent(new StorageEvent('storage', {
      key: 'token',
      oldValue: 'token-123',
      newValue: null,
    }))

    expect(await screen.findByRole('heading', { name: 'Sign in to the control plane' })).toBeInTheDocument()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Your session has expired. Sign in again to continue.'
    )
  })

  it('renders the route access login immediately even when control-plane session bootstrap is still pending', async () => {
    localStorage.setItem('token', 'token-123')
    window.location.hash = '/access-login?route_id=route-1&route_name=Protected%20App&path_prefix=%2Fprotected'

    const fetchMock = vi.fn(
      () =>
        new Promise<Response>(() => {
          // Keep /auth/me pending to verify the access-login route is not blocked by control-plane bootstrap.
        })
    )

    vi.stubGlobal('fetch', fetchMock)

    const { default: App } = await import('./App')

    await renderWithI18n(<App />)

    expect(screen.getByRole('heading', { name: 'Sign in to continue' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue to Route' })).toBeInTheDocument()
  })
})
