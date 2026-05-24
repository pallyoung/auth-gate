import React from 'react'
import { useTranslation } from 'react-i18next'
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
  const [error, setError] = React.useState('')
  const [submitting, setSubmitting] = React.useState(false)

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

  const dnsProviderOptions = [
    { value: 'cloudflare', label: t('providers.cloudflare') },
    { value: 'route53', label: t('providers.route53') },
    { value: 'manual', label: t('providers.manual') },
  ]

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setSubmitting(true)

    // Validate domain
    const domainRegex = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/
    if (!domainRegex.test(form.domain)) {
      setError(t('form.invalidDomain'))
      setSubmitting(false)
      return
    }

    // Validate provider config
    if (form.dns_provider === 'cloudflare' && !providerConfig.api_token) {
      setError(t('form.cloudflareRequired'))
      setSubmitting(false)
      return
    }
    if (form.dns_provider === 'route53' && (!providerConfig.access_key_id || !providerConfig.secret_access_key)) {
      setError(t('form.route53Required'))
      setSubmitting(false)
      return
    }

    try {
      await onSubmit({
        name: form.name,
        domain: form.domain,
        dns_provider: form.dns_provider,
        provider_config: providerConfig,
      })
    } catch (e) {
      setError((e as Error).message)
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {error && <Alert variant="error">{error}</Alert>}

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

        {form.dns_provider === 'manual' && (
          <div className="rounded-[18px] border border-[var(--warning-500)]/30 bg-[var(--warning-500)]/10 px-4 py-4">
            <p className="text-sm text-[var(--text-primary)]">
              <strong>{t('form.manualTitle')}</strong> {t('form.manualDescription')}
            </p>
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
