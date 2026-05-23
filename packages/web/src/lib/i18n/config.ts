export const SUPPORTED_LOCALES = ['en', 'zh-CN'] as const

export type AppLocale = (typeof SUPPORTED_LOCALES)[number]

export const DEFAULT_LOCALE: AppLocale = 'en'
export const LOCALE_STORAGE_KEY = 'auth-gate.locale'

type StorageLike = Pick<Storage, 'getItem' | 'setItem'>
type NavigatorLike = Pick<Navigator, 'language' | 'languages'>

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

function getSafeStorage(): StorageLike | null {
  const storage = globalThis.localStorage

  if (
    storage &&
    typeof storage.getItem === 'function' &&
    typeof storage.setItem === 'function'
  ) {
    return storage
  }

  return null
}

function getSafeNavigator(): NavigatorLike | null {
  const currentNavigator = globalThis.navigator

  if (currentNavigator && typeof currentNavigator.language === 'string') {
    return currentNavigator
  }

  return null
}

export function getPersistedLocale() {
  return normalizeLocale(getSafeStorage()?.getItem(LOCALE_STORAGE_KEY))
}

export function persistLocale(locale: string) {
  const normalized = normalizeLocale(locale) ?? DEFAULT_LOCALE
  getSafeStorage()?.setItem(LOCALE_STORAGE_KEY, normalized)
}

export function detectInitialLocale() {
  const persisted = getPersistedLocale()
  if (persisted) return persisted

  const currentNavigator = getSafeNavigator()
  const languages = Array.isArray(currentNavigator?.languages) ? currentNavigator.languages : []
  for (const candidate of languages) {
    const normalized = normalizeLocale(candidate)
    if (normalized) return normalized
  }

  return normalizeLocale(currentNavigator?.language) ?? DEFAULT_LOCALE
}
