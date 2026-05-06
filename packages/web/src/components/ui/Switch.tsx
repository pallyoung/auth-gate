import React from 'react'
import { cn } from '../../lib/utils'

type SwitchSize = 'sm' | 'md'

interface SwitchProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  label?: string
  switchSize?: SwitchSize
}

const sizeStyles: Record<SwitchSize, { wrapper: string; knob: string }> = {
  sm: {
    wrapper: 'w-10 h-6 md:w-9 md:h-5',
    knob: 'w-5 h-5 md:w-4 md:h-4 data-[state=checked]:translate-x-5 md:data-[state=checked]:translate-x-4',
  },
  md: {
    wrapper: 'w-12 h-7 md:w-11 md:h-6',
    knob: 'w-6 h-6 md:w-5 md:h-5 data-[state=checked]:translate-x-6 md:data-[state=checked]:translate-x-5',
  },
}

export function Switch({
  label,
  switchSize = 'md',
  checked,
  onChange,
  className,
  id,
  ...props
}: SwitchProps) {
  const switchId = id || label?.toLowerCase().replace(/\s+/g, '-')

  return (
    <label htmlFor={switchId} className="inline-flex items-center gap-2 cursor-pointer">
      <div className="relative">
        <input
          type="checkbox"
          id={switchId}
          checked={checked}
          onChange={onChange}
          className="sr-only peer"
          {...props}
        />
        <div
          className={cn(
            'rounded-full transition-colors duration-[var(--duration-normal)]',
            'bg-[var(--neutral-300)] peer-checked:bg-[var(--primary-500)]',
            'peer-focus:ring-2 peer-focus:ring-[var(--primary-500)]/50',
            'peer-disabled:opacity-50 peer-disabled:cursor-not-allowed',
            'touch-manipulation',
            sizeStyles[switchSize].wrapper,
            className
          )}
        />
        <div
          data-state={checked ? 'checked' : 'unchecked'}
          className={cn(
            'absolute top-0.5 left-0.5 rounded-full bg-white',
            'transition-transform duration-[var(--duration-normal)]',
            'shadow-sm',
            sizeStyles[switchSize].knob
          )}
        />
      </div>
      {label && (
        <span className="text-[var(--text-sm)] text-[var(--text-primary)]">{label}</span>
      )}
    </label>
  )
}
