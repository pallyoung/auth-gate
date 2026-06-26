import React, { forwardRef } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '../../lib/utils'

type SelectSize = 'sm' | 'md' | 'lg'

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
  error?: string
  hint?: string
  selectSize?: SelectSize
  options: { value: string; label: string }[]
}

const sizeStyles: Record<SelectSize, string> = {
  sm: 'h-10 text-sm',
  md: 'h-12 text-sm',
  lg: 'h-14 text-base',
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
      <div className="flex flex-col gap-2">
        {label && (
          <label
            htmlFor={selectId}
            className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]"
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
              'w-full appearance-none rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)]',
              'px-4 pr-11 text-[var(--text-primary)]',
              'transition-all duration-[var(--duration-normal)]',
              'hover:border-[var(--border-strong)]',
              'focus:border-[var(--primary-500)] focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]',
              'disabled:cursor-not-allowed disabled:opacity-55',
              sizeStyles[selectSize],
              error && 'border-[rgba(248,113,113,0.3)] text-[var(--error)]',
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
            className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-subtle)]"
            aria-hidden="true"
          />
        </div>
        {error && (
          <span id={`${selectId}-error`} role="alert" className="text-xs font-medium text-[var(--error)]">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={`${selectId}-hint`} className="text-xs text-[var(--text-muted)]">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Select.displayName = 'Select'
