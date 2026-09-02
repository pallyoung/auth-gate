import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  detectBrowserLocale,
  detectInitialLocale,
  normalizeLocale,
  persistLocale,
} from './config'

describe('i18n config', () => {
  beforeEach(() => {
    globalThis.localStorage?.clear?.()
  })

  it('normalizes common browser locale tags', () => {
    expect(normalizeLocale('en-US')).toBe('en')
    expect(normalizeLocale('zh')).toBe('zh-CN')
    expect(normalizeLocale('zh-Hans-CN')).toBe('zh-CN')
    expect(normalizeLocale('fr-FR')).toBeNull()
  })

  it('detects browser language without writing localStorage', () => {
    vi.stubGlobal('navigator', {
      language: 'zh-CN',
      languages: ['zh-CN', 'en'],
    })

    expect(detectBrowserLocale()).toBe('zh-CN')
    expect(localStorage.getItem('auth-gate.locale')).toBeNull()
  })

  it('prefers a saved manual preference over the browser language', () => {
    persistLocale('en')
    vi.stubGlobal('navigator', {
      language: 'zh-CN',
      languages: ['zh-CN'],
    })

    expect(detectInitialLocale()).toBe('en')
  })
})
