import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { HostEntryInput } from '../../../lib/api/types'
import { Button } from '../../ui/Button'
import { Input } from '../../ui/Input'
import { Switch } from '../../ui/Switch'

interface HostEntryFormProps {
  initial?: Partial<HostEntryInput>
  onSubmit: (data: HostEntryInput) => Promise<void> | void
  onCancel: () => void
}

const ipv4 = /^(25[0-5]|2[0-4]\d|[01]?\d\d?)(\.(25[0-5]|2[0-4]\d|[01]?\d\d?)){3}$/
const ipv6Simple = /^[0-9a-fA-F:]+$/
const hostnameRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

export const HostEntryForm = ({ initial, onSubmit, onCancel }: HostEntryFormProps) => {
  const { t } = useTranslation('hosts')
  const [ip, setIp] = useState(initial?.ip ?? '')
  const [hostnamesText, setHostnamesText] = useState((initial?.hostnames ?? []).join(' '))
  const [comment, setComment] = useState(initial?.comment ?? '')
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmedIP = ip.trim()
    if (!ipv4.test(trimmedIP) && !ipv6Simple.test(trimmedIP)) {
      setError(t('errors.invalid_host_ip'))
      return
    }
    const hostnames = hostnamesText
      .split(/\s+/)
      .map((h) => h.trim())
      .filter((h) => h.length > 0)
    if (hostnames.length === 0 || hostnames.some((h) => !hostnameRegex.test(h))) {
      setError(t('errors.invalid_host_hostname'))
      return
    }
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit({ ip: trimmedIP, hostnames, comment, enabled })
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label htmlFor="host-entry-ip" className="block text-xs font-semibold text-[var(--text-muted)]">{t('form.ip')}</label>
        <Input id="host-entry-ip" value={ip} onChange={(e) => setIp(e.target.value)} />
      </div>
      <div>
        <label htmlFor="host-entry-hostnames" className="block text-xs font-semibold text-[var(--text-muted)]">{t('form.hostnames')}</label>
        <Input id="host-entry-hostnames" value={hostnamesText} onChange={(e) => setHostnamesText(e.target.value)} />
      </div>
      <div>
        <label htmlFor="host-entry-comment" className="block text-xs font-semibold text-[var(--text-muted)]">{t('form.comment')}</label>
        <Input id="host-entry-comment" value={comment} onChange={(e) => setComment(e.target.value)} />
      </div>
      <div className="flex items-center gap-2">
        <Switch checked={enabled} onChange={(event) => setEnabled(event.target.checked)} aria-label={t('form.enabled')} />
        <span className="text-sm">{t('form.enabled')}</span>
      </div>
      {error && <p className="text-sm text-[var(--color-danger-500)]">{error}</p>}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} disabled={submitting}>
          {t('form.cancel')}
        </Button>
        <Button type="submit" disabled={submitting}>
          {t('form.saveEntry')}
        </Button>
      </div>
    </form>
  )
}
