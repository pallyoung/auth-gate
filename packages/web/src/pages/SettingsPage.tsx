import React from 'react'
import { Button, Card } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { RefreshCw, Database, Shield, Server } from 'lucide-react'
import { ApiError } from '../lib/api/client'
import { configApi } from '../lib/api/config'
import { getSessionUser } from '../lib/session-store'

export function SettingsPage() {
  const [reloading, setReloading] = React.useState(false)
  const [message, setMessage] = React.useState('')
  const [error, setError] = React.useState('')
  const canReload = (getSessionUser()?.permissions?.can_manage_routes ?? false) || (getSessionUser()?.permissions?.can_manage_auth ?? false)

  const handleReload = async () => {
    setReloading(true)
    setMessage('')
    setError('')
    try {
      const result = await configApi.reload()
      setMessage(result.message)
    } catch (e) {
      if (e instanceof ApiError) {
        setError(e.message)
      } else {
        setError('Failed to reload configuration')
      }
    } finally {
      setReloading(false)
    }
  }

  return (
    <div className="animate-fade-in">
      <PageHeader title="Settings" description="Manage your Auth Gate configuration" />

      {message && <div className="mb-4 p-3 rounded-[var(--radius-md)] bg-[var(--success-light)] text-[var(--success)]">{message}</div>}
      {error && <div className="mb-4 p-3 rounded-[var(--radius-md)] bg-[var(--error-light)] text-[var(--error)]">{error}</div>}

      <div className="space-y-6">
        <Card>
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-lg bg-[var(--primary-100)] text-[var(--primary-500)]">
              <RefreshCw className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <h3 className="text-base font-medium text-[var(--text-primary)]">Configuration</h3>
              <p className="text-sm text-[var(--text-muted)] mt-1">
                Reload configuration from database to apply manual changes.
              </p>
              <Button
                className="mt-3"
                loading={reloading}
                disabled={!canReload}
                onClick={handleReload}
              >
                {reloading ? 'Reloading...' : 'Reload Config'}
              </Button>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-lg bg-[var(--success-light)] text-[var(--success)]">
              <Database className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <h3 className="text-base font-medium text-[var(--text-primary)]">Storage</h3>
              <p className="text-sm text-[var(--text-muted)] mt-1">
                Data is stored in SQLite database at <code className="text-xs bg-[var(--neutral-100)] px-1 py-0.5 rounded">data/auth-gate.db</code>
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-lg bg-[var(--warning-light)] text-[var(--warning)]">
              <Shield className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <h3 className="text-base font-medium text-[var(--text-primary)]">Security</h3>
              <p className="text-sm text-[var(--text-muted)] mt-1">
                Configure a strong <code className="text-xs bg-[var(--neutral-100)] px-1 py-0.5 rounded">auth.jwt_secret</code> before production deployment. Ephemeral JWT secrets should remain development-only.
              </p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-lg bg-[var(--info-light)] text-[var(--info)]">
              <Server className="w-5 h-5" />
            </div>
            <div className="flex-1">
              <h3 className="text-base font-medium text-[var(--text-primary)]">About</h3>
              <div className="text-sm text-[var(--text-muted)] mt-1 space-y-1">
                <p><strong>Auth Gate</strong> - API Gateway with Authentication</p>
                <p>Version: 0.1.0</p>
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>
  )
}
