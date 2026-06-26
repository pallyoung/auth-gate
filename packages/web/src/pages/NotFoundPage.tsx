import { Home, ArrowLeft, MapPin } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function NotFoundPage() {
  const { t } = useTranslation('common')

  return (
    <div className="min-h-[calc(100vh-var(--layout-header-h)-2rem)] flex items-center justify-center px-4">
      <div className="text-center max-w-lg">
        {/* Illustration */}
        <div className="relative mb-8">
          <div className="text-[120px] font-bold font-[var(--font-display)] text-[var(--neutral-200)] select-none leading-none">
            404
          </div>
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="w-20 h-20 rounded-full bg-[var(--error-light)] flex items-center justify-center">
              <MapPin className="w-10 h-10 text-[var(--error)]" />
            </div>
          </div>
        </div>

        {/* Message */}
        <h1 className="text-2xl font-semibold text-[var(--text-primary)] mb-3">
          {t('notFound.title')}
        </h1>
        <p className="text-[var(--text-secondary)] mb-8 leading-relaxed">
          {t('notFound.description')}
        </p>

        {/* Actions */}
        <div className="flex items-center justify-center gap-3">
          <button
            onClick={() => window.history.back()}
            className="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium
                       text-[var(--text-secondary)] bg-[var(--bg-surface)] border border-[var(--border-subtle)]
                       hover:bg-[var(--bg-elevated)] hover:text-[var(--text-primary)]
                       transition-colors cursor-pointer"
          >
            <ArrowLeft className="w-4 h-4" />
            {t('notFound.goBack')}
          </button>
          <button
            onClick={() => { window.location.hash = '/' }}
            className="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium
                       text-white bg-[var(--primary-500)]
                       hover:bg-[var(--primary-400)]
                       transition-colors cursor-pointer"
          >
            <Home className="w-4 h-4" />
            {t('notFound.goHome')}
          </button>
        </div>
      </div>
    </div>
  )
}
