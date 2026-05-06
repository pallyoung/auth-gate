import React, { forwardRef } from 'react'
import { cn } from '../../lib/utils'

type InputSize = 'sm' | 'md' | 'lg'

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
  hint?: string
  inputSize?: InputSize
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
}

const sizeStyles: Record<InputSize, string> = {
  sm: 'h-10 md:h-8 text-[var(--text-sm)]',
  md: 'h-12 md:h-10 text-[var(--text-sm)]',
  lg: 'h-14 md:h-12 text-[var(--text-base)]',
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      label,
      error,
      hint,
      inputSize = 'md',
      leftIcon,
      rightIcon,
      className,
      id,
      ...props
    },
    ref
  ) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, '-')
    const errorId = error ? `${inputId}-error` : undefined
    const hintId = hint && !error ? `${inputId}-hint` : undefined

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="text-[var(--text-sm)] font-medium text-[var(--text-primary)]"
          >
            {label}
          </label>
        )}
        <div className="relative">
          {leftIcon && (
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--neutral-400)]" aria-hidden="true">
              {leftIcon}
            </span>
          )}
          <input
            ref={ref}
            id={inputId}
            aria-invalid={!!error}
            aria-describedby={errorId || hintId}
            className={cn(
              'w-full px-3 bg-[var(--bg-card)] border border-[var(--border-default)]',
              'rounded-[var(--radius-md)] text-[var(--text-primary)]',
              'placeholder:text-[var(--neutral-400)]',
              'hover:border-[var(--border-strong)]',
              'focus:border-[var(--primary-500)] focus:ring-2 focus:ring-[var(--primary-500)]/20',
              'focus:outline-none',
              'disabled:opacity-50 disabled:cursor-not-allowed disabled:bg-[var(--neutral-100)]',
              'transition-colors duration-[var(--duration-fast)]',
              'touch-manipulation',
              sizeStyles[inputSize],
              leftIcon && 'pl-9',
              rightIcon && 'pr-9',
              error && 'border-[var(--error)] focus:border-[var(--error)] focus:ring-[var(--error)]/20',
              className
            )}
            {...props}
          />
          {rightIcon && (
            <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--neutral-400)]" aria-hidden="true">
              {rightIcon}
            </span>
          )}
        </div>
        {error && (
          <span id={errorId} role="alert" className="text-[var(--text-xs)] text-[var(--error)]">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={hintId} className="text-[var(--text-xs)] text-[var(--text-muted)]">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
