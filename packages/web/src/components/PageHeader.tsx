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
    <div className="mb-6 rounded-[32px] border border-[var(--border-soft)] bg-[linear-gradient(135deg,rgba(255,255,255,0.78),rgba(255,255,255,0.42))] px-5 py-6 shadow-[var(--shadow-md)] backdrop-blur-xl md:mb-8 md:px-7 md:py-7">
      <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
        <div className="min-w-0">
          <div className="eyebrow">
            <span className="inline-flex h-2.5 w-2.5 rounded-full bg-[var(--primary-500)] animate-pulse-glow" />
            {eyebrow ?? t('brand.controlPlane')}
          </div>
          <h1 className="mt-3 text-[var(--text-2xl)] font-semibold tracking-[-0.04em] text-[var(--text-primary)]">
            {title}
          </h1>
          {description && (
            <p className="mt-2 max-w-2xl text-sm leading-6 text-[var(--text-muted)] md:text-base">
              {description}
            </p>
          )}
          {meta && <div className={cn('mt-4 flex flex-wrap items-center gap-3')}>{meta}</div>}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
    </div>
  )
}
