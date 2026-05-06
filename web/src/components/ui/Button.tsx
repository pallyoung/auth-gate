import React from 'react'
import { cn } from '../../lib/utils'

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
type ButtonSize = 'sm' | 'md' | 'lg'

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  icon?: React.ReactNode
  iconPosition?: 'left' | 'right'
}

const variantStyles: Record<ButtonVariant, string> = {
  primary: 'bg-[var(--primary-500)] text-white hover:bg-[var(--primary-600)] active:bg-[var(--primary-700)]',
  secondary: 'border border-[var(--primary-500)] text-[var(--primary-500)] hover:bg-[var(--primary-50)] active:bg-[var(--primary-100)]',
  ghost: 'text-[var(--primary-500)] hover:bg-[var(--primary-50)] active:bg-[var(--primary-100)]',
  danger: 'bg-[var(--error)] text-white hover:opacity-90 active:opacity-80',
}

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'h-10 md:h-8 px-3 text-[var(--text-sm)] gap-1.5',
  md: 'h-12 md:h-10 px-4 text-[var(--text-sm)] gap-2',
  lg: 'h-14 md:h-12 px-6 text-[var(--text-base)] gap-2',
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  icon,
  iconPosition = 'left',
  className,
  disabled,
  children,
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading

  return (
    <button
      className={cn(
        'inline-flex items-center justify-center font-medium rounded-[var(--radius-md)] transition-all',
        'duration-[var(--duration-fast)] ease-[var(--ease-out)]',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--primary-500)] focus-visible:ring-offset-2',
        'touch-manipulation',
        variantStyles[variant],
        sizeStyles[size],
        isDisabled && 'opacity-50 cursor-not-allowed',
        className
      )}
      disabled={isDisabled}
      aria-disabled={isDisabled}
      aria-busy={loading}
      {...props}
    >
      {loading && (
        <svg
          className="animate-spin h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            className="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            strokeWidth="4"
          />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      )}
      {!loading && icon && iconPosition === 'left' && <span aria-hidden="true">{icon}</span>}
      {children}
      {!loading && icon && iconPosition === 'right' && <span aria-hidden="true">{icon}</span>}
    </button>
  )
}
