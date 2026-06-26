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
  sm: 'h-10 text-sm',
  md: 'h-12 text-sm',
  lg: 'h-14 text-base',
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
      <div className="flex flex-col gap-2">
        {label && (
          <label
            htmlFor={inputId}
            className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]"
          >
            {label}
          </label>
        )}
        <div className="relative">
          {leftIcon && (
            <span
              className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-[var(--text-subtle)]"
              aria-hidden="true"
            >
              {leftIcon}
            </span>
          )}
          <input
            ref={ref}
            id={inputId}
            aria-invalid={!!error}
            aria-describedby={errorId || hintId}
            className={cn(
              'w-full rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)]',
              'px-4 text-[var(--text-primary)]',
              'placeholder:text-[var(--text-subtle)] transition-all duration-[var(--duration-normal)]',
              'hover:border-[var(--border-strong)]',
              'focus:border-[var(--primary-500)] focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]',
              'disabled:cursor-not-allowed disabled:opacity-55',
              sizeStyles[inputSize],
              leftIcon && 'pl-11',
              rightIcon && 'pr-11',
              error && 'border-[rgba(248,113,113,0.3)] text-[var(--error)]',
              className
            )}
            {...props}
          />
          {rightIcon && (
            <span
              className="pointer-events-none absolute right-4 top-1/2 -translate-y-1/2 text-[var(--text-subtle)]"
              aria-hidden="true"
            >
              {rightIcon}
            </span>
          )}
        </div>
        {error && (
          <span id={errorId} role="alert" className="text-xs font-medium text-[var(--error)]">
            {error}
          </span>
        )}
        {hint && !error && (
          <span id={hintId} className="text-xs text-[var(--text-muted)]">
            {hint}
          </span>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
