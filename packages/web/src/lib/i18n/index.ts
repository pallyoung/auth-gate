import i18next, { type i18n as I18nInstance } from 'i18next'
import { initReactI18next } from 'react-i18next'
import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  detectInitialLocale,
  normalizeLocale,
  type AppLocale,
} from './config'
import { resources } from './resources'

export async function createAppI18n(initialLocale?: AppLocale): Promise<I18nInstance> {
  const i18n = i18next.createInstance()

  await i18n.use(initReactI18next).init({
    resources,
    lng: initialLocale ?? detectInitialLocale(),
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: ['en', 'zh-CN'],
    defaultNS: 'common',
    interpolation: {
      escapeValue: false,
    },
  })

  i18n.on('languageChanged', (nextLanguage) => {
    const normalized = normalizeLocale(nextLanguage) ?? DEFAULT_LOCALE
    localStorage.setItem(LOCALE_STORAGE_KEY, normalized)
  })

  return i18n
}

export const i18nPromise = createAppI18n()
