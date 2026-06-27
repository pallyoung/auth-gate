import '@testing-library/jest-dom/vitest'
import { afterEach, vi } from 'vitest'

declare const process: {
  env: Record<string, string | undefined>
}

process.env.NODE_ENV = 'test'

Object.defineProperty(globalThis, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

// jsdom does not implement scrollIntoView; the custom Select component calls it
// on the active option element when the dropdown opens. Provide a no-op so the
// component does not throw in tests.
if (typeof Element !== 'undefined' && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = vi.fn()
}

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
