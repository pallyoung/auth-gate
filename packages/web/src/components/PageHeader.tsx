import React from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'

interface PageHeaderProps {
  title: string
  description?: string
  eyebrow?: string
  action?: React.ReactNode
  meta?: React.ReactNode
}

export function PageHeader({
  title,
  description,
  eyebrow,
  action,
  meta,
}: PageHeaderProps) {
  const { t } = useTranslation('layout')

  return (
    <div className="mb-6 md:mb-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div className="min-w-0">
          {eyebrow && (
            <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--primary-600)]">
              {eyebrow}
            </div>
          )}
          <h1 className="text-[var(--text-2xl)] font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
            {title}
          </h1>
          {description && (
            <p className="mt-1.5 max-w-2xl text-sm text-[var(--text-muted)]">
              {description}
            </p>
          )}
          {meta && <div className={cn('mt-3 flex flex-wrap items-center gap-3')}>{meta}</div>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
    </div>
  )
}
