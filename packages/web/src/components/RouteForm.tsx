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
    tls_cert: route?.tls_cert || '',
    tls_key: route?.tls_key || '',
    tls_enabled: route?.tls_enabled ?? false,
    certificate_id: route?.certificate_id || '',
    timeout_ms: route?.timeout_ms || 0,
    retry_attempts: route?.retry_attempts || 0,
    path_match_mode: route?.path_match_mode || '',
    rewrite_target: route?.rewrite_target || '',
    redirect_code: route?.redirect_code || 0,
  }
}

function normalizeHost(host: string | undefined) {
  const trimmed = (host || '').trim().toLowerCase()
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    return trimmed.slice(1, -1)
  }
  return trimmed
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
        tls_cert: (form.tls_cert || '').trim(),
        tls_key: (form.tls_key || '').trim(),
        rewrite_target: (form.rewrite_target || '').trim(),
        certificate_id: certificateId,
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
          <Input
            label={t('form.name')}
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder={t('form.namePlaceholder')}
          />
          <Input
            label={t('form.host')}
            value={form.host}
            onChange={(event) => setForm({ ...form, host: event.target.value })}
            placeholder={t('form.hostPlaceholder')}
            hint={t('form.hostHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
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
          <Input
            label={t('form.backend')}
            value={form.backend}
            onChange={(event) => setForm({ ...form, backend: event.target.value })}
            placeholder={t('form.backendPlaceholder')}
            hint={t('form.backendHint')}
          />
        </div>

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
          <Select
            label={t('form.redirectCode')}
            value={String(form.redirect_code || 0)}
            onChange={(event) => setForm({ ...form, redirect_code: parseInt(event.target.value, 10) || 0 })}
            options={redirectCodeOptions}
            hint={t('form.redirectHint')}
          />
        </div>
      </Card>

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
