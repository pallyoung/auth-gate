import React from 'react'
import { cn } from '../../lib/utils'

interface MetricCardProps extends React.HTMLAttributes<HTMLDivElement> {
  label: string
  value: string | number
  hint?: string
  icon?: React.ReactNode
  tone?: 'primary' | 'accent' | 'neutral'
}

const toneStyles = {
  primary: 'from-[rgba(15,143,139,0.18)] to-transparent text-[var(--primary-700)]',
  accent: 'from-[rgba(189,122,24,0.18)] to-transparent text-[var(--accent-600)]',
  neutral: 'from-[rgba(23,33,45,0.08)] to-transparent text-[var(--text-secondary)]',
}

export function MetricCard({
  label,
  value,
  hint,
  icon,
  tone = 'neutral',
  className,
  ...props
}: MetricCardProps) {
  return (
    <div
      className={cn(
        'glass-panel rounded-[24px] p-5',
        'bg-[linear-gradient(180deg,rgba(255,255,255,0.5),rgba(255,255,255,0.14))]',
        className
      )}
      {...props}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">{label}</div>
          <div className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[var(--text-primary)]">{value}</div>
          {hint ? <div className="mt-2 text-sm text-[var(--text-muted)]">{hint}</div> : null}
        </div>
        {icon ? (
          <div className={cn('rounded-[18px] bg-gradient-to-br p-3', toneStyles[tone])}>
            {icon}
          </div>
        ) : null}
      </div>
    </div>
  )
}
