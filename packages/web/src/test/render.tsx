import { render, type RenderOptions } from '@testing-library/react'
import type { ReactElement } from 'react'
import { I18nextProvider } from 'react-i18next'
import { createAppI18n } from '../lib/i18n'
import type { AppLocale } from '../lib/i18n/config'

export async function renderWithI18n(
  ui: ReactElement,
  options?: RenderOptions & { locale?: AppLocale }
) {
  const { locale, ...renderOptions } = options ?? {}
  const i18n = await createAppI18n(locale)
  const renderResult = render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>, renderOptions)
  return Object.assign(renderResult, { i18n })
}
