import React from 'react'
import { Button, Card } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { RefreshCw, Database, Shield, Server } from 'lucide-react'

export function SettingsPage() {
  const [reloading, setReloading] = React.useState(false)

  const handleReload = async () => {
    setReloading(true)
    try {
      await fetch('/api/config/reload')
      alert('Configuration reloaded')
    } catch (e) {
      alert('Failed to reload: ' + e)
    }
    setReloading(false)
  }

  return (
    <div className="animate-fade-in">
      <PageHeader title="Settings" description="Manage your Auth Gate configuration" />

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
                Remember to change the admin token in production. Configure it in <code className="text-xs bg-[var(--neutral-100)] px-1 py-0.5 rounded">configs/config.yaml</code>
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
