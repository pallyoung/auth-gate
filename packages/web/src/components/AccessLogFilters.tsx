import React from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { Input } from './ui/Input'
import { Select } from './ui/Select'
import type { AccessLogQueryParams } from '../lib/api/types'

interface AccessLogFiltersProps {
  filters: AccessLogQueryParams
  onChange: (filters: AccessLogQueryParams) => void
}

type TextFilterKey = 'client_ip' | 'path' | 'username'
type TextFilterDraft = Record<TextFilterKey, string>

function getTextFilterDraft(filters: AccessLogQueryParams): TextFilterDraft {
  return {
    client_ip: filters.client_ip ?? '',
    path: filters.path ?? '',
    username: filters.username ?? '',
  }
}

function mergeTextFilters(
  filters: AccessLogQueryParams,
  draft: TextFilterDraft
): AccessLogQueryParams {
  const next = { ...filters }
  for (const key of Object.keys(draft) as TextFilterKey[]) {
    if (draft[key]) next[key] = draft[key]
    else delete next[key]
  }
  return next
}

export const AccessLogFilters: React.FC<AccessLogFiltersProps> = ({ filters, onChange }) => {
  const { t } = useTranslation('accessLogs')
  const [textDraft, setTextDraft] = React.useState<TextFilterDraft>(() => getTextFilterDraft(filters))
  const textDraftRef = React.useRef(textDraft)
  const filtersRef = React.useRef(filters)
  const onChangeRef = React.useRef(onChange)
  const composingRef = React.useRef(false)
  const debounceTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)
  const inputRefs = React.useRef<Record<TextFilterKey, HTMLInputElement | null>>({
    client_ip: null,
    path: null,
    username: null,
  })

  filtersRef.current = filters
  onChangeRef.current = onChange

  React.useEffect(() => {
    const nextDraft = getTextFilterDraft(filters)
    textDraftRef.current = nextDraft
    setTextDraft(nextDraft)
  }, [filters])

  React.useEffect(() => () => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
  }, [])

  const commitTextFilters = React.useCallback((draft: TextFilterDraft) => {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current)
    debounceTimerRef.current = setTimeout(() => {
      debounceTimerRef.current = null
      onChangeRef.current(mergeTextFilters(filtersRef.current, draft))
    }, 300)
  }, [])

  const handleTextChange = (key: TextFilterKey, value: string) => {
    const nextDraft = { ...textDraftRef.current, [key]: value }
    textDraftRef.current = nextDraft
    setTextDraft(nextDraft)
    if (!composingRef.current) commitTextFilters(nextDraft)
  }

  const handleCompositionStart = () => {
    composingRef.current = true
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
      debounceTimerRef.current = null
    }
  }

  const handleCompositionEnd = () => {
    composingRef.current = false
    commitTextFilters(textDraftRef.current)
  }

  const clearTextFilter = (key: TextFilterKey) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
      debounceTimerRef.current = null
    }
    const nextDraft = { ...textDraftRef.current, [key]: '' }
    textDraftRef.current = nextDraft
    setTextDraft(nextDraft)
    onChangeRef.current(mergeTextFilters(filtersRef.current, nextDraft))
    inputRefs.current[key]?.focus()
  }

  const clearButton = (key: TextFilterKey, label: string) => textDraft[key] ? (
    <button
      type="button"
      onClick={() => clearTextFilter(key)}
      className="absolute right-2 top-1/2 z-10 flex h-8 w-8 -translate-y-1/2 cursor-pointer items-center justify-center rounded-[8px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--primary-500)]"
      aria-label={`${t('filters.clear')} ${label}`}
    >
      <X className="h-4 w-4" aria-hidden="true" />
    </button>
  ) : null

  const handleChange = (key: keyof AccessLogQueryParams, value: string | number | undefined) => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
      debounceTimerRef.current = null
    }
    onChange({ ...mergeTextFilters(filters, textDraftRef.current), [key]: value })
  }

  const authResultOptions = [
    { value: '', label: t('filters.all') },
    { value: 'pass', label: t('filters.pass') },
    { value: 'fail', label: t('filters.fail') },
    { value: 'none', label: t('filters.none') },
  ]

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      <div>
        <label htmlFor="access-log-client-ip" className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.clientIP')}
        </label>
        <div className="relative">
          <Input
            ref={(node) => { inputRefs.current.client_ip = node }}
            id="access-log-client-ip"
            placeholder="192.168.1.1"
            value={textDraft.client_ip}
            onChange={(e) => handleTextChange('client_ip', e.target.value)}
            onCompositionStart={handleCompositionStart}
            onCompositionEnd={handleCompositionEnd}
            className={textDraft.client_ip ? 'pr-11' : undefined}
          />
          {clearButton('client_ip', t('filters.clientIP'))}
        </div>
      </div>

      <div>
        <label htmlFor="access-log-path" className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.path')}
        </label>
        <div className="relative">
          <Input
            ref={(node) => { inputRefs.current.path = node }}
            id="access-log-path"
            placeholder="/api/users"
            value={textDraft.path}
            onChange={(e) => handleTextChange('path', e.target.value)}
            onCompositionStart={handleCompositionStart}
            onCompositionEnd={handleCompositionEnd}
            className={textDraft.path ? 'pr-11' : undefined}
          />
          {clearButton('path', t('filters.path'))}
        </div>
      </div>

      <div>
        <label htmlFor="access-log-username" className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.username')}
        </label>
        <div className="relative">
          <Input
            ref={(node) => { inputRefs.current.username = node }}
            id="access-log-username"
            placeholder="admin"
            value={textDraft.username}
            onChange={(e) => handleTextChange('username', e.target.value)}
            onCompositionStart={handleCompositionStart}
            onCompositionEnd={handleCompositionEnd}
            className={textDraft.username ? 'pr-11' : undefined}
          />
          {clearButton('username', t('filters.username'))}
        </div>
      </div>

      <div>
        <label htmlFor="access-log-auth-result" className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.authResult')}
        </label>
        <Select
          id="access-log-auth-result"
          options={authResultOptions}
          value={filters.auth_result || ''}
          onChange={(e) => handleChange('auth_result', e.target.value || undefined)}
        />
      </div>
    </div>
  )
}
