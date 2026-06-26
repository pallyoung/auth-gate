import React from 'react'
import { cn } from '../../lib/utils'

interface ProgressBarProps {
  value: number
  label: string
  detail?: string
  className?: string
}

function getBarColor(value: number): string {
  if (value >= 85) return 'bg-[var(--error)]'
  if (value >= 60) return 'bg-[var(--warning)]'
  return 'bg-[var(--success)]'
}

function getTrackColor(value: number): string {
  if (value >= 85) return 'bg-[var(--error-light)]'
  if (value >= 60) return 'bg-[var(--warning-light)]'
  return 'bg-[var(--success-light)]'
}

export function ProgressBar({ value, label, detail, className }: ProgressBarProps) {
  const clamped = Math.min(100, Math.max(0, value))

  return (
    <div className={cn('space-y-2', className)}>
      <div className="flex items-baseline justify-between text-sm">
        <span className="font-medium text-[var(--text-primary)]">{label}</span>
        <span className="tabular-nums text-[var(--text-muted)]">
          {detail ?? `${clamped.toFixed(1)}%`}
        </span>
      </div>
      <div className={cn('h-2.5 w-full overflow-hidden rounded-full', getTrackColor(clamped))}>
        <div
          className={cn('h-full rounded-full transition-all duration-500', getBarColor(clamped))}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  )
}
