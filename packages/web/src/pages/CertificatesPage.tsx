import React from 'react'
import { FileKey, Key, Plus, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { CertificateForm } from '../components/CertificateForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { certificatesApi } from '../lib/api/certificates'
import { getSessionUser } from '../lib/session-store'
import type { Certificate as CertificateType } from '../lib/api/types'

const renewableDNSProviders = new Set(['cloudflare', 'route53'])

export function CertificatesPage() {
  const { t, i18n } = useTranslation('certificates')
  const sessionUser = getSessionUser()
  const [certificates, setCertificates] = React.useState<CertificateType[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [certificateListUnavailable, setCertificateListUnavailable] = React.useState(false)
  const [certificateDirectoryUnavailable, setCertificateDirectoryUnavailable] = React.useState(false)
  const [showForm, setShowForm] = React.useState(false)
  const [renewingIds, setRenewingIds] = React.useState<string[]>([])
  const activeRenewIdsRef = React.useRef(new Set<string>())
  const requestGenerationRef = React.useRef(0)
  const certificatesEnabled = sessionUser?.features?.certificates === true
  const canManageCertificates =
    certificatesEnabled && (sessionUser?.permissions?.can_manage_routes ?? false)
  const showDirectoryMetrics = !certificateListUnavailable
  const errorMessage = resolveLocalizedText(t, error)

  const getErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (!(err instanceof ApiError)) {
      return { translationKey: 'errors.network' }
    }

    switch (err.code) {
      case 'unauthorized':
      case 'invalid_token':
        return { translationKey: 'errors.unauthorized' }
      case 'insufficient_permissions':
        return { translationKey: 'errors.insufficientPermissions' }
      case 'cert_not_found':
        return { translationKey: 'errors.certNotFound' }
      case 'invalid_name':
        return { translationKey: 'errors.invalidName' }
      case 'invalid_domain':
        return { translationKey: 'errors.invalidDomain' }
      case 'domain_exists':
        return { translationKey: 'errors.domainExists' }
      case 'invalid_provider':
        return { translationKey: 'errors.invalidProvider' }
      case 'dns_provider_error':
        return { translationKey: 'errors.dnsProvider' }
      case 'acme_error':
        return { translationKey: 'errors.acme' }
      case 'cert_not_active':
        return { translationKey: 'errors.certNotActive' }
      case 'database_error':
        return { translationKey: 'errors.certificateStoreFailure' }
      default:
        return { message: err.message }
    }
  }, [])

  const getListErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (err instanceof ApiError && err.code === 'database_error') {
      return { translationKey: 'errors.certificateDirectoryUnavailable' }
    }

    return getErrorState(err)
  }, [getErrorState])

  const fetchData = React.useCallback(async () => {
    const requestGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = requestGeneration

    try {
      setError(null)
      setCertificateListUnavailable(false)
      setCertificateDirectoryUnavailable(false)
      const data = await certificatesApi.list()
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      setCertificates(data)
    } catch (e) {
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      setCertificates([])
      setCertificateListUnavailable(true)
      setCertificateDirectoryUnavailable(e instanceof ApiError && e.code === 'database_error')
      setError(getListErrorState(e))
    } finally {
      if (requestGenerationRef.current === requestGeneration) {
        setLoading(false)
      }
    }
  }, [getListErrorState])

  React.useEffect(() => {
    if (!certificatesEnabled) {
      setLoading(false)
      return
    }

    fetchData()
  }, [certificatesEnabled, fetchData])

  const handleCreate = async (data: { name: string; domain: string; dns_provider: string; provider_config: Record<string, string> }) => {
    try {
      await certificatesApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (e) {
      throw new LocalizedError(getErrorState(e))
    }
  }

  const handleDelete = async (cert: CertificateType) => {
    if (!confirm(t('page.deleteConfirm', { domain: cert.domain }))) return
    try {
      await certificatesApi.delete(cert.id)
      await fetchData()
    } catch (e) {
      setError(getErrorState(e))
    }
  }

  const handleRenew = async (cert: CertificateType) => {
    if (activeRenewIdsRef.current.has(cert.id)) {
      return
    }
    if (!confirm(t('page.renewConfirm', { domain: cert.domain }))) return
    try {
      activeRenewIdsRef.current.add(cert.id)
      setRenewingIds((current) => (
        current.includes(cert.id) ? current : [...current, cert.id]
      ))
      await certificatesApi.renew(cert.id)
      await fetchData()
    } catch (e) {
      setError(getErrorState(e))
    } finally {
      activeRenewIdsRef.current.delete(cert.id)
      setRenewingIds((current) => current.filter((id) => id !== cert.id))
    }
  }

  const activeCount = certificates.filter((c) => c.status === 'active').length
  const pendingCount = certificates.filter((c) => c.status === 'pending' || c.status === 'renewing').length
  const failedCount = certificates.filter((c) => c.status === 'failed').length

  const statusVariant = (status: string) => {
    switch (status) {
      case 'active':
        return 'success'
      case 'pending':
      case 'renewing': return 'warning'
      case 'failed':
        return 'error'
      default:
        return 'default'
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return t('page.noDate')
    return new Intl.DateTimeFormat(i18n.resolvedLanguage === 'zh-CN' ? 'zh-CN' : 'en-US', {
      year: 'numeric',
      month: i18n.resolvedLanguage === 'zh-CN' ? 'long' : 'short',
      day: 'numeric',
    }).format(new Date(dateStr))
  }

  const statusLabel = (status: string) => t(`status.${status}` as const)
  const providerLabel = (provider?: string) => {
    if (!provider) return t('page.noDate')
    return t(`providers.${provider}` as any, { defaultValue: provider })
  }

  const canRenewCertificate = (certificate: CertificateType) =>
    certificate.status === 'active' && renewableDNSProviders.has(certificate.dns_provider)

  const columns = [
    {
      key: 'name',
      header: t('page.certificate'),
      render: (value: string, row: CertificateType) => (
        <div>
          <div className="font-semibold text-[var(--text-primary)]">{value}</div>
          <div className="mt-1 font-mono text-xs text-[var(--text-muted)]">{row.domain}</div>
        </div>
      ),
    },
    {
      key: 'status',
      header: t('page.status'),
      className: 'w-28',
      render: (value: string) => <Badge variant={statusVariant(value) as any} badgeSize="sm">{statusLabel(value)}</Badge>,
    },
    {
      key: 'not_after',
      header: t('page.expires'),
      className: 'w-36',
      render: (value: string) => {
        if (!value) return t('page.noDate')
        const date = new Date(value)
        const now = new Date()
        const daysLeft = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
        return (
          <div>
            <div className="text-sm">{formatDate(value)}</div>
            {daysLeft > 0 && daysLeft <= 30 && (
              <div className="text-xs text-[var(--accent-500)]">{t('page.daysLeft', { count: daysLeft })}</div>
            )}
          </div>
        )
      },
    },
    {
      key: 'dns_provider',
      header: t('page.dnsProvider'),
      className: 'w-28',
      render: (value: string) => <Badge variant="default" badgeSize="sm">{providerLabel(value)}</Badge>,
    },
    {
      key: 'created_at',
      header: t('page.created'),
      className: 'w-32',
      render: (value: string) => <span className="text-sm text-[var(--text-muted)]">{formatDate(value)}</span>,
    },
  ]

  if (!certificatesEnabled) {
    return (
      <div className="animate-rise-in">
        <PageHeader
          eyebrow={t('page.eyebrow')}
          title={t('page.title')}
          description={t('page.description')}
          meta={<Badge variant="default">{t('page.badge')}</Badge>}
        />

        <Card padding="lg">
          <EmptyState
            icon={<FileKey className="h-8 w-8" />}
            title={t('page.disabledTitle')}
            description={t('page.disabledDescription')}
          />
        </Card>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          {t('page.loading')}
        </div>
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow={t('page.eyebrow')}
        title={t('page.title')}
        description={t('page.description')}
        meta={
          <>
            <Badge variant="primary">{t('page.badge')}</Badge>
            {showDirectoryMetrics ? (
              <span className="text-sm text-[var(--text-muted)]">{t('page.count', { count: certificates.length })}</span>
            ) : null}
          </>
        }
        action={
          canManageCertificates && !certificateListUnavailable ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => setShowForm(true)}>
              {t('page.provision')}
            </Button>
          ) : null
        }
      />

      {errorMessage && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {errorMessage}
        </Alert>
      )}

      {showDirectoryMetrics ? (
        <div className="mb-6 grid gap-4 md:grid-cols-3">
          <MetricCard
            label={t('page.activeCertificates')}
            value={activeCount}
            hint={t('page.activeCertificatesHint')}
            icon={<FileKey className="h-5 w-5" />}
            tone="primary"
          />
          <MetricCard
            label={t('page.inProgress')}
            value={pendingCount}
            hint={t('page.inProgressHint')}
            icon={<RefreshCw className="h-5 w-5" />}
            tone="warning"
          />
          <MetricCard
            label={t('page.failed')}
            value={failedCount}
            hint={t('page.failedHint')}
            icon={<Key className="h-5 w-5" />}
            tone="error"
          />
        </div>
      ) : null}

      <Card padding="lg" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('page.registryEyebrow')}
          </div>
          <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
            {t('page.registryTitle')}
          </h2>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('page.registryDescription')}
          </p>
        </div>

        {certificates.length === 0 ? (
          <EmptyState
            icon={<FileKey className="h-8 w-8" />}
            title={
              certificateDirectoryUnavailable
                ? t('page.directoryUnavailableTitle')
                : certificateListUnavailable
                ? t('page.listUnavailableTitle')
                : t('page.emptyTitle')
            }
            description={
              certificateDirectoryUnavailable
                ? t('page.directoryUnavailableDescription')
                : certificateListUnavailable
                ? t('page.listUnavailableDescription')
                : canManageCertificates
                ? t('page.emptyDescription')
                : t('page.readOnlyEmptyDescription')
            }
            action={
              canManageCertificates && !certificateListUnavailable
                ? <Button onClick={() => setShowForm(true)}>{t('page.provisionFirst')}</Button>
                : undefined
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={certificates}
            onDelete={canManageCertificates ? handleDelete : undefined}
            extraActions={(row: CertificateType) => (
              canManageCertificates && canRenewCertificate(row) ? (
                <button
                  type="button"
                  onClick={() => handleRenew(row)}
                  disabled={renewingIds.includes(row.id)}
                  className="flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm text-[var(--primary-600)] transition-colors hover:bg-[var(--bg-hover)] disabled:opacity-50"
                >
                  <RefreshCw className={`h-4 w-4 ${renewingIds.includes(row.id) ? 'animate-spin' : ''}`} />
                  {renewingIds.includes(row.id) ? t('page.renewing') : t('page.renew')}
                </button>
              ) : null
            )}
          />
        )}
      </Card>

      <Modal
        open={canManageCertificates && showForm}
        onClose={() => setShowForm(false)}
        title={t('page.modalTitle')}
      >
        <CertificateForm
          onSubmit={handleCreate}
          onCancel={() => setShowForm(false)}
        />
      </Modal>
    </div>
  )
}
