import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import type { Certificate, Route, RouteBackend, RouteInput } from '../lib/api/types'
import { certificatesApi } from '../lib/api/certificates'
import { Alert, Button, Card, Input, Select, Switch } from './ui'

interface RouteFormProps {
  route: Route | null
  certificates: Certificate[]
  onSubmit: (data: RouteInput) => void
  onCancel: () => void
}

type TLSCertMode = 'system' | 'upload'

function getInitialBackends(route: Route | null): RouteBackend[] {
  return (route?.backends || []).map((backend) => ({
    ...backend,
    url: backend.url || '',
    weight: backend.weight || 1,
    dial_timeout_ms: backend.dial_timeout_ms || 0,
    read_timeout_ms: backend.read_timeout_ms || 0,
    write_timeout_ms: backend.write_timeout_ms || 0,
    max_idle_conns: backend.max_idle_conns || 0,
  }))
}

function getInitialRouteForm(route: Route | null): RouteInput {
  return {
    name: route?.name || '',
    host: route?.host || '',
    path_prefix: route?.path_prefix || '',
    backend: route?.backend || '',
    backends: getInitialBackends(route),
    strip_prefix: route?.strip_prefix ?? true,
    enabled: route?.enabled ?? true,
    priority: route?.priority || 0,
    type: route?.type || 'proxy',
    static_root: route?.static_root || '',
    static_spa: route?.static_spa ?? false,
    tls_cert: route?.tls_cert || '',
    tls_key: route?.tls_key || '',
    tls_enabled: route?.tls_enabled ?? false,
    https_redirect: route?.https_redirect ?? false,
    certificate_id: route?.certificate_id || '',
    timeout_ms: route?.timeout_ms || 0,
    retry_attempts: route?.retry_attempts || 0,
    path_match_mode: route?.path_match_mode || '',
    header_name: route?.header_name || '',
    header_value: route?.header_value || '',
    rewrite_target: route?.rewrite_target || '',
    redirect_code: route?.redirect_code || 0,
    // Header manipulation
    set_request_headers: route?.set_request_headers || {},
    remove_request_headers: route?.remove_request_headers || [],
    add_response_headers: route?.add_response_headers || {},
    remove_response_headers: route?.remove_response_headers || [],
  }
}

function normalizeHost(host: string | undefined) {
  const trimmed = (host || '').trim().toLowerCase()
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    return trimmed.slice(1, -1)
  }
  return trimmed
}

/** Remove entries with empty keys from a header map, or return undefined if empty. */
function cleanHeaderMap(m: Record<string, string> | undefined): Record<string, string> | undefined {
  if (!m) return undefined
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(m)) {
    if (k.trim() !== '') out[k.trim()] = v
  }
  return Object.keys(out).length > 0 ? out : undefined
}

/** Trim and drop empty strings from a header name list, or return undefined if empty. */
function cleanHeaderList(arr: string[] | undefined): string[] | undefined {
  if (!arr) return undefined
  const out = arr.map((s) => s.trim()).filter((s) => s !== '')
  return out.length > 0 ? out : undefined
}

// ---- Header editor sub-components ----

interface HeaderMapSectionProps {
  label: string
  hint: string
  entries: Record<string, string>
  addLabel: string
  keyPlaceholder: string
  valuePlaceholder: string
  removeLabel: string
  emptyText: string
  onChange: (entries: Record<string, string>) => void
  idPrefix?: string
}

function HeaderMapSection({
  label, hint, entries, addLabel, keyPlaceholder, valuePlaceholder, removeLabel, emptyText, onChange, idPrefix,
}: HeaderMapSectionProps) {
  const pairs = Object.entries(entries)
  return (
    <div className="space-y-2">
      <div className="text-xs font-semibold text-[var(--text-muted)]">{label}</div>
      <p className="text-xs text-[var(--text-muted)]">{hint}</p>
      {pairs.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[var(--border-default)] px-4 py-3 text-xs text-[var(--text-muted)]">
          {emptyText}
        </div>
      ) : (
        <div className="space-y-2">
          {pairs.map(([k, v], i) => (
            <div key={`${k}-${i}`} className="flex items-end gap-2">
              <div className="flex-1">
                <Input
                  id={idPrefix ? `${idPrefix}-key-${i}` : undefined}
                  label={keyPlaceholder}
                  value={k}
                  onChange={(e) => {
                    const next = { ...entries }
                    delete next[k]
                    next[e.target.value] = v
                    onChange(next)
                  }}
                  placeholder={keyPlaceholder}
                />
              </div>
              <div className="flex-1">
                <Input
                  id={idPrefix ? `${idPrefix}-val-${i}` : undefined}
                  label={valuePlaceholder}
                  value={v}
                  onChange={(e) => onChange({ ...entries, [k]: e.target.value })}
                  placeholder={valuePlaceholder}
                />
              </div>
              <Button variant="ghost" size="sm" onClick={() => {
                const next = { ...entries }
                delete next[k]
                onChange(next)
              }}>
                {removeLabel}
              </Button>
            </div>
          ))}
        </div>
      )}
      <Button variant="secondary" size="sm" onClick={() => onChange({ ...entries, '': '' })}>
        {addLabel}
      </Button>
    </div>
  )
}

interface HeaderListSectionProps {
  label: string
  hint: string
  entries: string[]
  addLabel: string
  placeholder: string
  removeLabel: string
  emptyText: string
  onChange: (entries: string[]) => void
  idPrefix?: string
}

function HeaderListSection({
  label, hint, entries, addLabel, placeholder, removeLabel, emptyText, onChange, idPrefix,
}: HeaderListSectionProps) {
  return (
    <div className="space-y-2">
      <div className="text-xs font-semibold text-[var(--text-muted)]">{label}</div>
      <p className="text-xs text-[var(--text-muted)]">{hint}</p>
      {entries.length === 0 ? (
        <div className="rounded-lg border border-dashed border-[var(--border-default)] px-4 py-3 text-xs text-[var(--text-muted)]">
          {emptyText}
        </div>
      ) : (
        <div className="space-y-2">
          {entries.map((name, i) => (
            <div key={`${name}-${i}`} className="flex items-end gap-2">
              <div className="flex-1">
                <Input
                  id={idPrefix ? `${idPrefix}-${i}` : undefined}
                  label={placeholder}
                  value={name}
                  onChange={(e) => {
                    const next = [...entries]
                    next[i] = e.target.value
                    onChange(next)
                  }}
                  placeholder={placeholder}
                />
              </div>
              <Button variant="ghost" size="sm" onClick={() => onChange(entries.filter((_, j) => j !== i))}>
                {removeLabel}
              </Button>
            </div>
          ))}
        </div>
      )}
      <Button variant="secondary" size="sm" onClick={() => onChange([...entries, ''])}>
        {addLabel}
      </Button>
    </div>
  )
}

export function RouteForm({ route, certificates, onSubmit, onCancel }: RouteFormProps) {
  const { t } = useTranslation('routes')
  const [form, setForm] = React.useState<RouteInput>(() => getInitialRouteForm(route))
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const initialFormSeed = JSON.stringify(getInitialRouteForm(route))
  const isRegexPathMatchMode = form.path_match_mode === 'regex' || form.path_match_mode === 'regex_i'
  const errorMessage = resolveLocalizedText(t, error)

  const [tlsCertMode, setTlsCertMode] = React.useState<TLSCertMode>('system')
  const [uploadName, setUploadName] = React.useState('')
  const [uploadDomain, setUploadDomain] = React.useState('')
  const [uploadCertPem, setUploadCertPem] = React.useState('')
  const [uploadKeyPem, setUploadKeyPem] = React.useState('')

  const activeCerts = React.useMemo(
    () => certificates.filter((cert) => cert.status === 'active'),
    [certificates]
  )

  React.useEffect(() => {
    setForm(getInitialRouteForm(route))
    setError(null)
    setTlsCertMode(route?.certificate_id ? 'system' : 'system')
    setUploadName('')
    setUploadDomain('')
    setUploadCertPem('')
    setUploadKeyPem('')
  }, [initialFormSeed])

  const updateBackend = (index: number, updater: (current: RouteBackend) => RouteBackend) => {
    setForm((current) => ({
      ...current,
      backends: (current.backends || []).map((backend, backendIndex) => (
        backendIndex === index ? updater(backend) : backend
      )),
    }))
  }

  const addBackend = () => {
    setForm((current) => ({
      ...current,
      backends: [
        ...(current.backends || []),
        { url: '', weight: 1, dial_timeout_ms: 0, read_timeout_ms: 0, write_timeout_ms: 0, max_idle_conns: 0 },
      ],
    }))
  }

  const removeBackend = (index: number) => {
    setForm((current) => ({
      ...current,
      backends: (current.backends || []).filter((_, backendIndex) => backendIndex !== index),
    }))
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) {
      return
    }

    const submitToken = Symbol('route-form-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setSubmitting(true)
    try {
      // If TLS is enabled and user chose upload mode, create the certificate first.
      let certificateId = form.certificate_id || ''
      if (form.tls_enabled && tlsCertMode === 'upload') {
        const trimmedUploadName = uploadName.trim()
        const trimmedUploadDomain = uploadDomain.trim()
        if (!trimmedUploadName) {
          setError({ translationKey: 'form.uploadCertName' })
          setSubmitting(false)
          activeSubmitRef.current = null
          return
        }
        if (!trimmedUploadDomain) {
          setError({ translationKey: 'form.uploadCertDomain' })
          setSubmitting(false)
          activeSubmitRef.current = null
          return
        }
        if (!uploadCertPem.trim()) {
          setError({ translationKey: 'form.uploadCertPem' })
          setSubmitting(false)
          activeSubmitRef.current = null
          return
        }
        if (!uploadKeyPem.trim()) {
          setError({ translationKey: 'form.uploadKeyPem' })
          setSubmitting(false)
          activeSubmitRef.current = null
          return
        }
        const newCert = await certificatesApi.create({
          name: trimmedUploadName,
          domain: trimmedUploadDomain,
          source: 'imported',
          cert_pem: uploadCertPem.trim(),
          key_pem: uploadKeyPem.trim(),
        })
        certificateId = newCert.id
      }

      const normalizedBackends = (form.backends || [])
        .map((backend) => ({
          url: (backend.url || '').trim(),
          weight: backend.weight,
          ...(backend.dial_timeout_ms ? { dial_timeout_ms: backend.dial_timeout_ms } : {}),
          ...(backend.read_timeout_ms ? { read_timeout_ms: backend.read_timeout_ms } : {}),
          ...(backend.write_timeout_ms ? { write_timeout_ms: backend.write_timeout_ms } : {}),
          ...(backend.max_idle_conns ? { max_idle_conns: backend.max_idle_conns } : {}),
          ...(backend.rewrite_target ? { rewrite_target: backend.rewrite_target } : {}),
          ...(backend.redirect_code ? { redirect_code: backend.redirect_code } : {}),
        }))
        .filter((backend) => backend.url !== '')

      await onSubmit({
        ...form,
        host: normalizeHost(form.host),
        backend: (form.backend || '').trim(),
        backends: normalizedBackends,
        type: form.type || 'proxy',
        static_root: (form.static_root || '').trim(),
        static_spa: form.static_spa ?? false,
        tls_cert: (form.tls_cert || '').trim(),
        tls_key: (form.tls_key || '').trim(),
        rewrite_target: (form.rewrite_target || '').trim(),
        certificate_id: certificateId,
        // Header manipulation — strip empty entries
        set_request_headers: cleanHeaderMap(form.set_request_headers),
        remove_request_headers: cleanHeaderList(form.remove_request_headers),
        add_response_headers: cleanHeaderMap(form.add_response_headers),
        remove_response_headers: cleanHeaderList(form.remove_response_headers),
      })
    } catch (err) {
      setError(getLocalizedTextState(err))
    } finally {
      if (activeSubmitRef.current === submitToken) {
        activeSubmitRef.current = null
        setSubmitting(false)
      }
    }
  }

  const pathMatchModes = [
    {
      value: '',
      label: t('pathMatchModes.plainPrefix.label'),
      hint: t('pathMatchModes.plainPrefix.hint'),
    },
    {
      value: 'exact',
      label: t('pathMatchModes.exact.label'),
      hint: t('pathMatchModes.exact.hint'),
    },
    {
      value: 'stop',
      label: t('pathMatchModes.stop.label'),
      hint: t('pathMatchModes.stop.hint'),
    },
    {
      value: 'regex',
      label: t('pathMatchModes.regex.label'),
      hint: t('pathMatchModes.regex.hint'),
    },
    {
      value: 'regex_i',
      label: t('pathMatchModes.regexInsensitive.label'),
      hint: t('pathMatchModes.regexInsensitive.hint'),
    },
  ]
  const redirectCodeOptions = [
    { value: '0', label: t('form.noRedirect') },
    { value: '301', label: t('form.movedPermanently') },
    { value: '302', label: t('form.foundTemporary') },
  ]

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {errorMessage && <Alert variant="error">{errorMessage}</Alert>}

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.identityEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.identityDescription')}
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Select
            label={t('form.routeType')}
            value={form.type || 'proxy'}
            onChange={(event) => setForm({ ...form, type: event.target.value as 'proxy' | 'static' })}
            options={[
              { value: 'proxy', label: t('form.routeTypeProxy') },
              { value: 'static', label: t('form.routeTypeStatic') },
            ]}
            hint={form.type === 'static' ? t('form.routeTypeStaticHint') : t('form.routeTypeProxyHint')}
          />
          <Input
            label={t('form.name')}
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder={t('form.namePlaceholder')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.host')}
            value={form.host}
            onChange={(event) => setForm({ ...form, host: event.target.value })}
            placeholder={t('form.hostPlaceholder')}
            hint={t('form.hostHint')}
          />
          <Input
            label={t('form.pathPrefix')}
            value={form.path_prefix}
            onChange={(event) => setForm({ ...form, path_prefix: event.target.value })}
            placeholder={
              isRegexPathMatchMode
                ? t('form.pathPrefixRegexPlaceholder')
                : t('form.pathPrefixPlaceholder')
            }
            hint={isRegexPathMatchMode ? t('form.pathPrefixRegexHint') : t('form.pathPrefixHint')}
          />
        </div>

        {form.type === 'static' ? (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input
                label={t('form.staticRoot')}
                value={form.static_root || ''}
                onChange={(event) => setForm({ ...form, static_root: event.target.value })}
                placeholder={t('form.staticRootPlaceholder')}
                hint={t('form.staticRootHint')}
              />
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Switch
                label={t('form.staticSPA')}
                description={t('form.staticSPADescription')}
                checked={form.static_spa ?? false}
                onChange={(event) => setForm({ ...form, static_spa: event.target.checked })}
              />
            </div>
          </>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Input
              label={t('form.backend')}
              value={form.backend}
              onChange={(event) => setForm({ ...form, backend: event.target.value })}
              placeholder={t('form.backendPlaceholder')}
              hint={t('form.backendHint')}
            />
          </div>
        )}

        <Input
          label={t('form.priority')}
          type="number"
          value={form.priority}
          onChange={(event) => setForm({ ...form, priority: parseInt(event.target.value, 10) || 0 })}
          hint={t('form.priorityHint')}
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.timeoutMs')}
            type="number"
            min={0}
            value={form.timeout_ms || 0}
            onChange={(event) => setForm({ ...form, timeout_ms: parseInt(event.target.value, 10) || 0 })}
            hint={t('form.timeoutMsHint')}
          />
          <Input
            label={t('form.retryAttempts')}
            type="number"
            min={0}
            value={form.retry_attempts || 0}
            onChange={(event) => setForm({ ...form, retry_attempts: parseInt(event.target.value, 10) || 0 })}
            hint={t('form.retryAttemptsHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Select
            label={t('form.pathMatchMode')}
            value={form.path_match_mode || ''}
            onChange={(event) => setForm({ ...form, path_match_mode: event.target.value })}
            options={pathMatchModes.map(({ value, label }) => ({ value, label }))}
            hint={pathMatchModes.find((m) => m.value === (form.path_match_mode || ''))?.hint || pathMatchModes[0].hint}
          />
          <Input
            label={t('form.rewriteTarget')}
            value={form.rewrite_target || ''}
            onChange={(event) => setForm({ ...form, rewrite_target: event.target.value })}
            placeholder={t('form.rewriteTargetPlaceholder')}
            hint={t('form.rewriteTargetHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.headerName')}
            value={form.header_name || ''}
            onChange={(event) => setForm({ ...form, header_name: event.target.value })}
            placeholder={t('form.headerNamePlaceholder')}
            hint={t('form.headerNameHint')}
          />
          <Input
            label={t('form.headerValue')}
            value={form.header_value || ''}
            onChange={(event) => setForm({ ...form, header_value: event.target.value })}
            placeholder={t('form.headerValuePlaceholder')}
            hint={t('form.headerValueHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Select
            label={t('form.redirectCode')}
            value={String(form.redirect_code || 0)}
            onChange={(event) => setForm({ ...form, redirect_code: parseInt(event.target.value, 10) || 0 })}
            options={redirectCodeOptions}
            hint={t('form.redirectHint')}
          />
        </div>
      </Card>

      {/* Backend pool — only for proxy routes */}
      {form.type !== 'static' && (
      <Card tone="soft" className="space-y-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              {t('form.backendPoolEyebrow')}
            </div>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              {t('form.backendPoolDescription')}
            </p>
          </div>
          <Button variant="secondary" size="sm" onClick={addBackend}>
            {t('form.addBackend')}
          </Button>
        </div>

        {(form.backends || []).length === 0 ? (
          <div className="rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-5 text-sm text-[var(--text-muted)]">
            {t('form.noPooledBackends')}
          </div>
        ) : (
          <div className="space-y-3">
            {(form.backends || []).map((backend, index) => (
              <div
                key={`backend-${index}`}
                className="rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] p-4 shadow-[var(--shadow-sm)]"
              >
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div className="text-sm font-semibold text-[var(--text-primary)]">
                    {t('form.backendEntryTitle', { count: index + 1 })}
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeBackend(index)}
                    aria-label={t('form.removeBackend', { count: index + 1 })}
                  >
                    {t('form.removeBackendShort')}
                  </Button>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_140px]">
                  <Input
                    id={`backend-url-${index + 1}`}
                    label={t('form.backendUrlWithCount', { count: index + 1 })}
                    value={backend.url}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      url: event.target.value,
                    }))}
                    placeholder={t('form.backendPlaceholder')}
                  />
                  <Input
                    id={`backend-weight-${index + 1}`}
                    label={t('form.backendWeightWithCount', { count: index + 1 })}
                    type="number"
                    min={1}
                    value={backend.weight}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      weight: parseInt(event.target.value, 10) || 0,
                    }))}
                    hint={t('form.backendWeightHint')}
                  />
                </div>

                <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
                  <Input
                    id={`backend-dial-timeout-${index + 1}`}
                    label={t('form.backendDialTimeoutWithCount', { count: index + 1 })}
                    type="number"
                    min={0}
                    value={backend.dial_timeout_ms || 0}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      dial_timeout_ms: parseInt(event.target.value, 10) || 0,
                    }))}
                    hint={t('form.backendTimeoutHint')}
                  />
                  <Input
                    id={`backend-read-timeout-${index + 1}`}
                    label={t('form.backendReadTimeoutWithCount', { count: index + 1 })}
                    type="number"
                    min={0}
                    value={backend.read_timeout_ms || 0}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      read_timeout_ms: parseInt(event.target.value, 10) || 0,
                    }))}
                    hint={t('form.backendTimeoutHint')}
                  />
                  <Input
                    id={`backend-write-timeout-${index + 1}`}
                    label={t('form.backendWriteTimeoutWithCount', { count: index + 1 })}
                    type="number"
                    min={0}
                    value={backend.write_timeout_ms || 0}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      write_timeout_ms: parseInt(event.target.value, 10) || 0,
                    }))}
                    hint={t('form.backendTimeoutHint')}
                  />
                </div>

                <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
                  <Input
                    id={`backend-max-idle-conns-${index + 1}`}
                    label={t('form.backendMaxIdleConnsWithCount', { count: index + 1 })}
                    type="number"
                    min={0}
                    value={backend.max_idle_conns || 0}
                    onChange={(event) => updateBackend(index, (current) => ({
                      ...current,
                      max_idle_conns: parseInt(event.target.value, 10) || 0,
                    }))}
                    hint={t('form.backendMaxIdleConnsHint')}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
      )}

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.headersEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.headersDescription')}
          </p>
        </div>

        {/* Set Request Headers — key-value map */}
        <HeaderMapSection
          label={t('form.setRequestHeaders')}
          hint={t('form.setRequestHeadersHint')}
          entries={form.set_request_headers || {}}
          addLabel={t('form.addHeader')}
          keyPlaceholder={t('form.headerEntryKeyPlaceholder')}
          valuePlaceholder={t('form.headerEntryValuePlaceholder')}
          removeLabel={t('form.removeHeader')}
          emptyText={t('form.noHeadersConfigured')}
          onChange={(updated) => setForm({ ...form, set_request_headers: updated })}
          idPrefix="set-req-hdr"
        />

        {/* Remove Request Headers — string list */}
        <HeaderListSection
          label={t('form.removeRequestHeaders')}
          hint={t('form.removeRequestHeadersHint')}
          entries={form.remove_request_headers || []}
          addLabel={t('form.addHeaderName')}
          placeholder={t('form.headerEntryKeyPlaceholder')}
          removeLabel={t('form.removeHeader')}
          emptyText={t('form.noHeadersConfigured')}
          onChange={(updated) => setForm({ ...form, remove_request_headers: updated })}
          idPrefix="rm-req-hdr"
        />

        {/* Add Response Headers — key-value map */}
        <HeaderMapSection
          label={t('form.addResponseHeaders')}
          hint={t('form.addResponseHeadersHint')}
          entries={form.add_response_headers || {}}
          addLabel={t('form.addHeader')}
          keyPlaceholder={t('form.headerEntryKeyPlaceholder')}
          valuePlaceholder={t('form.headerEntryValuePlaceholder')}
          removeLabel={t('form.removeHeader')}
          emptyText={t('form.noHeadersConfigured')}
          onChange={(updated) => setForm({ ...form, add_response_headers: updated })}
          idPrefix="add-resp-hdr"
        />

        {/* Remove Response Headers — string list */}
        <HeaderListSection
          label={t('form.removeResponseHeaders')}
          hint={t('form.removeResponseHeadersHint')}
          entries={form.remove_response_headers || []}
          addLabel={t('form.addHeaderName')}
          placeholder={t('form.headerEntryKeyPlaceholder')}
          removeLabel={t('form.removeHeader')}
          emptyText={t('form.noHeadersConfigured')}
          onChange={(updated) => setForm({ ...form, remove_response_headers: updated })}
          idPrefix="rm-resp-hdr"
        />
      </Card>

      <Card tone="soft" className="space-y-4">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.tlsEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.tlsDescription')}
          </p>
        </div>

        <Switch
          label={t('form.tlsTermination')}
          checked={form.tls_enabled ?? false}
          onChange={(event) => setForm({ ...form, tls_enabled: event.target.checked })}
          aria-label={t('form.tlsTermination')}
        />

        {form.tls_enabled && (
          <>
            <Switch
              label={t('form.httpsRedirect')}
              checked={form.https_redirect ?? false}
              onChange={(event) => setForm({ ...form, https_redirect: event.target.checked })}
              aria-label={t('form.httpsRedirect')}
            />

            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setTlsCertMode('system')}
                className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  tlsCertMode === 'system'
                    ? 'bg-[var(--primary-500)] text-white'
                    : 'bg-[var(--bg-secondary)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                }`}
              >
                {t('form.tlsSourceSystem')}
              </button>
              <button
                type="button"
                onClick={() => setTlsCertMode('upload')}
                className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                  tlsCertMode === 'upload'
                    ? 'bg-[var(--primary-500)] text-white'
                    : 'bg-[var(--bg-secondary)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                }`}
              >
                {t('form.tlsSourceUpload')}
              </button>
            </div>

            {tlsCertMode === 'system' && (
              <div className="space-y-3">
                {route?.tls_enabled && route?.tls_cert && !route?.certificate_id && (
                  <div className="rounded-lg border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-3 text-sm text-[var(--text-muted)]">
                    {t('form.legacyCertPath', { path: route.tls_cert })}
                    <p className="mt-1 text-xs">{t('form.legacyCertPathHint')}</p>
                  </div>
                )}
                {activeCerts.length > 0 ? (
                  <Select
                    label={t('form.tlsSourceSystem')}
                    value={form.certificate_id || ''}
                    onChange={(event) => setForm({ ...form, certificate_id: event.target.value })}
                    options={[
                      { value: '', label: t('form.selectCertificate') },
                      ...activeCerts.map((cert) => ({
                        value: cert.id,
                        label: `${cert.name} (${cert.domain})`,
                      })),
                    ]}
                    hint={t('form.certificateSelectHint')}
                  />
                ) : (
                  <div className="rounded-lg border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-5 text-sm text-[var(--text-muted)]">
                    {t('form.noCertificates')}
                  </div>
                )}
              </div>
            )}

            {tlsCertMode === 'upload' && (
              <div className="space-y-4">
                <p className="text-xs text-[var(--text-muted)]">
                  {t('form.uploadCertHint')}
                </p>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <Input
                    label={t('form.uploadCertName')}
                    value={uploadName}
                    onChange={(event) => setUploadName(event.target.value)}
                    placeholder={t('form.uploadCertNamePlaceholder')}
                  />
                  <Input
                    label={t('form.uploadCertDomain')}
                    value={uploadDomain}
                    onChange={(event) => setUploadDomain(event.target.value)}
                    placeholder={form.host || 'example.com'}
                    hint={t('form.uploadCertDomainHint')}
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="upload-cert-pem" className="block text-xs font-semibold text-[var(--text-muted)]">
                    {t('form.uploadCertPem')}
                  </label>
                  <textarea
                    id="upload-cert-pem"
                    value={uploadCertPem}
                    onChange={(event) => setUploadCertPem(event.target.value)}
                    placeholder={t('form.uploadCertPemPlaceholder')}
                    rows={5}
                    className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--primary-500)] focus:outline-none focus:ring-1 focus:ring-[var(--primary-500)]"
                  />
                  <p className="text-xs text-[var(--text-muted)]">{t('form.uploadCertPemHint')}</p>
                </div>
                <div className="space-y-2">
                  <label htmlFor="upload-key-pem" className="block text-xs font-semibold text-[var(--text-muted)]">
                    {t('form.uploadKeyPem')}
                  </label>
                  <textarea
                    id="upload-key-pem"
                    value={uploadKeyPem}
                    onChange={(event) => setUploadKeyPem(event.target.value)}
                    placeholder={t('form.uploadKeyPemPlaceholder')}
                    rows={5}
                    className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--primary-500)] focus:outline-none focus:ring-1 focus:ring-[var(--primary-500)]"
                  />
                  <p className="text-xs text-[var(--text-muted)]">{t('form.uploadKeyPemHint')}</p>
                </div>
              </div>
            )}
          </>
        )}
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Switch
          label={t('form.stripPrefix')}
          description={t('form.stripPrefixDescription')}
          checked={form.strip_prefix}
          onChange={(event) => setForm({ ...form, strip_prefix: event.target.checked })}
        />
        <Switch
          label={t('form.enabled')}
          description={t('form.enabledDescription')}
          checked={form.enabled}
          onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
        />
      </div>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto" loading={submitting}>
          {route ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
