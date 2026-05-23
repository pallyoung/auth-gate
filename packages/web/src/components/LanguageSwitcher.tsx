import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'

const options = [
  { code: 'en', label: 'EN', ariaLabel: 'English' },
  { code: 'zh-CN', label: '中文', ariaLabel: '中文' },
] as const

export function LanguageSwitcher() {
  const { i18n } = useTranslation()

  return (
    <div
      className="inline-flex items-center gap-1 rounded-full border border-[var(--border-soft)] bg-[rgba(255,255,255,0.52)] p-1"
      role="group"
      aria-label="Language switcher"
    >
      {options.map((option) => {
        const active = i18n.resolvedLanguage === option.code || i18n.language === option.code

        return (
          <button
            key={option.code}
            type="button"
            aria-label={option.ariaLabel}
            aria-pressed={active}
            onClick={() => {
              void i18n.changeLanguage(option.code)
            }}
            className={cn(
              'rounded-full px-3 py-1.5 text-xs font-semibold transition-colors',
              active
                ? 'bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-sm)]'
                : 'text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]'
            )}
          >
            {option.label}
          </button>
        )
      })}
    </div>
  )
}
