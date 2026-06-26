import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'

const languageOptions = [
  { value: 'en', label: 'EN', ariaLabel: 'English' },
  { value: 'zh-CN', label: '中文', ariaLabel: '中文' },
] as const

interface LanguageSwitcherProps {
  className?: string
}

export function LanguageSwitcher({ className }: LanguageSwitcherProps) {
  const { i18n, t } = useTranslation('common')
  const active = i18n.resolvedLanguage === 'zh-CN' ? 'zh-CN' : 'en'

  return (
    <div
      className={cn(
        'inline-flex items-center rounded-[8px] border border-[var(--border-default)] bg-[var(--bg-card)] p-1',
        className
      )}
      role="group"
      aria-label={t('accessibility.languageSwitcher')}
    >
      {languageOptions.map((option) => {
        const isActive = active === option.value
        return (
          <button
            key={option.value}
            type="button"
            aria-label={option.ariaLabel}
            aria-pressed={isActive}
            onClick={() => void i18n.changeLanguage(option.value)}
            className={cn(
              'inline-flex min-h-8 min-w-8 items-center justify-center rounded-[6px] px-3 py-1 text-xs font-medium transition-colors',
              isActive
                ? 'bg-[var(--bg-soft-primary)] text-[var(--primary-600)]'
                : 'text-[var(--text-muted)] hover:text-[var(--text-primary)]'
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
