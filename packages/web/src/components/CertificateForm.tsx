import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { Alert, Button, Card, Input, Select } from './ui'

interface CertificateFormProps {
  onSubmit: (data: { name: string; domain: string; dns_provider: string; provider_config: Record<string, string> }) => Promise<void> | void
  onCancel: () => void
}

export function CertificateForm({ onSubmit, onCancel }: CertificateFormProps) {
  const { t } = useTranslation('certificates')
  const [form, setForm] = React.useState({
    name: '',
    domain: '',
    dns_provider: 'cloudflare',
  })
  const [providerConfig, setProviderConfig] = React.useState<Record<string, string>>({})
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const errorMessage = resolveLocalizedText(t, error)

  const handleDnsProviderChange = (value: string) => {
    setForm({ ...form, dns_provider: value })
    setProviderConfig({})

    if (value === 'route53') {
      setProviderConfig({
        access_key_id: '',
        secret_access_key: '',
        region: '',
      })
    }
  }

  const updateProviderConfig = (key: string, value: string) => {
    setProviderConfig((current) => ({ ...current, [key]: value }))
  }

  const normalizeProviderConfig = React.useCallback((config: Record<string, string>) => {
    const normalized = Object.fromEntries(
      Object.entries(config).map(([key, value]) => [key, value.trim()])
    )
    return normalized
  }, [])

  const dnsProviderOptions = [
    { value: 'cloudflare', label: t('providers.cloudflare') },
    { value: 'route53', label: t('providers.route53') },
  ]

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) {
      return
    }

    const submitToken = Symbol('certificate-form-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setSubmitting(true)

    try {
      const normalizedDomain = form.domain.trim()
      const normalizedProviderConfig = normalizeProviderConfig(providerConfig)

      // Validate domain
      const domainRegex = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/
      if (!domainRegex.test(normalizedDomain)) {
        setError({ translationKey: 'form.invalidDomain' })
        return
      }

      // Validate provider config
      if (form.dns_provider === 'cloudflare' && !normalizedProviderConfig.api_token) {
        setError({ translationKey: 'form.cloudflareRequired' })
        return
      }
      if (
        form.dns_provider === 'route53' &&
        (!normalizedProviderConfig.access_key_id || !normalizedProviderConfig.secret_access_key)
      ) {
        setError({ translationKey: 'form.route53Required' })
        return
      }

      await onSubmit({
        name: form.name,
        domain: normalizedDomain,
        dns_provider: form.dns_provider,
        provider_config: normalizedProviderConfig,
      })
    } catch (e) {
      setError(getLocalizedTextState(e))
    } finally {
      if (activeSubmitRef.current === submitToken) {
        activeSubmitRef.current = null
        setSubmitting(false)
      }
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {errorMessage && <Alert variant="error">{errorMessage}</Alert>}

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.detailsEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.detailsDescription')}
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.name')}
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder={t('form.namePlaceholder')}
            required
            hint={t('form.nameHint')}
          />
          <Input
            label={t('form.domain')}
            value={form.domain}
            onChange={(event) => setForm({ ...form, domain: event.target.value })}
            placeholder={t('form.domainPlaceholder')}
            required
            hint={t('form.domainHint')}
          />
        </div>
      </Card>

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.providerEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.providerDescription')}
          </p>
        </div>

        <Select
          label={t('form.provider')}
          value={form.dns_provider}
          onChange={(event) => handleDnsProviderChange(event.target.value)}
          options={dnsProviderOptions}
          required
        />

        {form.dns_provider === 'cloudflare' && (
          <Input
            label={t('form.cloudflareToken')}
            type="password"
            value={providerConfig.api_token || ''}
            onChange={(event) => updateProviderConfig('api_token', event.target.value)}
            placeholder="cf_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            required
            hint={t('form.cloudflareTokenHint')}
          />
        )}

        {form.dns_provider === 'route53' && (
          <div className="space-y-4">
            <Input
              label={t('form.route53AccessKey')}
              value={providerConfig.access_key_id || ''}
              onChange={(event) => updateProviderConfig('access_key_id', event.target.value)}
              placeholder="AKIAXXXXXXXXXXXXXXXX"
              required
            />
            <Input
              label={t('form.route53Secret')}
              type="password"
              value={providerConfig.secret_access_key || ''}
              onChange={(event) => updateProviderConfig('secret_access_key', event.target.value)}
              placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
              required
            />
            <Input
              label={t('form.route53Region')}
              value={providerConfig.region || ''}
              onChange={(event) => updateProviderConfig('region', event.target.value)}
              placeholder="us-east-1"
              hint={t('form.route53RegionHint')}
            />
          </div>
        )}
      </Card>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto" disabled={submitting}>
          {submitting ? t('form.provisioningButton') : t('form.provisionButton')}
        </Button>
      </div>
    </form>
  )
}
