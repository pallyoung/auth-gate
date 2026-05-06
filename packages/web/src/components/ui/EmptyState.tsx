import React from 'react'
import { cn } from '../../lib/utils'

interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-4 text-center" role="status">
      {icon && (
        <div className="mb-4 text-[var(--neutral-400)]" aria-hidden="true">
          {icon}
        </div>
      )}
      <h3 className="text-[var(--text-lg)] font-medium text-[var(--text-primary)] mb-1">
        {title}
      </h3>
      {description && (
        <p className="text-[var(--text-sm)] text-[var(--text-muted)] max-w-sm mb-4">
          {description}
        </p>
      )}
      {action}
    </div>
  )
}
