import React from 'react'
import { cn } from '../../lib/utils'

type AlertVariant = 'info' | 'success' | 'warning' | 'error'

interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: AlertVariant
  title?: string
  icon?: React.ReactNode
}

const variantStyles: Record<AlertVariant, string> = {
  info: 'bg-[var(--info-light)] text-[var(--info)] border-[rgba(47,114,200,0.14)]',
  success: 'bg-[var(--success-light)] text-[var(--success)] border-[rgba(31,157,103,0.14)]',
  warning: 'bg-[var(--warning-light)] text-[var(--warning)] border-[rgba(201,129,29,0.16)]',
  error: 'bg-[var(--error-light)] text-[var(--error)] border-[rgba(208,71,75,0.16)]',
}

export function Alert({
  variant = 'info',
  title,
  icon,
  className,
  children,
  ...props
}: AlertProps) {
  return (
    <div
      className={cn(
        'flex items-start gap-3 rounded-[12px] border px-4 py-3',
        variantStyles[variant],
        className
      )}
      {...props}
    >
      {icon ? <div className="mt-0.5 shrink-0">{icon}</div> : null}
      <div className="min-w-0">
        {title ? <div className="text-sm font-semibold">{title}</div> : null}
        <div className={cn('text-sm leading-6', title && 'mt-1')}>{children}</div>
      </div>
    </div>
  )
}
