import React from 'react'
import { Database, RefreshCw, Server, Shield } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { Alert, Button, Card, MetricCard } from '../components/ui'
import { configApi } from '../lib/api/config'
import { ApiError } from '../lib/api/client'
import { getSessionUser } from '../lib/session-store'

type SettingsAlertState = {
  translationKey: string
} | null

export function SettingsPage() {
  const { t } = useTranslation(['settings', 'users'])
  const [reloading, setReloading] = React.useState(false)
  const [message, setMessage] = React.useState<SettingsAlertState>(null)
  const [error, setError] = React.useState<SettingsAlertState>(null)
  const activeReloadRef = React.useRef<symbol | null>(null)

  const sessionUser = getSessionUser()
  const canReload =
    (sessionUser?.permissions?.can_manage_routes ?? false) ||
    (sessionUser?.permissions?.can_manage_auth ?? false)
  const roleLabel = (role: string) => {
    switch (role) {
      case 'member':
        return t('users:roles.member')
      case 'viewer':
        return t('users:roles.viewer')
      case 'editor':
        return t('users:roles.editor')
      case 'admin':
        return t('users:roles.admin')
      default:
        return role
    }
  }

  const getReloadErrorState = (err: unknown): Exclude<SettingsAlertState, null> => {
    if (!(err instanceof ApiError)) {
      return { translationKey: 'page.reloadFallback' }
    }

    switch (err.code) {
      case 'unauthorized':
      case 'invalid_token':
        return { translationKey: 'page.reloadUnauthorized' }
      case 'insufficient_permissions':
        return { translationKey: 'page.reloadForbidden' }
      default:
        return { translationKey: 'page.reloadFallback' }
    }
  }

  const handleReload = async () => {
    if (activeReloadRef.current) {
      return
    }

    const reloadToken = Symbol('settings-reload')
    activeReloadRef.current = reloadToken
    setReloading(true)
    setMessage(null)
    setError(null)

    try {
      await configApi.reload()
      setMessage({ translationKey: 'page.reloadSuccessMessage' })
    } catch (e) {
      setError(getReloadErrorState(e))
    } finally {
      if (activeReloadRef.current === reloadToken) {
        activeReloadRef.current = null
        setReloading(false)
      }
    }
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow={t('page.eyebrow')}
        title={t('page.title')}
        description={t('page.description')}
      />

      {message?.translationKey && (
        <Alert variant="success" title={t('page.successTitle')} className="mb-5">
          {t(message.translationKey as any)}
        </Alert>
      )}
      {error?.translationKey && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {t(error.translationKey as any)}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard
          label={t('page.reloadAccess')}
          value={canReload ? t('page.reloadGranted') : t('page.reloadRestricted')}
          hint={t('page.reloadAccessHint')}
          icon={<RefreshCw className="h-5 w-5" />}
          tone="primary"
        />
        <MetricCard
          label={t('page.storage')}
          value="SQLite"
          hint={t('page.storageHint')}
          icon={<Database className="h-5 w-5" />}
          tone="accent"
        />
        <MetricCard
          label={t('page.version')}
          value="0.1.0"
          hint={t('page.versionHint')}
          icon={<Server className="h-5 w-5" />}
        />
      </div>

      <div className="grid gap-5 lg:grid-cols-[1.05fr_0.95fr]">
        <Card padding="lg" className="space-y-5">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              {t('page.runtimeEyebrow')}
            </div>
            <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
              {t('page.runtimeTitle')}
            </h2>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              {t('page.runtimeDescription')}
            </p>
          </div>

          <div className="rounded-[24px] border border-[var(--border-default)] bg-[rgba(255,255,255,0.4)] p-5">
            <div className="flex items-start gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-[18px] bg-[var(--bg-soft-primary)] text-[var(--primary-600)]">
                <RefreshCw className="h-5 w-5" />
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
                  {t('page.reloadCardTitle')}
                </h3>
                <p className="mt-2 text-sm leading-6 text-[var(--text-muted)]">
                  {t('page.reloadCardDescription')}
                </p>
                <Button
                  className="mt-5"
                  loading={reloading}
                  disabled={!canReload}
                  onClick={handleReload}
                  icon={<RefreshCw className="h-4 w-4" />}
                >
                  {reloading ? t('page.reloadingButton') : t('page.reloadButton')}
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
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">{t('page.storageTitle')}</h3>
                <p className="text-sm text-[var(--text-muted)]">{t('page.storageSubtitle')}</p>
              </div>
            </div>
            <p className="text-sm leading-6 text-[var(--text-muted)]">
              {t('page.storageBody', { path: 'data/auth-gate.db' })}
            </p>
          </Card>

          <Card padding="lg" className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--warning-light)] text-[var(--warning)]">
                <Shield className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">{t('page.securityTitle')}</h3>
                <p className="text-sm text-[var(--text-muted)]">{t('page.securitySubtitle')}</p>
              </div>
            </div>
            <p className="text-sm leading-6 text-[var(--text-muted)]">{t('page.securityBody')}</p>
          </Card>

          <Card padding="lg" className="space-y-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--info-light)] text-[var(--info)]">
                <Server className="h-5 w-5" />
              </div>
              <div>
                <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">{t('page.aboutTitle')}</h3>
                <p className="text-sm text-[var(--text-muted)]">{t('page.aboutSubtitle')}</p>
              </div>
            </div>
            <div className="space-y-1 text-sm text-[var(--text-muted)]">
              <p><strong>{t('page.aboutProduct')}</strong></p>
              <p>{t('page.aboutVersion', { version: '0.1.0' })}</p>
              {sessionUser ? <p>{t('page.aboutRole', { role: roleLabel(sessionUser.role) })}</p> : null}
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
