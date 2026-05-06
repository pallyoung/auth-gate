import type { LoginResponse, SessionUser } from './api/types'

const TOKEN_KEY = 'token'

type Listener = () => void

let token = localStorage.getItem(TOKEN_KEY)
let user: SessionUser | null = null
const listeners = new Set<Listener>()

function emitChange() {
  for (const listener of listeners) {
    listener()
  }
}

function persistToken(nextToken: string | null) {
  token = nextToken
  if (nextToken) {
    localStorage.setItem(TOKEN_KEY, nextToken)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export function getSessionToken() {
  return token
}

export function getSessionUser() {
  return user
}

export function setSession(session: LoginResponse) {
  persistToken(session.token)
  user = {
    ...session.user,
    permissions: session.permissions,
  }
  emitChange()
}

export function clearSession() {
  persistToken(null)
  user = null
  emitChange()
}

export function setSessionUser(nextUser: SessionUser | null) {
  user = nextUser
  emitChange()
}

export function subscribeSession(listener: Listener) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}
