import React from 'react'
import { cn } from '../../lib/utils'

type SwitchSize = 'sm' | 'md'

interface SwitchProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label?: string
  description?: string
  switchSize?: SwitchSize
}

const sizeStyles: Record<SwitchSize, { wrapper: string; knob: string }> = {
  sm: {
    wrapper: 'h-6 w-10',
    knob: 'h-4 w-4 peer-checked:translate-x-4',
  },
  md: {
    wrapper: 'h-7 w-12',
    knob: 'h-5 w-5 peer-checked:translate-x-5',
  },
}

export function Switch({
  label,
  description,
  switchSize = 'md',
  checked,
  onChange,
  className,
  id,
  ...props
}: SwitchProps) {
  const switchId = id || label?.toLowerCase().replace(/\s+/g, '-')
  const labelId = label ? `${switchId}-label` : undefined
  const descriptionId = description ? `${switchId}-description` : undefined
  const isStandalone = !label && !description

  return (
    <label
      htmlFor={switchId}
      className={cn(
        isStandalone
          ? 'inline-flex items-center cursor-pointer'
          : 'flex items-center justify-between gap-4 rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-3 shadow-[var(--shadow-sm)] backdrop-blur-xl'
      )}
    >
      <div className="min-w-0">
        {label && <div id={labelId} className="text-sm font-semibold text-[var(--text-primary)]">{label}</div>}
        {description && <div id={descriptionId} className="mt-1 text-xs text-[var(--text-muted)]">{description}</div>}
      </div>
      <div className="relative shrink-0">
        <input
          type="checkbox"
          id={switchId}
          aria-labelledby={labelId}
          aria-describedby={descriptionId}
          checked={checked}
          onChange={onChange}
          className="peer sr-only"
          {...props}
        />
        <div
          className={cn(
            'rounded-full border border-[var(--border-default)] bg-[var(--neutral-300)] transition-all duration-[var(--duration-normal)]',
            'peer-checked:bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))]',
            'peer-disabled:opacity-50',
            sizeStyles[switchSize].wrapper,
            className
          )}
        />
        <div
          className={cn(
            'absolute left-1 top-1/2 -translate-y-1/2 rounded-full bg-white shadow-[var(--shadow-sm)] transition-transform duration-[var(--duration-normal)]',
            sizeStyles[switchSize].knob
          )}
        />
      </div>
    </label>
  )
}
