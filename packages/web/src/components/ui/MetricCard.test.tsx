import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MetricCard } from './MetricCard'
import { renderWithI18n } from '../../test/render'

describe('MetricCard', () => {
  it('supports warning and error tones for icon styling', async () => {
    await renderWithI18n(
      <div>
        <MetricCard
          label="Pending"
          value={3}
          tone="warning"
          icon={<span data-testid="warning-icon">W</span>}
        />
        <MetricCard
          label="Failed"
          value={1}
          tone="error"
          icon={<span data-testid="error-icon">E</span>}
        />
      </div>
    )

    expect(screen.getByTestId('warning-icon').parentElement).toHaveClass(
      'bg-[var(--bg-hover)]',
      'text-[var(--warning)]'
    )
    expect(screen.getByTestId('error-icon').parentElement).toHaveClass(
      'bg-[var(--bg-hover)]',
      'text-[var(--error)]'
    )
  })

  it.each(['en', 'zh-CN'] as const)('formats numeric metric values in %s', async (locale) => {
    await renderWithI18n(<MetricCard label="Count" value={12500} />, { locale })
    expect(screen.getByText(new Intl.NumberFormat(locale).format(12500))).toBeInTheDocument()
  })
})
