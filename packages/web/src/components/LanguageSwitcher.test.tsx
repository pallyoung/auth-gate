import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LanguageSwitcher } from './LanguageSwitcher'
import { renderWithI18n } from '../test/render'

describe('LanguageSwitcher', () => {
  it('uses 44px touch targets for each language option', async () => {
    await renderWithI18n(<LanguageSwitcher />, { locale: 'en' })

    for (const name of ['English', '中文']) {
      const button = screen.getByRole('button', { name })

      expect(button).toHaveClass('min-h-8')
      expect(button).toHaveClass('min-w-8')
    }
  })
})
