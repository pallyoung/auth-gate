import '@testing-library/jest-dom/vitest'
import { afterEach, vi } from 'vitest'

declare const process: {
  env: Record<string, string | undefined>
}

process.env.NODE_ENV = 'test'

const { cleanup } = await import('@testing-library/react')

const storage = new Map<string, string>()

Object.defineProperty(globalThis, 'localStorage', {
  value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value)
    },
    removeItem: (key: string) => {
      storage.delete(key)
    },
    clear: () => {
      storage.clear()
    },
  },
  configurable: true,
})

Object.defineProperty(globalThis, '__AUTH_GATE_TEST_NAVIGATE__', {
  value: (url: string) => {
    const resolved = new URL(url, window.location.href)
    window.history.pushState({}, '', resolved.toString())
  },
  writable: true,
  configurable: true,
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  try {
    globalThis.localStorage?.clear?.()
  } catch {
    return
  }
})
