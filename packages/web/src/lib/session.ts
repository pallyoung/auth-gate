import React from 'react'
import { authApi } from './api/auth'
import type { LoginResponse } from './api/types'
import {
  clearSession,
  clearSessionNotice,
  getSessionNotice,
  getSessionToken,
  getSessionUser,
  setSession,
  setSessionUser,
  subscribeSession,
} from './session-store'

type SessionNoticeState = ReturnType<typeof getSessionNotice> | 'recovery_failed'

export async function refreshSessionUser() {
  const token = getSessionToken()

  if (!token) {
    setSessionUser(null)
    return null
  }

  try {
    const currentUser = await authApi.me()
    if (getSessionToken() === token) {
      setSessionUser(currentUser)
    }
    return currentUser
  } catch (error) {
    throw error
  }
}

export async function login(username: string, password: string) {
  const session = await authApi.login(username, password)
  setSession(session)
  return session
}

export async function logout() {
  try {
    await authApi.logout()
  } finally {
    clearSession()
  }
}

export function useSession() {
  const [state, setState] = React.useState<{
    token: string | null
    user: ReturnType<typeof getSessionUser>
    notice: SessionNoticeState
    loading: boolean
    bootstrapping: boolean
  }>(() => {
    const token = getSessionToken()
    const user = getSessionUser()

    return {
      token,
      user,
      notice: getSessionNotice(),
      loading: Boolean(token && !user),
      bootstrapping: Boolean(token),
    }
  })

  React.useEffect(() => subscribeSession(() => {
    const token = getSessionToken()
    const user = getSessionUser()

    setState((current) => ({
      token,
      user,
      notice: getSessionNotice(),
      loading: Boolean(token && !user),
      bootstrapping: current.bootstrapping && Boolean(token),
    }))
  }), [])

  React.useEffect(() => {
    const token = getSessionToken()
    const user = getSessionUser()

    if (!token) {
      setState({
        token: null,
        user: null,
        notice: getSessionNotice(),
        loading: false,
        bootstrapping: false,
      })
      return
    }

    let cancelled = false
    if (!user) {
      setState((current) => ({ ...current, loading: true }))
    }

    const hadCachedUser = Boolean(user)
    refreshSessionUser()
      .then(() => getSessionNotice() as SessionNoticeState)
      .catch(() => {
        const nextToken = getSessionToken()
        const nextUser = getSessionUser()

        if (nextToken === token && !hadCachedUser && !nextUser) {
          return 'recovery_failed' as const
        }

        return getSessionNotice() as SessionNoticeState
      })
      .then((notice) => {
        if (cancelled) {
          return
        }
        setState({
          token: getSessionToken(),
          user: getSessionUser(),
          notice,
          loading: false,
          bootstrapping: false,
        })
      })

    return () => {
      cancelled = true
    }
  }, [])

  const clearNotice = React.useCallback(() => {
    setState((current) => {
      if (current.notice !== 'recovery_failed') {
        return current
      }

      return {
        ...current,
        notice: null,
      }
    })

    clearSessionNotice()
  }, [])

  return {
    ...state,
    login,
    logout,
    clearNotice,
  }
}
