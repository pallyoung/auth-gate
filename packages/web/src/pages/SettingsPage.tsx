import React from 'react'
import { Database, RefreshCw, Server, Shield } from 'lucide-react'
import { PageHeader } from '../components/PageHeader'
import { Alert, Button, Card, MetricCard } from '../components/ui'
import { configApi } from '../lib/api/config'
import { ApiError } from '../lib/api/client'
import { getSessionUser } from '../lib/session-store'

export function SettingsPage() {
  const [reloading, setReloading] = React.useState(false)
  const [message, setMessage] = React.useState('')
  const [error, setError] = React.useState('')

  const sessionUser = getSessionUser()
  const canReload =
    (sessionUser?.permissions?.can_manage_routes ?? false) ||
    (sessionUser?.permissions?.can_manage_auth ?? false)

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
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="Runtime Operations"
        title="Settings"
        description="Handle runtime reloads, inspect storage assumptions, and review operational safeguards."
      />

      {message && (
        <Alert variant="success" title="Configuration reloaded" className="mb-5">
          {message}
        </Alert>
      )}
      {error && (
        <Alert variant="error" title="Settings operation failed" className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard label="Reload Access" value={canReload ? 'Granted' : 'Restricted'} hint="Derived from current session permissions." icon={<RefreshCw className="h-5 w-5" />} tone="primary" />
        <MetricCard label="Storage" value="SQLite" hint="State persists in the local gateway database." icon={<Database className="h-5 w-5" />} tone="accent" />
        <MetricCard label="Version" value="0.1.0" hint="Current control plane release indicator." icon={<Server className="h-5 w-5" />} />
      </div>

      <div className="grid gap-5 lg:grid-cols-[1.05fr_0.95fr]">
        <Card padding="lg" className="space-y-5">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              Runtime Control
            </div>
            <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
              Configuration reload
            </h2>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              Apply manual changes from the database into the running gateway without restarting the process.
            </p>
          </div>

          <div className="rounded-[24px] border border-[var(--border-default)] bg-[rgba(255,255,255,0.4)] p-5">
            <div className="flex items-start gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-[18px] bg-[var(--bg-soft-primary)] text-[var(--primary-600)]">
                <RefreshCw className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">Push config to runtime</h3>
                <p className="mt-2 text-sm leading-6 text-[var(--text-muted)]">
                  Reload configuration after route or auth changes so the runtime compiler can rebuild its active state.
                </p>
                <Button
                  className="mt-5"
                  loading={reloading}
                  disabled={!canReload}
                  onClick={handleReload}
                  icon={<RefreshCw className="h-4 w-4" />}
                >
                  {reloading ? 'Reloading...' : 'Reload Config'}
                </Button>
              </div>
            </div>
          </div>
        </Card>

        <div className="space-y-5">
          <Card padding="lg" className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--success-light)] text-[var(--success)]">
                <Database className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">Storage</h3>
                <p className="text-sm text-[var(--text-muted)]">Persistent control plane state.</p>
              </div>
            </div>
            <p className="text-sm leading-6 text-[var(--text-muted)]">
              Data is stored in SQLite at <code className="app-code">data/auth-gate.db</code>.
            </p>
          </Card>

          <Card padding="lg" className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--warning-light)] text-[var(--warning)]">
                <Shield className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">Security</h3>
                <p className="text-sm text-[var(--text-muted)]">Deployment checklist.</p>
              </div>
            </div>
            <p className="text-sm leading-6 text-[var(--text-muted)]">
              Configure a strong <code className="app-code">auth.jwt_secret</code> before production deployment.
              Ephemeral JWT secrets should remain development-only.
            </p>
          </Card>

          <Card padding="lg" className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--info-light)] text-[var(--info)]">
                <Server className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">About</h3>
                <p className="text-sm text-[var(--text-muted)]">Runtime and product context.</p>
              </div>
            </div>
            <div className="space-y-1 text-sm text-[var(--text-muted)]">
              <p><strong>Auth Gate</strong> control plane</p>
              <p>Version: 0.1.0</p>
              {sessionUser ? <p>Active role: {sessionUser.role}</p> : null}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
