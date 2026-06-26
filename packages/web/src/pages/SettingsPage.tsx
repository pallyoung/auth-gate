import React from 'react'
import { Database, Globe, Plus, RefreshCw, Server, Shield, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { Alert, Button, Card, Input, MetricCard, Switch } from '../components/ui'
import { configApi } from '../lib/api/config'
import { ApiError } from '../lib/api/client'
import { getSessionUser } from '../lib/session-store'
import type { ListenEntry } from '../lib/api/types'

type SettingsAlertState = {
  translationKey: string
} | null

export function SettingsPage() {
  const { t } = useTranslation(['settings', 'users'])
  const [reloading, setReloading] = React.useState(false)
  const [message, setMessage] = React.useState<SettingsAlertState>(null)
  const [error, setError] = React.useState<SettingsAlertState>(null)
  const activeReloadRef = React.useRef<symbol | null>(null)

  // Listen ports state
  const [listenEntries, setListenEntries] = React.useState<ListenEntry[]>([
    { addr: ':80', tls: false },
    { addr: ':443', tls: true },
  ])
  const [configLoading, setConfigLoading] = React.useState(true)
  const [configSaving, setConfigSaving] = React.useState(false)
  const [configMessage, setConfigMessage] = React.useState<SettingsAlertState>(null)
  const [configError, setConfigError] = React.useState<SettingsAlertState>(null)

  // Log retention state
  const [retentionDays, setRetentionDays] = React.useState(0)
  const [retentionLoading, setRetentionLoading] = React.useState(true)
  const [retentionSaving, setRetentionSaving] = React.useState(false)
  const [retentionMessage, setRetentionMessage] = React.useState<SettingsAlertState>(null)
  const [retentionError, setRetentionError] = React.useState<SettingsAlertState>(null)

  const sessionUser = getSessionUser()
  const isAdmin = sessionUser?.role === 'admin'

  React.useEffect(() => {
    configApi.get().then((cfg) => {
      setListenEntries(cfg.listen.length > 0 ? cfg.listen : [{ addr: ':80', tls: false }])
    }).catch(() => {
      // Ignore - use defaults
    }).finally(() => setConfigLoading(false))
  }, [])

  React.useEffect(() => {
    if (!isAdmin) { setRetentionLoading(false); return }
    configApi.getLogRetention().then((res) => {
      setRetentionDays(res.days)
    }).catch(() => {
      // Ignore - use defaults
    }).finally(() => setRetentionLoading(false))
  }, [isAdmin])

  const canReload =
    (sessionUser?.permissions?.can_manage_routes ?? false) ||
    (sessionUser?.permissions?.can_manage_auth ?? false)
  const roleLabel = (role: string) => {
    switch (role) {
      case 'member': return t('users:roles.member')
      case 'viewer': return t('users:roles.viewer')
      case 'editor': return t('users:roles.editor')
      case 'admin': return t('users:roles.admin')
      default: return role
    }
  }

  const getReloadErrorState = (err: unknown): Exclude<SettingsAlertState, null> => {
    if (!(err instanceof ApiError)) return { translationKey: 'page.reloadFallback' }
    switch (err.code) {
      case 'unauthorized':
      case 'invalid_token': return { translationKey: 'page.reloadUnauthorized' }
      case 'insufficient_permissions': return { translationKey: 'page.reloadForbidden' }
      default: return { translationKey: 'page.reloadFallback' }
    }
  }

  const handleReload = async () => {
    if (activeReloadRef.current) return
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

  const addListenEntry = () => {
    setListenEntries([...listenEntries, { addr: '', tls: false }])
  }

  const removeListenEntry = (index: number) => {
    setListenEntries(listenEntries.filter((_, i) => i !== index))
  }

  const updateListenEntry = (index: number, field: keyof ListenEntry, value: string | boolean) => {
    const updated = [...listenEntries]
    updated[index] = { ...updated[index], [field]: value }
    setListenEntries(updated)
  }

  const handleSaveConfig = async () => {
    setConfigSaving(true)
    setConfigMessage(null)
    setConfigError(null)

    const valid = listenEntries.filter((e) => e.addr.trim() !== '')
    if (valid.length === 0) {
      setConfigError({ translationKey: 'config.atLeastOneAddr' })
      setConfigSaving(false)
      return
    }

    try {
      await configApi.update(valid)
      setConfigMessage({ translationKey: 'config.saved' })
    } catch {
      setConfigError({ translationKey: 'config.saveFailed' })
    } finally {
      setConfigSaving(false)
    }
  }

  const handleSaveRetention = async () => {
    setRetentionSaving(true)
    setRetentionMessage(null)
    setRetentionError(null)
    try {
      await configApi.updateLogRetention(retentionDays)
      setRetentionMessage({ translationKey: 'logRetention.saved' })
    } catch {
      setRetentionError({ translationKey: 'logRetention.saveFailed' })
    } finally {
      setRetentionSaving(false)
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
        <div className="space-y-5">
          {/* Listen Ports Card */}
          {isAdmin && (
            <Card padding="lg" className="space-y-5">
              <div>
                <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                  {t('config.eyebrow')}
                </div>
                <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
                  {t('config.title')}
                </h2>
                <p className="mt-2 text-sm text-[var(--text-muted)]">
                  {t('config.description')}
                </p>
              </div>

              {configMessage?.translationKey && (
                <Alert variant="success">{t(configMessage.translationKey as any)}</Alert>
              )}
              {configError?.translationKey && (
                <Alert variant="error">{t(configError.translationKey as any)}</Alert>
              )}

              {!configLoading && (
                <>
                  <div className="space-y-3">
                    {listenEntries.map((entry, index) => (
                      <div key={index} className="flex items-center gap-3">
                        <Globe className="h-4 w-4 shrink-0 text-[var(--text-muted)]" />
                        <Input
                          value={entry.addr}
                          onChange={(e) => updateListenEntry(index, 'addr', e.target.value)}
                          placeholder=":80"
                          className="flex-1"
                        />
                        <div className="flex items-center gap-1.5">
                          <Switch
                            checked={entry.tls}
                            onChange={(e) => updateListenEntry(index, 'tls', e.target.checked)}
                          />
                          <span className="whitespace-nowrap text-xs text-[var(--text-muted)]">HTTPS</span>
                        </div>
                        {listenEntries.length > 1 && (
                          <button
                            type="button"
                            onClick={() => removeListenEntry(index)}
                            className="flex h-10 w-10 items-center justify-center rounded-full text-[var(--error)] hover:bg-[var(--error-light)]"
                          >
                            <X className="h-4 w-4" />
                          </button>
                        )}
                      </div>
                    ))}
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={addListenEntry}
                      icon={<Plus className="h-3.5 w-3.5" />}
                    >
                      {t('config.addPort')}
                    </Button>
                  </div>

                  <div className="rounded-[18px] border border-[var(--border-default)] bg-[rgba(255,255,255,0.4)] px-4 py-3 text-xs text-[var(--text-muted)]">
                    {t('config.restartHint')}
                  </div>

                  <Button loading={configSaving} onClick={handleSaveConfig}>
                    {configSaving ? t('config.saving') : t('config.save')}
                  </Button>
                </>
              )}
            </Card>
          )}

          {/* Runtime Control Card */}
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

          {/* Log Retention Card */}
          {isAdmin && (
            <Card padding="lg" className="space-y-5">
              <div>
                <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                  {t('logRetention.eyebrow')}
                </div>
                <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
                  {t('logRetention.title')}
                </h2>
                <p className="mt-2 text-sm text-[var(--text-muted)]">
                  {t('logRetention.description')}
                </p>
              </div>

              {retentionMessage?.translationKey && (
                <Alert variant="success">{t(retentionMessage.translationKey as any)}</Alert>
              )}
              {retentionError?.translationKey && (
                <Alert variant="error">{t(retentionError.translationKey as any)}</Alert>
              )}

              {!retentionLoading && (
                <div className="rounded-[24px] border border-[var(--border-default)] bg-[rgba(255,255,255,0.4)] p-5">
                  <div className="flex items-start gap-4">
                    <div className="flex h-12 w-12 items-center justify-center rounded-[18px] bg-[var(--error-light)] text-[var(--error)]">
                      <Trash2 className="h-5 w-5" />
                    </div>
                    <div className="flex-1 space-y-4">
                      <div>
                        <h3 className="text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
                          {t('logRetention.label')}
                        </h3>
                        <p className="mt-1 text-sm text-[var(--text-muted)]">
                          {retentionDays > 0
                            ? t('logRetention.description')
                            : t('logRetention.disabled')}
                        </p>
                      </div>
                      <div className="flex items-center gap-3">
                        <Input
                          type="number"
                          min={0}
                          value={String(retentionDays)}
                          onChange={(e) => setRetentionDays(Math.max(0, parseInt(e.target.value) || 0))}
                          className="w-24"
                        />
                        <span className="text-sm text-[var(--text-muted)]">{t('logRetention.unit')}</span>
                      </div>
                      <p className="text-xs text-[var(--text-muted)]">
                        {t('logRetention.hint')}
                      </p>
                      <Button
                        loading={retentionSaving}
                        onClick={handleSaveRetention}
                        size="sm"
                      >
                        {retentionSaving ? t('logRetention.saving') : t('logRetention.save')}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </Card>
          )}
        </div>

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
