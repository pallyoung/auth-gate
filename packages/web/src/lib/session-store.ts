import type { LoginResponse, SessionUser } from './api/types'

const TOKEN_KEY = 'token'
const SESSION_USER_KEY = 'auth-gate.session-user'
const SESSION_NOTICE_KEY = 'auth-gate.session-notice'
type SessionNotice = 'expired' | null

type Listener = () => void

let token = localStorage.getItem(TOKEN_KEY)
let user: SessionUser | null = readStoredUser()
let notice: SessionNotice = readStoredNotice()
const listeners = new Set<Listener>()
let storageSyncAttached = false

function readStoredNotice(): SessionNotice {
  const storedNotice = localStorage.getItem(SESSION_NOTICE_KEY)
  if (storedNotice === 'expired') {
    return 'expired'
  }
  if (storedNotice) {
    localStorage.removeItem(SESSION_NOTICE_KEY)
  }
  return null
}

function readStoredUser() {
  const storedUser = localStorage.getItem(SESSION_USER_KEY)
  if (!storedUser) {
    return null
  }

  try {
    return JSON.parse(storedUser) as SessionUser
  } catch {
    localStorage.removeItem(SESSION_USER_KEY)
    return null
  }
}

function emitChange() {
  for (const listener of listeners) {
    listener()
  }
}

function syncStateFromStorage() {
  token = localStorage.getItem(TOKEN_KEY)
  user = readStoredUser()
  notice = readStoredNotice()
}

function userSnapshot(value: SessionUser | null) {
  return value ? JSON.stringify(value) : ''
}

function handleStorageChange(event: StorageEvent) {
  if (
    event.key &&
    event.key !== TOKEN_KEY &&
    event.key !== SESSION_USER_KEY &&
    event.key !== SESSION_NOTICE_KEY
  ) {
    return
  }

  const previousToken = token
  const previousUserSnapshot = userSnapshot(user)
  const previousNotice = notice

  syncStateFromStorage()

  if (
    previousToken === token &&
    previousUserSnapshot === userSnapshot(user) &&
    previousNotice === notice
  ) {
    return
  }

  emitChange()
}

function persistToken(nextToken: string | null) {
  token = nextToken
  if (nextToken) {
    localStorage.setItem(TOKEN_KEY, nextToken)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

function persistUser(nextUser: SessionUser | null) {
  user = nextUser
  if (nextUser) {
    localStorage.setItem(SESSION_USER_KEY, JSON.stringify(nextUser))
  } else {
    localStorage.removeItem(SESSION_USER_KEY)
  }
}

function persistNotice(nextNotice: SessionNotice) {
  notice = nextNotice
  if (nextNotice) {
    localStorage.setItem(SESSION_NOTICE_KEY, nextNotice)
  } else {
    localStorage.removeItem(SESSION_NOTICE_KEY)
  }
}

export function getSessionToken() {
  token = localStorage.getItem(TOKEN_KEY)
  return token
}

export function getSessionUser() {
  user = readStoredUser()
  return user
}

export function getSessionNotice() {
  notice = readStoredNotice()
  return notice
}

export function setSession(session: LoginResponse) {
  persistToken(session.token)
  persistUser({
    ...session.user,
    permissions: session.permissions,
  })
  persistNotice(null)
  emitChange()
}

export function clearSession(nextNotice: SessionNotice = null) {
  persistToken(null)
  persistUser(null)
  persistNotice(nextNotice)
  emitChange()
}

export function setSessionUser(nextUser: SessionUser | null) {
  persistUser(nextUser)
  emitChange()
}

export function clearSessionNotice() {
  if (!notice) {
    return
  }
  persistNotice(null)
  emitChange()
}

export function subscribeSession(listener: Listener) {
  listeners.add(listener)
  if (!storageSyncAttached) {
    window.addEventListener('storage', handleStorageChange)
    storageSyncAttached = true
  }
  return () => {
    listeners.delete(listener)
    if (storageSyncAttached && listeners.size === 0) {
      window.removeEventListener('storage', handleStorageChange)
      storageSyncAttached = false
    }
  }
}
