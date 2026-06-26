import { clearSession, getSessionToken } from '../session-store'
import type { ApiErrorEnvelope } from './types'

const ensureArray = <T>(value: unknown): T[] => Array.isArray(value) ? value as T[] : []

// API base path — defaults to /api (dual-engine mode).
// In single-engine compatibility mode, initApiBase() detects the /_authgate prefix.
let controlPlaneAPIBasePath = '/api'

/**
 * Detect the API base path from the backend.
 * Tries /api/config/app first (dual-engine), then /_authgate/api/config/app
 * (single-engine compatibility). If both fail the default /api is kept.
 */
export async function initApiBase(): Promise<void> {
  if (typeof process !== 'undefined' && process.env?.NODE_ENV === 'test') {
    // Tests use single-engine mock URLs (/_authgate/api/*).
    controlPlaneAPIBasePath = '/_authgate/api'
    return
  }
  for (const base of ['/api', '/_authgate/api']) {
    try {
      const res = await fetch(`${base}/config/app`)
      if (res.ok) {
        const data = await res.json()
        if (data.api_base) {
          controlPlaneAPIBasePath = data.api_base
          return
        }
      }
    } catch {
      // try next
    }
  }
}

export function getApiBasePath(): string {
  return controlPlaneAPIBasePath
}

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
