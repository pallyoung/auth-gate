import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { HostProfileInput } from '../../../lib/api/types'
import { Button } from '../../ui/Button'
import { Input } from '../../ui/Input'

interface HostProfileFormProps {
  initial?: Partial<HostProfileInput>
  onSubmit: (data: HostProfileInput) => Promise<void> | void
  onCancel: () => void
}

const profileNameRegex = /^[A-Za-z0-9 _.\-]+$/
const maxNameLength = 32

export const HostProfileForm = ({ initial, onSubmit, onCancel }: HostProfileFormProps) => {
  const { t } = useTranslation('hosts')
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmedName = name.trim()
    if (trimmedName.length === 0 || trimmedName.length > maxNameLength || !profileNameRegex.test(trimmedName)) {
      setError(t('errors.invalid_host_profile_name'))
      return
    }
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit({ name: trimmedName, description: description.trim() })
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label htmlFor="host-profile-name" className="block text-xs font-semibold text-[var(--text-muted)]">
          {t('form.name')}
        </label>
        <Input id="host-profile-name" value={name} onChange={(e) => setName(e.target.value)} maxLength={maxNameLength} />
      </div>
      <div>
        <label htmlFor="host-profile-description" className="block text-xs font-semibold text-[var(--text-muted)]">
          {t('form.description')}
        </label>
        <Input id="host-profile-description" value={description} onChange={(e) => setDescription(e.target.value)} />
      </div>
      {error && <p className="text-sm text-[var(--color-danger-500)]">{error}</p>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} disabled={submitting}>
          {t('form.cancel')}
        </Button>
        <Button type="submit" disabled={submitting}>
          {t('form.saveProfile')}
        </Button>
      </div>
    </form>
  )
}
