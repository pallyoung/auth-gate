import React from 'react'
import { Network, Plus, Pencil, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { PageHeader } from '../components/PageHeader'
import { HostsTable } from '../components/Hosts/HostsTable'
import { ProfileSwitcher } from '../components/Hosts/ProfileSwitcher'
import { HostEntryForm } from '../components/Hosts/HostEntryForm'
import { HostProfileForm } from '../components/Hosts/HostProfileForm'
import { Alert, Button, Card, EmptyState, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { hostsApi } from '../lib/api/hosts'
import { getSessionUser } from '../lib/session-store'
import type { HostEntry, HostEntryInput, HostProfile, HostProfileInput } from '../lib/api/types'

export function HostsPage() {
  const { t } = useTranslation('hosts')
  const [profiles, setProfiles] = React.useState<HostProfile[]>([])
  const [selectedProfileId, setSelectedProfileId] = React.useState<string | null>(null)
  const [entries, setEntries] = React.useState<HostEntry[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [success, setSuccess] = React.useState<string | null>(null)
  const [pendingAction, setPendingAction] = React.useState<'activate' | 'deactivate' | null>(null)
  const actionInFlight = React.useRef(false)
  const [markerBanner, setMarkerBanner] = React.useState(false)
  const [showProfileForm, setShowProfileForm] = React.useState(false)
  const [editingProfile, setEditingProfile] = React.useState<HostProfile | null>(null)
  const [showEntryForm, setShowEntryForm] = React.useState(false)
  const [editingEntry, setEditingEntry] = React.useState<HostEntry | null>(null)
  const canManage = getSessionUser()?.permissions?.can_manage_hosts ?? false
  const selectedProfileIdRef = React.useRef(selectedProfileId)
  selectedProfileIdRef.current = selectedProfileId

  const fetchEntries = React.useCallback(async (profileId: string) => {
    try {
      const data = await hostsApi.listEntries(profileId)
      setEntries(data)
    } catch (err) {
      setError((err as Error).message)
    }
  }, [])

  React.useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        setError(null)
        const data = await hostsApi.list()
        if (cancelled) return
        setProfiles(data.profiles)
        if (data.profiles.length > 0) {
          const activeId = data.profiles.find((p) => p.is_active)?.id || data.profiles[0].id
          setSelectedProfileId(activeId)
          await fetchEntries(activeId)
        }
      } catch (err) {
        if (!cancelled) setError((err as Error).message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => { cancelled = true }
  }, [fetchEntries])

  React.useEffect(() => {
    if (selectedProfileId && !loading) {
      fetchEntries(selectedProfileId)
    }
  }, [selectedProfileId, loading, fetchEntries])

  const handleSelectProfile = (id: string) => {
    setSelectedProfileId(id)
  }

  const handleApply = async (action: 'activate' | 'deactivate') => {
    const profile = profiles.find((p) => p.id === selectedProfileId)
    if (!profile || actionInFlight.current) return
    actionInFlight.current = true
    setPendingAction(action)
    setMarkerBanner(false)
    setError(null)
    setSuccess(null)
    try {
      const updated = await hostsApi[action](profile.id)
      setProfiles((current) => current.map((p) => {
        if (p.id === updated.id) return updated
        return action === 'activate' ? { ...p, is_active: false } : p
      }))
      setSuccess(t(action === 'activate' ? 'activated' : 'deactivated', { name: profile.name }))
    } catch (err) {
      if (err instanceof ApiError && err.code === 'host_marker_missing') {
        setMarkerBanner(true)
      } else if (err instanceof ApiError && err.code && ['host_permission_denied', 'host_render_failure'].includes(err.code)) {
        setError(t(`errors.${err.code}`))
      } else {
        setError((err as Error).message)
      }
    } finally {
      actionInFlight.current = false
      setPendingAction(null)
    }
  }

  const handleCreateProfile = async (data: HostProfileInput) => {
    await hostsApi.create(data)
    setShowProfileForm(false)
    const result = await hostsApi.list()
    setProfiles(result.profiles)
    if (!selectedProfileIdRef.current && result.profiles.length > 0) {
      setSelectedProfileId(result.profiles[0].id)
    }
  }

  const handleUpdateProfile = async (data: HostProfileInput) => {
    if (!editingProfile) return
    await hostsApi.update(editingProfile.id, data)
    setEditingProfile(null)
    const result = await hostsApi.list()
    setProfiles(result.profiles)
  }

  const handleDeleteProfile = async (profile: HostProfile) => {
    if (!window.confirm(t('deleteProfileConfirm', { name: profile.name }))) return
    await hostsApi.delete(profile.id)
    const result = await hostsApi.list()
    setProfiles(result.profiles)
    if (selectedProfileId === profile.id) {
      setSelectedProfileId(result.profiles.length > 0 ? result.profiles[0].id : null)
    }
  }

  const handleCreateEntry = async (data: HostEntryInput) => {
    if (!selectedProfileId) return
    await hostsApi.createEntry(selectedProfileId, data)
    setShowEntryForm(false)
    await fetchEntries(selectedProfileId)
  }

  const handleUpdateEntry = async (data: HostEntryInput) => {
    if (!selectedProfileId || !editingEntry) return
    await hostsApi.updateEntry(selectedProfileId, editingEntry.id, data)
    setEditingEntry(null)
    await fetchEntries(selectedProfileId)
  }

  const handleDeleteEntry = async (entry: HostEntry) => {
    if (!selectedProfileId) return
    if (!confirm(t('deleteEntryConfirm', { hostnames: entry.hostnames }))) return
    await hostsApi.deleteEntry(selectedProfileId, entry.id)
    await fetchEntries(selectedProfileId)
  }

  const handleToggleEntry = async (entry: HostEntry, enabled: boolean) => {
    if (!selectedProfileId) return
    await hostsApi.updateEntry(selectedProfileId, entry.id, {
      ip: entry.ip,
      hostnames: entry.hostnames.split(/\s+/).filter(Boolean),
      comment: entry.comment,
      enabled,
    })
    await fetchEntries(selectedProfileId)
  }

  const selectedProfile = profiles.find((p) => p.id === selectedProfileId)
  const activatedProfile = profiles.find((p) => p.is_active)

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--border-default)] border-t-[var(--color-primary-500)]" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        eyebrow={t('eyebrow')}
        description={t('description')}
        action={
          canManage ? (
            <Button onClick={() => setShowProfileForm(true)}>
              <Plus className="mr-2 h-4 w-4" />
              {t('addProfile')}
            </Button>
          ) : undefined
        }
      />

      {error && <Alert variant="error" role="alert">{error}</Alert>}
      {success && <Alert variant="success" role="status">{success}</Alert>}
      <Alert variant={activatedProfile ? 'success' : 'info'}>
        <span>{activatedProfile ? t('activeProfile', { name: activatedProfile.name }) : t('noActiveProfile')}</span>
      </Alert>
      {markerBanner && <Alert variant="warning" role="alert">{t('markerBanner')}</Alert>}

      {profiles.length === 0 ? (
        <Card>
          <EmptyState icon={<Network />} title={t('empty.title')} description={t('empty.description')} />
        </Card>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3">
            <ProfileSwitcher
              profiles={profiles}
              selectedId={selectedProfileId ?? ''}
              disabled={!!pendingAction}
              onChange={handleSelectProfile}
            />
            {canManage && selectedProfileId && (
              <>
                <Button variant="secondary" disabled={!!pendingAction} loading={pendingAction === 'activate'} onClick={() => handleApply('activate')}>
                  {t(pendingAction === 'activate' ? 'activating' : selectedProfile?.is_active ? 'reapply' : 'activate')}
                </Button>
                {selectedProfile?.is_active && (
                  <Button variant="secondary" disabled={!!pendingAction} loading={pendingAction === 'deactivate'} onClick={() => handleApply('deactivate')}>
                    {t(pendingAction === 'deactivate' ? 'deactivating' : 'deactivate')}
                  </Button>
                )}
              </>
            )}
          </div>

          <p className="text-sm text-[var(--text-muted)]">{t('applyHint')}</p>
          <Card>
            <div className="mb-4 flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-lg font-semibold text-[var(--text-primary)]">
                    {selectedProfile?.name ?? t('title')}
                  </h3>
                  {canManage && selectedProfile && (
                    <>
                      <button
                        type="button"
                        disabled={!!pendingAction}
                        onClick={() => setEditingProfile(selectedProfile)}
                        className="rounded p-1 text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                        title={t('editProfile')}
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                      <button
                        type="button"
                        disabled={!!pendingAction}
                        onClick={() => handleDeleteProfile(selectedProfile)}
                        className="rounded p-1 text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--color-danger-500)]"
                        title={t('deleteProfile')}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </>
                  )}
                </div>
                {selectedProfile?.description && (
                  <p className="mt-1 text-sm text-[var(--text-muted)]">{selectedProfile.description}</p>
                )}
              </div>
              {canManage && (
                <Button onClick={() => setShowEntryForm(true)}>
                  <Plus className="mr-2 h-4 w-4" />
                  {t('addEntry')}
                </Button>
              )}
            </div>

            <HostsTable
              entries={entries}
              canManage={canManage}
              onEdit={(entry) => setEditingEntry(entry)}
              onDelete={handleDeleteEntry}
              onToggleEnabled={handleToggleEntry}
            />
          </Card>
        </>
      )}

      <Modal open={showProfileForm} onClose={() => setShowProfileForm(false)} title={t('addProfile')}>
        {showProfileForm && (
          <HostProfileForm
            onSubmit={handleCreateProfile}
            onCancel={() => setShowProfileForm(false)}
          />
        )}
      </Modal>

      <Modal open={!!editingProfile} onClose={() => setEditingProfile(null)} title={t('editProfile')}>
        {editingProfile && (
          <HostProfileForm
            initial={{ name: editingProfile.name, description: editingProfile.description }}
            onSubmit={handleUpdateProfile}
            onCancel={() => setEditingProfile(null)}
          />
        )}
      </Modal>

      <Modal open={showEntryForm} onClose={() => setShowEntryForm(false)} title={t('addEntry')}>
        {showEntryForm && (
          <HostEntryForm
            onSubmit={handleCreateEntry}
            onCancel={() => setShowEntryForm(false)}
          />
        )}
      </Modal>

      <Modal open={!!editingEntry} onClose={() => setEditingEntry(null)} title={t('form.save')}>
        {editingEntry && (
          <HostEntryForm
            initial={{ ip: editingEntry.ip, hostnames: editingEntry.hostnames.split(/\s+/).filter(Boolean), comment: editingEntry.comment, enabled: editingEntry.enabled }}
            onSubmit={handleUpdateEntry}
            onCancel={() => setEditingEntry(null)}
          />
        )}
      </Modal>
    </div>
  )
}
