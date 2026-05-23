export const SUPPORTED_LOCALES = ['en', 'zh-CN'] as const

export type AppLocale = (typeof SUPPORTED_LOCALES)[number]

export const DEFAULT_LOCALE: AppLocale = 'en'
export const LOCALE_STORAGE_KEY = 'auth-gate.locale'

export function normalizeLocale(raw: string | null | undefined): AppLocale | null {
  if (!raw) return null

  const lower = raw.toLowerCase()

  if (lower === 'en' || lower.startsWith('en-')) {
    return 'en'
  }

  if (lower === 'zh' || lower === 'zh-cn' || lower.startsWith('zh-')) {
    return 'zh-CN'
  }

  return null
}

export function detectInitialLocale() {
  const persisted = normalizeLocale(localStorage.getItem(LOCALE_STORAGE_KEY))
  if (persisted) return persisted

  const languages = Array.isArray(navigator.languages) ? navigator.languages : []
  for (const candidate of languages) {
    const normalized = normalizeLocale(candidate)
    if (normalized) return normalized
  }

  return normalizeLocale(navigator.language) ?? DEFAULT_LOCALE
}
