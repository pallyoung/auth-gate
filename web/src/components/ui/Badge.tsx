import React from 'react'
import { cn } from '../../lib/utils'

type BadgeVariant = 'default' | 'primary' | 'success' | 'warning' | 'error'
type BadgeSize = 'sm' | 'md'

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant
  badgeSize?: BadgeSize
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'bg-[var(--neutral-100)] text-[var(--neutral-700)]',
  primary: 'bg-[var(--primary-100)] text-[var(--primary-700)]',
  success: 'bg-[var(--success-light)] text-[var(--success)]',
  warning: 'bg-[var(--warning-light)] text-[var(--warning)]',
  error: 'bg-[var(--error-light)] text-[var(--error)]',
}

const sizeStyles: Record<BadgeSize, string> = {
  sm: 'h-5 px-1.5 text-[10px]',
  md: 'h-6 px-2 text-[var(--text-xs)]',
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
        'inline-flex items-center font-medium rounded-[var(--radius-sm)]',
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
