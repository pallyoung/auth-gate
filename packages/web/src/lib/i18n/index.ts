import i18next, { type i18n as I18nInstance } from 'i18next'
import { initReactI18next } from 'react-i18next'
import {
  applyDocumentLocale,
  DEFAULT_LOCALE,
  detectInitialLocale,
  persistLocale,
  SUPPORTED_LOCALES,
  type AppLocale,
} from './config'
import { resources } from './resources'

export async function createAppI18n(initialLocale?: AppLocale): Promise<I18nInstance> {
  const i18n = i18next.createInstance()
  const resolvedLocale = initialLocale ?? detectInitialLocale()

  await i18n.use(initReactI18next).init({
    resources,
    lng: resolvedLocale,
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: [...SUPPORTED_LOCALES],
    defaultNS: 'common',
    interpolation: {
      escapeValue: false,
    },
  })

  applyDocumentLocale(i18n.resolvedLanguage ?? resolvedLocale)

  // Keep <html lang> in sync. Do not persist here — only manual switcher writes preference.
  i18n.on('languageChanged', (nextLanguage) => {
    applyDocumentLocale(nextLanguage)
  })

  return i18n
}

/** Change language and remember it as the user's manual preference. */
export async function changeAppLocale(i18n: I18nInstance, locale: AppLocale) {
  await i18n.changeLanguage(locale)
  persistLocale(locale)
}

let i18nPromiseCache: Promise<I18nInstance> | null = null

function getI18nPromise() {
  i18nPromiseCache ??= createAppI18n()
  return i18nPromiseCache
}

export const i18nPromise = {
  then(onfulfilled, onrejected) {
    return getI18nPromise().then(onfulfilled, onrejected)
  },
  catch(onrejected) {
    return getI18nPromise().catch(onrejected)
  },
  finally(onfinally) {
    return getI18nPromise().finally(onfinally)
  },
  [Symbol.toStringTag]: 'Promise',
} as Promise<I18nInstance>
