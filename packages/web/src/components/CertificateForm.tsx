import React from 'react'
import { Alert, Button, Card, Input, Select } from './ui'

interface CertificateFormProps {
  onSubmit: (data: { name: string; domain: string; dns_provider: string; provider_config: Record<string, string> }) => Promise<void> | void
  onCancel: () => void
}

const dnsProviderOptions = [
  { value: 'cloudflare', label: 'CloudFlare' },
  { value: 'route53', label: 'AWS Route53' },
  { value: 'manual', label: 'Manual (DIY)' },
]

export function CertificateForm({ onSubmit, onCancel }: CertificateFormProps) {
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

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setSubmitting(true)

    // Validate domain
    const domainRegex = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/
    if (!domainRegex.test(form.domain)) {
      setError('Invalid domain format. Use something like "example.com" or "*.example.com"')
      setSubmitting(false)
      return
    }

    // Validate provider config
    if (form.dns_provider === 'cloudflare' && !providerConfig.api_token) {
      setError('CloudFlare API token is required')
      setSubmitting(false)
      return
    }
    if (form.dns_provider === 'route53' && (!providerConfig.access_key_id || !providerConfig.secret_access_key)) {
      setError('AWS Access Key ID and Secret Access Key are required for Route53')
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
            Certificate Details
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Provide a name and domain for the certificate. Wildcard certificates use "*.example.com" format.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Certificate Name"
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder="My Wildcard Cert"
            required
            hint="A friendly name to identify this certificate"
          />
          <Input
            label="Domain"
            value={form.domain}
            onChange={(event) => setForm({ ...form, domain: event.target.value })}
            placeholder="*.example.com"
            required
            hint="Use *.example.com for wildcard certificates"
          />
        </div>
      </Card>

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            DNS Provider
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Select how DNS validation will be performed. CloudFlare and Route53 support automatic DNS-01 challenge.
          </p>
        </div>

        <Select
          label="DNS Provider"
          value={form.dns_provider}
          onChange={(event) => handleDnsProviderChange(event.target.value)}
          options={dnsProviderOptions}
          required
        />

        {form.dns_provider === 'cloudflare' && (
          <Input
            label="CloudFlare API Token"
            type="password"
            value={providerConfig.api_token || ''}
            onChange={(event) => updateProviderConfig('api_token', event.target.value)}
            placeholder="cf_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            required
            hint="Create an API token in CloudFlare dashboard with Zone:DNS:Edit permission"
          />
        )}

        {form.dns_provider === 'route53' && (
          <div className="space-y-4">
            <Input
              label="AWS Access Key ID"
              value={providerConfig.access_key_id || ''}
              onChange={(event) => updateProviderConfig('access_key_id', event.target.value)}
              placeholder="AKIAXXXXXXXXXXXXXXXX"
              required
            />
            <Input
              label="AWS Secret Access Key"
              type="password"
              value={providerConfig.secret_access_key || ''}
              onChange={(event) => updateProviderConfig('secret_access_key', event.target.value)}
              placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
              required
            />
            <Input
              label="AWS Region (optional)"
              value={providerConfig.region || ''}
              onChange={(event) => updateProviderConfig('region', event.target.value)}
              placeholder="us-east-1"
              hint="Leave blank to use default region"
            />
          </div>
        )}

        {form.dns_provider === 'manual' && (
          <div className="rounded-[18px] border border-[var(--warning-500)]/30 bg-[var(--warning-500)]/10 px-4 py-4">
            <p className="text-sm text-[var(--text-primary)]">
              <strong>Manual Mode:</strong> You will need to manually create DNS TXT records when prompted. This is useful for testing or when you don't have API access to your DNS provider.
            </p>
          </div>
        )}
      </Card>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto" disabled={submitting}>
          {submitting ? 'Provisioning...' : 'Provision Certificate'}
        </Button>
      </div>
    </form>
  )
}