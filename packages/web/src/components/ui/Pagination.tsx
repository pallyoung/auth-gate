import React from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useTranslation } from 'react-i18next'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

export const Pagination: React.FC<PaginationProps> = ({ page, totalPages, onPageChange }) => {
  const { t, i18n } = useTranslation('common')
  const pages: (number | string)[] = []

  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) {
      pages.push(i)
    }
  } else {
    pages.push(1)
    if (page > 3) {
      pages.push('...')
    }
    for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) {
      pages.push(i)
    }
    if (page < totalPages - 2) {
      pages.push('...')
    }
    pages.push(totalPages)
  }

  return (
    <nav aria-label={t('pagination.label')} className="flex flex-wrap items-center justify-center gap-2 py-4">
      <button
        type="button"
        aria-label={t('pagination.previous')}
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className={cn(
          'flex h-8 w-8 items-center justify-center rounded-lg border transition-colors',
          page <= 1
            ? 'cursor-not-allowed border-[var(--border-default)] opacity-50'
            : 'border-[var(--border-default)] hover:bg-[var(--bg-elevated)]'
        )}
      >
        <ChevronLeft className="h-4 w-4" />
      </button>

      {pages.map((p, index) =>
        typeof p === 'number' ? (
          <button
            key={index}
            type="button"
            aria-label={t('pagination.page', { page: p })}
            aria-current={p === page ? 'page' : undefined}
            onClick={() => onPageChange(p)}
            className={cn(
              'flex h-8 w-8 items-center justify-center rounded-lg border text-sm transition-colors',
              p === page
                ? 'border-[var(--primary-500)] bg-[var(--primary-500)] text-white'
                : 'border-[var(--border-default)] hover:bg-[var(--bg-elevated)]'
            )}
          >
            {p.toLocaleString(i18n.resolvedLanguage)}
          </button>
        ) : (
          <span key={index} className="flex h-8 w-8 items-center justify-center text-[var(--text-muted)]">
            {p}
          </span>
        )
      )}

      <button
        type="button"
        aria-label={t('pagination.next')}
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className={cn(
          'flex h-8 w-8 items-center justify-center rounded-lg border transition-colors',
          page >= totalPages
            ? 'cursor-not-allowed border-[var(--border-default)] opacity-50'
            : 'border-[var(--border-default)] hover:bg-[var(--bg-elevated)]'
        )}
      >
        <ChevronRight className="h-4 w-4" />
      </button>
    </nav>
  )
}
