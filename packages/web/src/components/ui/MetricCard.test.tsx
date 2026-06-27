import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { MetricCard } from './MetricCard'

describe('MetricCard', () => {
  it('supports warning and error tones for icon styling', () => {
    render(
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
})
