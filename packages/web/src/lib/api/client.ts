import { clearSession, getSessionToken } from '../session-store'
import type { ApiErrorEnvelope } from './types'

const ensureArray = <T>(value: unknown): T[] => Array.isArray(value) ? value as T[] : []
const controlPlaneAPIBasePath = '/_authgate/api'

export class ApiError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface RequestOptions extends RequestInit {
  authMode?: 'control-plane' | 'none'
}

async function doRequest<T>(path: string, options?: RequestOptions): Promise<T> {
  const authMode = options?.authMode ?? 'control-plane'
  const token = authMode === 'control-plane' ? getSessionToken() : null
  const headers = new Headers(options?.headers)

  if (!headers.has('Content-Type') && options?.body) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(`${controlPlaneAPIBasePath}${path}`, { ...options, headers })
  if (res.status === 401 && authMode === 'control-plane' && token && getSessionToken() === token) {
    clearSession('expired')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: { message: 'Unknown error' } })) as ApiErrorEnvelope
    const message = err.error?.message || `HTTP ${res.status}`
    throw new ApiError(message, res.status, err.error?.code)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json()
}

export async function request<T>(path: string, options?: RequestInit): Promise<T> {
  return doRequest<T>(path, { ...options, authMode: 'control-plane' })
}

export async function publicRequest<T>(path: string, options?: RequestInit): Promise<T> {
  return doRequest<T>(path, { ...options, authMode: 'none' })
}

export async function listResource<T>(path: string): Promise<T[]> {
  return ensureArray<T>(await request<unknown>(path))
}
