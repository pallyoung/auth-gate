import React from 'react'
import { FileKey, Key, Plus, RefreshCw, ShieldCheck } from 'lucide-react'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { CertificateForm } from '../components/CertificateForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { certificatesApi } from '../lib/api/certificates'
import { getSessionUser } from '../lib/session-store'
import type { Certificate as CertificateType } from '../lib/api/types'

export function CertificatesPage() {
  const [certificates, setCertificates] = React.useState<CertificateType[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [showForm, setShowForm] = React.useState(false)
  const [renewingId, setRenewingId] = React.useState<string | null>(null)
  const canManageCertificates = getSessionUser()?.permissions?.can_manage_routes !== false

  const fetchData = React.useCallback(async () => {
    try {
      setError('')
      const data = await certificatesApi.list()
      setCertificates(data)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleCreate = async (data: { name: string; domain: string; dns_provider: string; provider_config: Record<string, string> }) => {
    try {
      await certificatesApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleDelete = async (cert: CertificateType) => {
    if (!confirm(`Delete certificate for "${cert.domain}"? This will also remove the certificate files.`)) return
    try {
      await certificatesApi.delete(cert.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleRenew = async (cert: CertificateType) => {
    if (!confirm(`Renew certificate for "${cert.domain}"? This will take a few minutes.`)) return
    try {
      setRenewingId(cert.id)
      await certificatesApi.renew(cert.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setRenewingId(null)
    }
  }

  const activeCount = certificates.filter(c => c.status === 'active').length
  const pendingCount = certificates.filter(c => c.status === 'pending' || c.status === 'renewing').length
  const failedCount = certificates.filter(c => c.status === 'failed').length

  const statusVariant = (status: string) => {
    switch (status) {
      case 'active': return 'success'
      case 'pending':
      case 'renewing': return 'warning'
      case 'failed': return 'error'
      default: return 'default'
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
  }

  const columns = [
    {
      key: 'name',
      header: 'Certificate',
      render: (value: string, row: CertificateType) => (
        <div>
          <div className="font-semibold text-[var(--text-primary)]">{value}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)] font-mono">{row.domain}</div>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      className: 'w-28',
      render: (value: string) => <Badge variant={statusVariant(value) as any} badgeSize="sm">{value}</Badge>,
    },
    {
      key: 'not_after',
      header: 'Expires',
      className: 'w-36',
      render: (value: string) => {
        if (!value) return '-'
        const date = new Date(value)
        const now = new Date()
        const daysLeft = Math.ceil((date.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
        return (
          <div>
            <div className="text-sm">{formatDate(value)}</div>
            {daysLeft > 0 && daysLeft <= 30 && (
              <div className="text-xs text-[var(--accent-500)]">{daysLeft} days left</div>
            )}
          </div>
        )
      },
    },
    {
      key: 'dns_provider',
      header: 'DNS Provider',
      className: 'w-28',
      render: (value: string) => <Badge variant="default" badgeSize="sm">{value}</Badge>,
    },
    {
      key: 'created_at',
      header: 'Created',
      className: 'w-32',
      render: (value: string) => <span className="text-sm text-[var(--text-muted)]">{formatDate(value)}</span>,
    },
  ]

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          Loading certificates...
        </div>
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="TLS Automation"
        title="Certificates"
        description="Provision and manage SSL certificates via Let's Encrypt ACME with automatic DNS-01 validation."
        meta={
          <>
            <Badge variant="primary">ACME</Badge>
            <span className="text-sm text-[var(--text-muted)]">{certificates.length} certificates</span>
          </>
        }
        action={
          canManageCertificates ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => setShowForm(true)}>
              Provision Certificate
            </Button>
          ) : null
        }
      />

      {error && (
        <Alert variant="error" title="Certificate operation failed" className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard label="Active Certificates" value={activeCount} hint="Valid certificates ready for use." icon={<FileKey className="h-5 w-5" />} tone="primary" />
        <MetricCard label="In Progress" value={pendingCount} hint="Certificate provisioning or renewal in progress." icon={<RefreshCw className="h-5 w-5" />} tone="warning" />
        <MetricCard label="Failed" value={failedCount} hint="Certificates that encountered errors." icon={<Key className="h-5 w-5" />} tone="error" />
      </div>

      <Card padding="lg" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Certificate Registry
          </div>
          <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
            ACME provisioned certificates
          </h2>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Certificates are automatically renewed 30 days before expiration using DNS-01 challenge for wildcard support.
          </p>
        </div>

        {certificates.length === 0 ? (
          <EmptyState
            icon={<FileKey className="h-8 w-8" />}
            title="No certificates provisioned"
            description="Provision your first certificate to enable HTTPS for your routes. Wildcard certificates are supported via DNS-01 challenge."
            action={canManageCertificates ? <Button onClick={() => setShowForm(true)}>Provision First Certificate</Button> : undefined}
          />
        ) : (
          <DataTable
            columns={columns}
            data={certificates}
            onDelete={canManageCertificates ? handleDelete : undefined}
            extraActions={(row: CertificateType) => (
              canManageCertificates && row.status === 'active' ? (
                <button
                  onClick={() => handleRenew(row)}
                  disabled={renewingId === row.id}
                  className="flex items-center gap-2 px-3 py-1.5 text-sm text-[var(--primary-600)] hover:bg-[var(--bg-hover)] rounded-lg transition-colors disabled:opacity-50"
                >
                  <RefreshCw className={`h-4 w-4 ${renewingId === row.id ? 'animate-spin' : ''}`} />
                  {renewingId === row.id ? 'Renewing...' : 'Renew'}
                </button>
              ) : null
            )}
          />
        )}
      </Card>

      <Modal
        open={canManageCertificates && showForm}
        onClose={() => setShowForm(false)}
        title="Provision Certificate"
      >
        <CertificateForm
          onSubmit={handleCreate}
          onCancel={() => setShowForm(false)}
        />
      </Modal>
    </div>
  )
}