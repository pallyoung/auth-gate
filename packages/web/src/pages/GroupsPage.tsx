import React from 'react'
import { Layers, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { GroupForm } from '../components/GroupForm'
import { PageHeader } from '../components/PageHeader'
import { Alert, Badge, Button, Card, EmptyState, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { permissionGroupsApi } from '../lib/api/groups'
import { routesApi } from '../lib/api/routes'
import { getSessionUser } from '../lib/session-store'
import type { PermissionGroup, PermissionGroupInput, Route } from '../lib/api/types'

export function GroupsPage() {
  const { t } = useTranslation('groups')
  const [groups, setGroups] = React.useState<PermissionGroup[]>([])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [showForm, setShowForm] = React.useState(false)
  const [editingGroup, setEditingGroup] = React.useState<PermissionGroup | null>(null)
  const canManage = getSessionUser()?.permissions?.can_manage_users ?? false
  const errorMessage = resolveLocalizedText(t, error)

  const getErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (!(err instanceof ApiError)) {
      return { translationKey: 'errors.network' }
    }
    switch (err.code) {
      case 'group_not_found':
        return { translationKey: 'errors.groupNotFound' }
      case 'invalid_group_name':
        return { translationKey: 'errors.invalidGroupName' }
      case 'duplicate_group_name':
        return { translationKey: 'errors.duplicateGroupName' }
      case 'group_store_failure':
        return { translationKey: 'errors.groupStoreFailure' }
      default:
        return { message: err.message }
    }
  }, [])

  const fetchData = React.useCallback(async () => {
    try {
      setError(null)
      const [groupsResult, routesResult] = await Promise.allSettled([
        permissionGroupsApi.list(),
        routesApi.list(),
      ])
      if (groupsResult.status === 'fulfilled') {
        setGroups(groupsResult.value)
      } else {
        setGroups([])
        setError(getErrorState(groupsResult.reason))
      }
      if (routesResult.status === 'fulfilled') {
        setRoutes(routesResult.value)
      } else {
        setRoutes([])
      }
    } finally {
      setLoading(false)
    }
  }, [getErrorState])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleCreate = async (data: PermissionGroupInput) => {
    try {
      await permissionGroupsApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleUpdate = async (data: PermissionGroupInput) => {
    if (!editingGroup) return
    try {
      await permissionGroupsApi.update(editingGroup.id, data)
      setShowForm(false)
      setEditingGroup(null)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleDelete = async (group: PermissionGroup) => {
    if (!confirm(t('page.deleteConfirm', { name: group.name }))) return
    try {
      await permissionGroupsApi.delete(group.id)
      await fetchData()
    } catch (err) {
      setError(getErrorState(err))
    }
  }

  const columns = [
    {
      key: 'name',
      header: t('table.name'),
      render: (value: string) => (
        <div className="font-semibold text-[var(--text-primary)]">{value}</div>
      ),
    },
    {
      key: 'route_ids',
      header: t('table.routes'),
      className: 'w-32',
      render: (value: string[]) => {
        const count = value?.length ?? 0
        return count === 0 ? '—' : `${count}`
      },
    },
    {
      key: 'route_paths_detail',
      header: t('table.paths'),
      render: (_value: unknown, row: PermissionGroup) => {
        const routeIDs = row.route_ids || []
        if (routeIDs.length === 0) return '—'
        return (
          <div className="flex flex-wrap gap-1">
            {routeIDs.slice(0, 3).map((routeID) => {
              const route = routes.find((r) => r.id === routeID)
              const label = route?.name || route?.path_prefix || routeID.slice(0, 8)
              const paths = row.route_paths?.[routeID]
              return (
                <Badge key={routeID} variant="default" badgeSize="sm">
                  {label}: {(!paths || paths.length === 0) ? '*' : paths.join(', ')}
                </Badge>
              )
            })}
            {routeIDs.length > 3 && (
              <Badge variant="default" badgeSize="sm">+{routeIDs.length - 3}</Badge>
            )}
          </div>
        )
      },
    },
  ]

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
            <Badge variant="primary">{t('page.groupsCount', { count: groups.length })}</Badge>
          </>
        }
        action={
          canManage ? (
            <Button
              icon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingGroup(null)
                setShowForm(true)
              }}
            >
              {t('page.addGroup')}
            </Button>
          ) : null
        }
      />

      {errorMessage && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {errorMessage}
        </Alert>
      )}

      <Card padding="lg" className="space-y-5">
        {groups.length === 0 ? (
          <EmptyState
            icon={<Layers className="h-8 w-8" />}
            title={t('page.emptyTitle')}
            description={t('page.emptyDescription')}
            action={
              canManage
                ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button>
                : undefined
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={groups}
            onEdit={canManage ? (group) => { setEditingGroup(group); setShowForm(true) } : undefined}
            onDelete={canManage ? handleDelete : undefined}
          />
        )}
      </Card>

      <Modal
        open={canManage && showForm}
        onClose={() => {
          setShowForm(false)
          setEditingGroup(null)
        }}
        title={editingGroup ? t('page.editModalTitle') : t('page.addModalTitle')}
      >
        <GroupForm
          group={editingGroup}
          routes={routes}
          onSubmit={editingGroup ? handleUpdate : handleCreate}
          onCancel={() => {
            setShowForm(false)
            setEditingGroup(null)
          }}
        />
      </Modal>
    </div>
  )
}
