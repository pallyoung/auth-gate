import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { Alert, Button, Card, Input } from './ui'
import type { CertificateInput } from '../lib/api/types'

interface CertificateFormProps {
  onSubmit: (data: CertificateInput) => Promise<void> | void
  onCancel: () => void
}

type FormMode = 'local_ca' | 'imported'

export function CertificateForm({ onSubmit, onCancel }: CertificateFormProps) {
  const { t } = useTranslation('certificates')
  const [mode, setMode] = React.useState<FormMode>('local_ca')
  const [name, setName] = React.useState('')
  const [domain, setDomain] = React.useState('')
  const [days, setDays] = React.useState('')
  const [certPem, setCertPem] = React.useState('')
  const [keyPem, setKeyPem] = React.useState('')
  const [organization, setOrganization] = React.useState('')
  const [organizationalUnit, setOrganizationalUnit] = React.useState('')
  const [country, setCountry] = React.useState('')
  const [province, setProvince] = React.useState('')
  const [locality, setLocality] = React.useState('')
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const errorMessage = resolveLocalizedText(t, error)

  const domainRegex = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) return

    const submitToken = Symbol('certificate-form-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setSubmitting(true)

    try {
      const trimmedName = name.trim()
      const trimmedDomain = domain.trim()

      if (!domainRegex.test(trimmedDomain)) {
        setError({ translationKey: 'form.invalidDomain' })
        return
      }

      if (mode === 'imported') {
        if (!certPem.trim()) {
          setError({ translationKey: 'form.certPemRequired' })
          return
        }
        if (!keyPem.trim()) {
          setError({ translationKey: 'form.keyPemRequired' })
          return
        }
      }

      const input: CertificateInput = {
        name: trimmedName,
        domain: trimmedDomain,
      }

      if (mode === 'imported') {
        input.source = 'imported'
        input.cert_pem = certPem.trim()
        input.key_pem = keyPem.trim()
      } else {
        const parsedDays = parseInt(days, 10)
        if (!isNaN(parsedDays) && parsedDays > 0) {
          input.days = parsedDays
        }
        const trimmedOrg = organization.trim()
        const trimmedOU = organizationalUnit.trim()
        const trimmedCountry = country.trim()
        const trimmedProvince = province.trim()
        const trimmedLocality = locality.trim()
        if (trimmedOrg) input.organization = trimmedOrg
        if (trimmedOU) input.organizational_unit = trimmedOU
        if (trimmedCountry) input.country = trimmedCountry
        if (trimmedProvince) input.province = trimmedProvince
        if (trimmedLocality) input.locality = trimmedLocality
      }

      await onSubmit(input)
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

      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setMode('local_ca')}
          className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
            mode === 'local_ca'
              ? 'bg-[var(--primary-500)] text-white'
              : 'bg-[var(--bg-secondary)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
          }`}
        >
          {t('form.localCaTab')}
        </button>
        <button
          type="button"
          onClick={() => setMode('imported')}
          className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
            mode === 'imported'
              ? 'bg-[var(--primary-500)] text-white'
              : 'bg-[var(--bg-secondary)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
          }`}
        >
          {t('form.importTab')}
        </button>
      </div>

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
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t('form.namePlaceholder')}
            required
            hint={t('form.nameHint')}
          />
          <Input
            label={t('form.domain')}
            value={domain}
            onChange={(event) => setDomain(event.target.value)}
            placeholder={t('form.domainPlaceholder')}
            required
            hint={t('form.domainHint')}
          />
          {mode === 'local_ca' && (
            <Input
              label={t('form.validityDays')}
              type="number"
              value={days}
              onChange={(event) => setDays(event.target.value)}
              placeholder={t('form.validityDaysPlaceholder')}
              min={1}
              max={3650}
              hint={t('form.validityDaysHint')}
            />
          )}
        </div>
      </Card>

      {mode === 'local_ca' && (
        <Card tone="soft" className="space-y-5">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              {t('form.subjectEyebrow')}
            </div>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              {t('form.subjectDescription')}
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Input
              label={t('form.organization')}
              value={organization}
              onChange={(event) => setOrganization(event.target.value)}
              placeholder={t('form.organizationPlaceholder')}
              hint={t('form.subjectHint')}
            />
            <Input
              label={t('form.organizationalUnit')}
              value={organizationalUnit}
              onChange={(event) => setOrganizationalUnit(event.target.value)}
              placeholder={t('form.organizationalUnitPlaceholder')}
              hint={t('form.subjectHint')}
            />
          </div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <Input
              label={t('form.country')}
              value={country}
              onChange={(event) => setCountry(event.target.value)}
              placeholder={t('form.countryPlaceholder')}
              maxLength={2}
              hint={t('form.subjectHint')}
            />
            <Input
              label={t('form.province')}
              value={province}
              onChange={(event) => setProvince(event.target.value)}
              placeholder={t('form.provincePlaceholder')}
              hint={t('form.subjectHint')}
            />
            <Input
              label={t('form.locality')}
              value={locality}
              onChange={(event) => setLocality(event.target.value)}
              placeholder={t('form.localityPlaceholder')}
              hint={t('form.subjectHint')}
            />
          </div>
        </Card>
      )}

      {mode === 'imported' && (
        <Card tone="soft" className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="cert-pem" className="block text-xs font-semibold text-[var(--text-muted)]">
              {t('form.certPem')}
            </label>
            <textarea
              id="cert-pem"
              value={certPem}
              onChange={(event) => setCertPem(event.target.value)}
              placeholder={t('form.certPemPlaceholder')}
              rows={5}
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--primary-500)] focus:outline-none focus:ring-1 focus:ring-[var(--primary-500)]"
            />
            <p className="text-xs text-[var(--text-muted)]">{t('form.certPemHint')}</p>
          </div>
          <div className="space-y-2">
            <label htmlFor="key-pem" className="block text-xs font-semibold text-[var(--text-muted)]">
              {t('form.keyPem')}
            </label>
            <textarea
              id="key-pem"
              value={keyPem}
              onChange={(event) => setKeyPem(event.target.value)}
              placeholder={t('form.keyPemPlaceholder')}
              rows={5}
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--primary-500)] focus:outline-none focus:ring-1 focus:ring-[var(--primary-500)]"
            />
            <p className="text-xs text-[var(--text-muted)]">{t('form.keyPemHint')}</p>
          </div>
        </Card>
      )}

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
