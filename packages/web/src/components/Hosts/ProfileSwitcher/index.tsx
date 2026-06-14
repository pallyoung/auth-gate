import { useTranslation } from 'react-i18next'
import type { HostProfile } from '../../../lib/api/types'
import { Button } from '../../ui/Button'

interface ProfileSwitcherProps {
  profiles: HostProfile[]
  activeId: string
  canManage: boolean
  onChange: (id: string) => void
}

export const ProfileSwitcher = ({ profiles, activeId, canManage, onChange }: ProfileSwitcherProps) => {
  const { t } = useTranslation('hosts')
  if (profiles.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t('title')}>
      {profiles.map((p) => (
        <Button
          key={p.id}
          variant={p.id === activeId ? 'primary' : 'secondary'}
          onClick={() => onChange(p.id)}
          aria-pressed={p.id === activeId}
        >
          {p.name}
        </Button>
      ))}
      {!canManage && (
        <span className="ml-2 text-xs text-[var(--text-muted)]">
          {activeId ? t('active') : null}
        </span>
      )}
    </div>
  )
}
