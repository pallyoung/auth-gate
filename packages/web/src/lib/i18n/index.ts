import i18next, { type i18n as I18nInstance } from 'i18next'
import { initReactI18next } from 'react-i18next'
import {
  DEFAULT_LOCALE,
  detectInitialLocale,
  persistLocale,
  SUPPORTED_LOCALES,
  type AppLocale,
} from './config'
import { resources } from './resources'

export async function createAppI18n(initialLocale?: AppLocale): Promise<I18nInstance> {
  const i18n = i18next.createInstance()

  await i18n.use(initReactI18next).init({
    resources,
    lng: initialLocale ?? detectInitialLocale(),
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: [...SUPPORTED_LOCALES],
    defaultNS: 'common',
    interpolation: {
      escapeValue: false,
    },
  })

  i18n.on('languageChanged', (nextLanguage) => {
    persistLocale(nextLanguage)
  })

  return i18n
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
