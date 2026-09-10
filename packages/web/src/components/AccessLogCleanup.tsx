import React from 'react'
import { useTranslation } from 'react-i18next'
import { accessLogsApi } from '../lib/api/access-logs'
import { accessLogErrorKey } from '../lib/access-log-errors'
import type { AccessLogPurgeMode, AccessLogPurgeResult, AccessLogQueryParams } from '../lib/api/types'
import { Modal, ModalFooter } from './ui/Modal'
import { Button } from './ui/Button'
import { Select } from './ui/Select'
import { Alert } from './ui/Alert'

interface Props {
  filters: AccessLogQueryParams
  initialMode?: AccessLogPurgeMode
  onClose: () => void
  onPurged: (removed: number) => void
}

const filterLabels = {
  host: 'filters.host', client_ip: 'filters.clientIP', path: 'filters.path', username: 'filters.username',
  auth_result: 'filters.authResult', route_id: 'filters.route', status_code: 'filters.statusCode',
  start_time: 'time.start', end_time: 'time.end',
} as const

export function AccessLogCleanup({ filters, initialMode = 'filtered', onClose, onPurged }: Props) {
  const { t, i18n } = useTranslation('accessLogs')
  const [mode, setMode] = React.useState<AccessLogPurgeMode>(initialMode)
  const [preview, setPreview] = React.useState<AccessLogPurgeResult | null>(null)
  const [loading, setLoading] = React.useState(false)
  const [deleting, setDeleting] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [refresh, setRefresh] = React.useState(0)
  const pending = React.useRef(false)
  const requestId = React.useRef(0)
  const filterJSON = JSON.stringify(Object.fromEntries(
    Object.entries(filters).filter(([key, value]) => key in filterLabels && value !== undefined && value !== '')
  ))
  const cleanFilters = React.useMemo(() => JSON.parse(filterJSON) as AccessLogQueryParams, [filterJSON])
  const noFilters = mode === 'filtered' && Object.keys(cleanFilters).length === 0

  React.useEffect(() => {
    const id = ++requestId.current
    setPreview(null)
    setError(null)
    if (noFilters) { setLoading(false); return }
    setLoading(true)
    accessLogsApi.purge({ mode, filters: mode === 'filtered' ? cleanFilters : undefined, preview: true })
      .then((result) => { if (requestId.current === id) setPreview(result) })
      .catch((err) => { if (requestId.current === id) setError(accessLogErrorKey(err, 'preview')) })
      .finally(() => { if (requestId.current === id) setLoading(false) })
    return () => { requestId.current += 1 }
  }, [mode, cleanFilters, noFilters, refresh])

  const execute = async () => {
    if (pending.current || !preview || !preview.matched || loading || noFilters) return
    pending.current = true
    setDeleting(true)
    setError(null)
    try {
      const result = await accessLogsApi.purge({ mode, filters: mode === 'filtered' ? cleanFilters : undefined, preview: false, cutoff_time: preview.cutoff_time })
      onPurged(result.removed ?? 0)
    } catch (err) {
      setError(accessLogErrorKey(err, 'purge'))
    } finally {
      pending.current = false
      setDeleting(false)
    }
  }

  const displayValue = (key: string, value: string | number) => {
    if (key === 'start_time' || key === 'end_time') return new Date(value).toLocaleString(i18n.resolvedLanguage)
    if (key === 'auth_result' && ['pass', 'fail', 'none'].includes(String(value))) return t(`filters.${value}`)
    return String(value)
  }

  return <Modal open onClose={onClose} title={t('cleanup.title')} dialogRole="alertdialog" showCloseButton={false} closeDisabled={deleting}>
    <div className="space-y-4">
      <Button variant="ghost" disabled={deleting} onClick={onClose}>{t('cleanup.cancel')}</Button>
      <Select label={t('cleanup.mode')} value={mode} disabled={deleting} onChange={(event) => setMode(event.target.value as AccessLogPurgeMode)} options={[
        { value: 'filtered', label: t('cleanup.filtered') },
        { value: 'expired', label: t('cleanup.expired') },
        { value: 'all', label: t('cleanup.all') },
      ]} />
      {mode === 'filtered' && <>
        <p className="text-sm text-[var(--text-secondary)]">{t(noFilters ? 'cleanup.noFilters' : 'cleanup.filteredDescription')}</p>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
          {Object.entries(cleanFilters).map(([key, value]) => <React.Fragment key={key}>
            <dt className="text-[var(--text-muted)]">{t(filterLabels[key as keyof typeof filterLabels])}</dt>
            <dd className="break-all text-[var(--text-primary)]">{displayValue(key, value)}</dd>
          </React.Fragment>)}
        </dl>
      </>}
      {mode === 'all' && <p className="text-sm text-[var(--text-primary)]">{t('cleanup.allDescription')}</p>}
      {mode === 'expired' && preview && <p className="text-sm text-[var(--text-primary)]">{t(preview.retention_days ? 'cleanup.expiredDescription' : 'errors.retentionDisabled', { count: preview.retention_days })}</p>}
      <div role="status" className="min-h-12 text-sm text-[var(--text-primary)]">
        {loading ? t('cleanup.previewing') : preview ? t(preview.matched ? 'cleanup.matched' : 'cleanup.empty', { count: preview.matched ?? 0 }) : null}
        {preview && !(mode === 'expired' && !preview.retention_days) && <p className="mt-2 text-xs text-[var(--text-muted)]">{t('cleanup.cutoff', { time: new Date(preview.cutoff_time).toLocaleString(i18n.resolvedLanguage) })}</p>}
      </div>
      {error && <div role="alert"><Alert variant="error">{t(error)} <Button size="sm" variant="ghost" disabled={deleting} onClick={() => setRefresh((n) => n + 1)}>{t('page.retry')}</Button></Alert></div>}
      <p className="text-sm text-[var(--error)]">{t('cleanup.irreversible')}</p>
    </div>
    <ModalFooter>
      <Button variant="danger" className="min-w-40" loading={deleting} disabled={loading || !preview?.matched || noFilters} onClick={execute}>{t('cleanup.confirm', { count: preview?.matched ?? 0 })}</Button>
    </ModalFooter>
  </Modal>
}
