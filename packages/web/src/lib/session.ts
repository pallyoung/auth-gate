import React from 'react'
import { authApi } from './api/auth'
import type { LoginResponse } from './api/types'
import {
  clearSession,
  getSessionToken,
  getSessionUser,
  setSession,
  setSessionUser,
  subscribeSession,
} from './session-store'

export async function refreshSessionUser() {
  if (!getSessionToken()) {
    setSessionUser(null)
    return null
  }

  try {
    const currentUser = await authApi.me()
    setSessionUser(currentUser)
    return currentUser
  } catch (error) {
    clearSession()
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
  const [state, setState] = React.useState(() => ({
    token: getSessionToken(),
    user: getSessionUser(),
    loading: Boolean(getSessionToken()),
  }))

  React.useEffect(() => subscribeSession(() => {
    setState((current) => ({
      ...current,
      token: getSessionToken(),
      user: getSessionUser(),
    }))
  }), [])

  React.useEffect(() => {
    if (!getSessionToken()) {
      setState({ token: null, user: null, loading: false })
      return
    }

    let cancelled = false
    setState((current) => ({ ...current, loading: true }))

    refreshSessionUser()
      .catch(() => null)
      .finally(() => {
        if (cancelled) {
          return
        }
        setState({
          token: getSessionToken(),
          user: getSessionUser(),
          loading: false,
        })
      })

    return () => {
      cancelled = true
    }
  }, [])

  return {
    ...state,
    login,
    logout,
  }
}
