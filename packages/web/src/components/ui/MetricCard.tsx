import React from 'react'
import { cn } from '../../lib/utils'

type MetricCardTone = 'primary' | 'accent' | 'neutral' | 'warning' | 'error'

interface MetricCardProps extends React.HTMLAttributes<HTMLDivElement> {
  label: string
  value: string | number
  hint?: string
  trend?: string
  trendUp?: boolean
  icon?: React.ReactNode
  tone?: MetricCardTone
}

const toneStyles: Record<MetricCardTone, string> = {
  primary: 'text-[var(--primary-600)]',
  accent: 'text-[var(--accent-600)]',
  neutral: 'text-[var(--text-secondary)]',
  warning: 'text-[var(--warning)]',
  error: 'text-[var(--error)]',
}

export function MetricCard({
  label,
  value,
  hint,
  trend,
  trendUp,
  icon,
  tone = 'neutral',
  className,
  ...props
}: MetricCardProps) {
  return (
    <div
      className={cn(
        'rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-card)] p-4',
        className
      )}
      {...props}
    >
      <div className="flex items-start justify-between">
        <div className="min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-[0.1em] text-[var(--text-muted)]">{label}</div>
          <div className="mt-2 text-2xl font-bold tracking-[-0.02em] text-[var(--text-primary)]">{value}</div>
          {hint && (
            <div className="mt-1.5 text-xs text-[var(--text-muted)]">{hint}</div>
          )}
          {trend && (
            <div className={cn(
              'mt-2 inline-flex items-center gap-1 text-[11px] font-medium',
              trendUp ? 'text-[var(--success)]' : 'text-[var(--error)]'
            )}>
              <span>{trendUp ? '↑' : '↓'}</span>
              <span>{trend}</span>
            </div>
          )}
        </div>
        {icon && (
          <div className={cn('flex h-8 w-8 items-center justify-center rounded-[8px] bg-[var(--bg-hover)]', toneStyles[tone])}>
            {icon}
          </div>
        )}
      </div>
    </div>
  )
}
