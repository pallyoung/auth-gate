import React from 'react'

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div
      className="flex flex-col items-center justify-center px-5 py-14 text-center md:px-8 md:py-16"
      role="status"
    >
      {icon && (
        <div
          className="mb-5 flex h-16 w-16 items-center justify-center rounded-[22px] bg-[var(--bg-soft-primary)] text-[var(--primary-600)] shadow-[var(--shadow-sm)]"
          aria-hidden="true"
        >
          {icon}
        </div>
      )}
      <h3 className="text-xl font-semibold tracking-[-0.02em] text-[var(--text-primary)]">{title}</h3>
      {description && (
        <p className="mt-2 max-w-md text-sm leading-6 text-[var(--text-muted)]">{description}</p>
      )}
      {action && <div className="mt-6">{action}</div>}
    </div>
  )
}
