import React from 'react'
import { cn } from '../../lib/utils'

type BadgeVariant = 'default' | 'primary' | 'success' | 'warning' | 'error'
type BadgeSize = 'sm' | 'md'

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant
  badgeSize?: BadgeSize
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'bg-[rgba(23,33,45,0.08)] text-[var(--text-secondary)] border border-[rgba(23,33,45,0.08)]',
  primary: 'bg-[var(--bg-soft-primary)] text-[var(--primary-700)] border border-[rgba(15,143,139,0.14)]',
  success: 'bg-[var(--success-light)] text-[var(--success)] border border-[rgba(31,157,103,0.12)]',
  warning: 'bg-[var(--warning-light)] text-[var(--warning)] border border-[rgba(201,129,29,0.14)]',
  error: 'bg-[var(--error-light)] text-[var(--error)] border border-[rgba(208,71,75,0.14)]',
}

const sizeStyles: Record<BadgeSize, string> = {
  sm: 'min-h-6 px-2.5 text-[11px]',
  md: 'min-h-7 px-3 text-xs',
}

export function Badge({
  variant = 'default',
  badgeSize = 'md',
  className,
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center justify-center rounded-full font-semibold uppercase tracking-[0.12em]',
        variantStyles[variant],
        sizeStyles[badgeSize],
        className
      )}
      {...props}
    >
      {children}
    </span>
  )
}
