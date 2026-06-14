import React from 'react'
import { Network, Plus } from 'lucide-react'
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
  const [activeProfileId, setActiveProfileId] = React.useState<string | null>(null)
  const [entries, setEntries] = React.useState<HostEntry[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [markerBanner, setMarkerBanner] = React.useState(false)
  const [showProfileForm, setShowProfileForm] = React.useState(false)
  const [showEntryForm, setShowEntryForm] = React.useState(false)
  const [editingEntry, setEditingEntry] = React.useState<HostEntry | null>(null)
  const canManage = getSessionUser()?.permissions?.can_manage_hosts ?? false
  const activeProfileIdRef = React.useRef(activeProfileId)
  activeProfileIdRef.current = activeProfileId

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
          const activeId = data.active_id || data.profiles[0].id
          setActiveProfileId(activeId)
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
    if (activeProfileId && !loading) {
      fetchEntries(activeProfileId)
    }
  }, [activeProfileId, loading, fetchEntries])

  const handleSelectProfile = (id: string) => {
    setActiveProfileId(id)
  }

  const handleActivate = async () => {
    if (!activeProfileId) return
    setMarkerBanner(false)
    try {
      await hostsApi.activate(activeProfileId)
      const data = await hostsApi.list()
      setProfiles(data.profiles)
    } catch (err) {
      if (err instanceof ApiError && err.code === 'host_marker_missing') {
        setMarkerBanner(true)
      } else {
        setError((err as Error).message)
      }
    }
  }

  const handleCreateProfile = async (data: HostProfileInput) => {
    await hostsApi.create(data)
    setShowProfileForm(false)
    const result = await hostsApi.list()
    setProfiles(result.profiles)
  }

  const handleCreateEntry = async (data: HostEntryInput) => {
    if (!activeProfileId) return
    await hostsApi.createEntry(activeProfileId, data)
    setShowEntryForm(false)
    await fetchEntries(activeProfileId)
  }

  const handleUpdateEntry = async (data: HostEntryInput) => {
    if (!activeProfileId || !editingEntry) return
    await hostsApi.updateEntry(activeProfileId, editingEntry.id, data)
    setEditingEntry(null)
    await fetchEntries(activeProfileId)
  }

  const handleDeleteEntry = async (entry: HostEntry) => {
    if (!activeProfileId) return
    if (!confirm(t('errors.host_entry_not_found'))) return
    await hostsApi.deleteEntry(activeProfileId, entry.id)
    await fetchEntries(activeProfileId)
  }

  const handleToggleEntry = async (entry: HostEntry, enabled: boolean) => {
    if (!activeProfileId) return
    await hostsApi.updateEntry(activeProfileId, entry.id, {
      ip: entry.ip,
      hostnames: entry.hostnames.split(/\s+/).filter(Boolean),
      comment: entry.comment,
      enabled,
    })
    await fetchEntries(activeProfileId)
  }

  const activeProfile = profiles.find((p) => p.id === activeProfileId)

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

      {error && <Alert variant="error">{error}</Alert>}
      {markerBanner && <Alert variant="warning">{t('markerBanner')}</Alert>}

      {profiles.length === 0 ? (
        <Card>
          <EmptyState icon={<Network />} title={t('empty.title')} description={t('empty.description')} />
        </Card>
      ) : (
        <>
          <div className="flex items-center gap-3">
            <ProfileSwitcher
              profiles={profiles}
              activeId={activeProfileId ?? ''}
              canManage={canManage}
              onChange={handleSelectProfile}
            />
            {canManage && activeProfileId && (
              <Button variant="secondary" onClick={handleActivate}>
                {t('activate')}
              </Button>
            )}
          </div>

          <Card>
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h3 className="text-lg font-semibold text-[var(--text-primary)]">
                  {activeProfile?.name ?? t('title')}
                </h3>
                {activeProfile?.description && (
                  <p className="mt-1 text-sm text-[var(--text-muted)]">{activeProfile.description}</p>
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
