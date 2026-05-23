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
  const { i18n } = useTranslation()
  const active = i18n.resolvedLanguage === 'zh-CN' ? 'zh-CN' : 'en'

  return (
    <div
      className={cn(
        'inline-flex items-center rounded-full border border-[var(--border-default)] bg-[rgba(255,255,255,0.62)] p-1 shadow-[var(--shadow-sm)]',
        className
      )}
      role="group"
      aria-label="Language switcher"
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
              'rounded-full px-3 py-1.5 text-xs font-semibold transition-colors',
              isActive
                ? 'bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-sm)]'
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
