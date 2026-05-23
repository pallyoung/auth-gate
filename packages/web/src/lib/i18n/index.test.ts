import { beforeEach, describe, expect, it, vi } from 'vitest'

describe('app i18n runtime', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.resetModules()
  })

  it('prefers the persisted locale over browser language', async () => {
    localStorage.setItem('auth-gate.locale', 'zh-CN')
    vi.stubGlobal('navigator', {
      language: 'en-US',
      languages: ['en-US', 'en'],
    })

    const { createAppI18n } = await import('./index')

    const i18n = await createAppI18n()

    expect(i18n.resolvedLanguage).toBe('zh-CN')
  })

  it('maps browser zh variants to zh-CN when nothing is persisted', async () => {
    vi.stubGlobal('navigator', {
      language: 'zh-TW',
      languages: ['zh-TW', 'en-US'],
    })

    const { createAppI18n } = await import('./index')

    const i18n = await createAppI18n()

    expect(i18n.resolvedLanguage).toBe('zh-CN')
  })

  it('falls back to en when persisted and browser locales are unsupported', async () => {
    localStorage.setItem('auth-gate.locale', 'fr-FR')
    vi.stubGlobal('navigator', {
      language: 'fr-FR',
      languages: ['fr-FR'],
    })

    const { createAppI18n } = await import('./index')

    const i18n = await createAppI18n()

    expect(i18n.resolvedLanguage).toBe('en')
  })

  it('persists manual language changes', async () => {
    vi.stubGlobal('navigator', {
      language: 'en-US',
      languages: ['en-US'],
    })

    const { createAppI18n } = await import('./index')

    const i18n = await createAppI18n()
    await i18n.changeLanguage('zh-CN')

    expect(localStorage.getItem('auth-gate.locale')).toBe('zh-CN')
  })
})
