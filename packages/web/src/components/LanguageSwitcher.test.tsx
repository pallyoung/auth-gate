import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

  it('remembers the locale when the user picks a language', async () => {
    const user = userEvent.setup()

    await renderWithI18n(<LanguageSwitcher />, { locale: 'en' })

    expect(localStorage.getItem('auth-gate.locale')).toBeNull()

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(localStorage.getItem('auth-gate.locale')).toBe('zh-CN')
    expect(document.documentElement.lang).toBe('zh-CN')
  })
})
