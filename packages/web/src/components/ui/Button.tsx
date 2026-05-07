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
  primary: cn(
    'text-[var(--text-inverse)] shadow-[var(--shadow-md)]',
    'bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))]',
    'hover:-translate-y-0.5 hover:shadow-[var(--shadow-lg)]'
  ),
  secondary: cn(
    'glass-panel text-[var(--text-primary)]',
    'hover:bg-[var(--bg-card-strong)] hover:border-[var(--border-strong)] hover:-translate-y-0.5'
  ),
  ghost: cn(
    'text-[var(--text-secondary)] border border-transparent',
    'hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]'
  ),
  danger: cn(
    'text-white shadow-[var(--shadow-sm)]',
    'bg-[linear-gradient(135deg,var(--error),#a72d34)]',
    'hover:-translate-y-0.5 hover:shadow-[var(--shadow-md)]'
  ),
}

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'h-9 px-3.5 text-sm gap-1.5',
  md: 'h-11 px-5 text-sm gap-2',
  lg: 'h-12 px-6 text-base gap-2.5',
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
  type,
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading

  return (
    <button
      type={type ?? 'button'}
      className={cn(
        'inline-flex items-center justify-center rounded-full font-semibold tracking-[-0.01em]',
        'transition-all duration-[var(--duration-normal)] ease-[var(--ease-out)]',
        'focus-visible:outline-none touch-manipulation',
        'disabled:translate-y-0 disabled:shadow-none disabled:opacity-55',
        variantStyles[variant],
        sizeStyles[size],
        className
      )}
      disabled={isDisabled}
      aria-disabled={isDisabled}
      aria-busy={loading}
      {...props}
    >
      {loading && (
        <svg className="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24" aria-hidden="true">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      )}
      {!loading && icon && iconPosition === 'left' && (
        <span className="inline-flex items-center" aria-hidden="true">
          {icon}
        </span>
      )}
      <span>{children}</span>
      {!loading && icon && iconPosition === 'right' && (
        <span className="inline-flex items-center" aria-hidden="true">
          {icon}
        </span>
      )}
    </button>
  )
}
