import React from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from './ui/Input'
import { Select } from './ui/Select'
import { Button } from './ui/Button'
import { localDateTimeToISO, toLocalDateTime, presetTimeRange, type LogTimePreset } from '../lib/access-log-time'
import type { AccessLogQueryParams } from '../lib/api/types'

interface Props {
  filters: AccessLogQueryParams
  preset: LogTimePreset
  onApply: (range: Pick<AccessLogQueryParams, 'start_time' | 'end_time'>, preset: LogTimePreset) => void
}

export function AccessLogTimeRange({ filters, preset, onApply }: Props) {
  const { t } = useTranslation('accessLogs')
  const [choice, setChoice] = React.useState(preset)
  const [start, setStart] = React.useState(() => toLocalDateTime(filters.start_time))
  const [end, setEnd] = React.useState(() => toLocalDateTime(filters.end_time))
  const [error, setError] = React.useState<'start' | 'end' | 'reversed' | null>(null)
  const startRef = React.useRef<HTMLInputElement>(null)
  const endRef = React.useRef<HTMLInputElement>(null)
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone

  React.useEffect(() => {
    setChoice(preset)
    setStart(toLocalDateTime(filters.start_time))
    setEnd(toLocalDateTime(filters.end_time))
    setError(null)
  }, [preset, filters.start_time, filters.end_time])

  const apply = (event: React.FormEvent) => {
    event.preventDefault()
    const startISO = localDateTimeToISO(start)
    const endISO = localDateTimeToISO(end)
    if (!startISO) { setError('start'); startRef.current?.focus(); return }
    if (!endISO) { setError('end'); endRef.current?.focus(); return }
    if (startISO > endISO) { setError('reversed'); endRef.current?.focus(); return }
    setError(null)
    onApply({ start_time: startISO, end_time: endISO }, 'custom')
  }

  return (
    <form noValidate onSubmit={apply} className="space-y-3 border-t border-[var(--border-default)] pt-4">
      <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Select
          id="access-log-time-range" label={t('filters.timeRange')} value={choice}
          options={[
            { value: '1h', label: t('filters.lastHour') },
            { value: '6h', label: t('filters.last6Hours') },
            { value: '24h', label: t('filters.last24Hours') },
            { value: '7d', label: t('filters.last7Days') },
            { value: 'all', label: t('time.all') },
            { value: 'custom', label: t('time.custom') },
          ]}
          onChange={(event) => {
            const next = event.target.value as LogTimePreset
            setChoice(next)
            setError(null)
            if (next !== 'custom') onApply(presetTimeRange(next), next)
          }}
        />
        {choice === 'custom' && <>
          <Input ref={startRef} id="access-log-start" label={t('time.start')} placeholder="YYYY-MM-DD HH:mm" value={start}
            onChange={(event) => { setStart(event.target.value); setError(null) }}
            error={error === 'start' ? t('time.invalid') : undefined} />
          <Input ref={endRef} id="access-log-end" label={t('time.end')} placeholder="YYYY-MM-DD HH:mm" value={end}
            onChange={(event) => { setEnd(event.target.value); setError(null) }}
            error={error === 'end' ? t('time.invalid') : error === 'reversed' ? t('time.reversed') : undefined} />
          <Button type="submit" variant="secondary" className="self-start sm:mt-6">{t('time.apply')}</Button>
        </>}
      </div>
      {choice === 'custom' && <p className="text-xs text-[var(--text-muted)]">{t('time.format', { zone })}</p>}
    </form>
  )
}
