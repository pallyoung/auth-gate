import React, { forwardRef } from 'react'
import { cn } from '../../lib/utils'
import { ChevronDown } from 'lucide-react'

type SelectSize = 'sm' | 'md' | 'lg'

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
  hint?: string
  selectSize?: SelectSize
  options: { value: string; label: string }[]
}

const sizeStyles: Record<SelectSize, string> = {
  sm: 'h-10 md:h-8 text-[var(--text-sm)]',
  md: 'h-12 md:h-10 text-[var(--text-sm)]',
  lg: 'h-14 md:h-12 text-[var(--text-base)]',
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      label,
      error,
      hint,
      selectSize = 'md',
      options,
      className,
      id,
      ...props
    },
    ref
  ) => {
    const selectId = id || label?.toLowerCase().replace(/\s+/g, '-')
    
    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={selectId}
            className="text-[var(--text-sm)] font-medium text-[var(--text-primary)]"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={selectId}
            aria-invalid={!!error}
            aria-describedby={error ? `${selectId}-error` : hint ? `${selectId}-hint` : undefined}
            className={cn(
              'w-full px-3 pr-9 bg-[var(--bg-card)] border border-[var(--border-default)]',
              'rounded-[var(--radius-md)] text-[var(--text-primary)]',
              'hover:border-[var(--border-strong)]',
              'focus:border-[var(--primary-500)] focus:ring-2 focus:ring-[var(--primary-500)]/20',
              'focus:outline-none',
              'disabled:opacity-50 disabled:cursor-not-allowed',
              'appearance-none cursor-pointer',
              'transition-colors duration-[var(--duration-fast)]',
              'touch-manipulation',
              sizeStyles[selectSize],
              error && 'border-[var(--error)] focus:border-[var(--error)] focus:ring-[var(--error)]/20',
              className
            )}
            {...props}
          >
            {options.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <ChevronDown 
            className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--neutral-400)] pointer-events-none" 
            aria-hidden="true"
          />
        </div>
        {error && (
          <span id={`${selectId}-error`} role="alert" className="text-[var(--text-xs)] text-[var(--error)]">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={`${selectId}-hint`} className="text-[var(--text-xs)] text-[var(--text-muted)]">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Select.displayName = 'Select'
