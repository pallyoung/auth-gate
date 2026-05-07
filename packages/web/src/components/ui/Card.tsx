import React from 'react'
import { cn } from '../../lib/utils'

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  padding?: 'none' | 'sm' | 'md' | 'lg'
  tone?: 'default' | 'soft' | 'strong' | 'inverse'
}

const paddingStyles = {
  none: '',
  sm: 'p-4',
  md: 'p-5 md:p-6',
  lg: 'p-6 md:p-7',
}

const toneStyles = {
  default: 'glass-panel',
  soft: 'bg-[var(--bg-card-soft)] border border-[var(--border-soft)] shadow-[var(--shadow-sm)] backdrop-blur-xl',
  strong: 'glass-panel-strong',
  inverse: 'bg-[var(--bg-inverse)] text-[var(--text-inverse)] border border-white/10 shadow-[var(--shadow-lg)]',
}

export function Card({
  padding = 'md',
  tone = 'default',
  className,
  children,
  ...props
}: CardProps) {
  return (
    <div
      className={cn(
        'rounded-[var(--radius-lg)]',
        toneStyles[tone],
        paddingStyles[padding],
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}

interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {}

export function CardHeader({ className, children, ...props }: CardHeaderProps) {
  return (
    <div className={cn('mb-5 flex items-start justify-between gap-4', className)} {...props}>
      {children}
    </div>
  )
}

interface CardTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {}

export function CardTitle({ className, children, ...props }: CardTitleProps) {
  return (
    <h3 className={cn('text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]', className)} {...props}>
      {children}
    </h3>
  )
}

interface CardContentProps extends React.HTMLAttributes<HTMLDivElement> {}

export function CardContent({ className, children, ...props }: CardContentProps) {
  return (
    <div className={cn('', className)} {...props}>
      {children}
    </div>
  )
}

interface CardFooterProps extends React.HTMLAttributes<HTMLDivElement> {}

export function CardFooter({ className, children, ...props }: CardFooterProps) {
  return (
    <div
      className={cn(
        'mt-5 flex items-center justify-end gap-2 border-t border-[var(--border-default)] pt-4',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}
