import { useTranslation } from 'react-i18next'
import type { HostProfile } from '../../../lib/api/types'
import { Badge } from '../../ui/Badge'
import { Button } from '../../ui/Button'

interface ProfileSwitcherProps {
  profiles: HostProfile[]
  selectedId: string
  disabled?: boolean
  onChange: (id: string) => void
}

export const ProfileSwitcher = ({ profiles, selectedId, disabled, onChange }: ProfileSwitcherProps) => {
  const { t } = useTranslation('hosts')
  if (profiles.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t('title')}>
      {profiles.map((p) => (
        <Button
          key={p.id}
          variant={p.id === selectedId ? 'primary' : 'secondary'}
          disabled={disabled}
          onClick={() => onChange(p.id)}
          aria-label={p.name}
          aria-describedby={p.is_active ? `profile-active-${p.id}` : undefined}
          aria-pressed={p.id === selectedId}
        >
          {p.name}
          {p.is_active && <Badge variant="success" className="ml-2" id={`profile-active-${p.id}`}>{t('active')}</Badge>}
        </Button>
      ))}
    </div>
  )
}
