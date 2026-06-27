import React from 'react'
import { LockKeyhole, Plus, ShieldCheck, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { UserForm } from '../components/UserForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { ApiError } from '../lib/api/client'
import { LocalizedError, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { routesApi } from '../lib/api/routes'
import { usersApi } from '../lib/api/users'
import { permissionGroupsApi } from '../lib/api/groups'
import { getSessionUser } from '../lib/session-store'
import type { PermissionGroup, Route, User, UserInput } from '../lib/api/types'

export function UsersPage() {
  const { t } = useTranslation('users')
  const [users, setUsers] = React.useState<User[]>([])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [groups, setGroups] = React.useState<PermissionGroup[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [userListUnavailable, setUserListUnavailable] = React.useState(false)
  const [userDirectoryUnavailable, setUserDirectoryUnavailable] = React.useState(false)
  const [routeListUnavailable, setRouteListUnavailable] = React.useState(false)
  const [showForm, setShowForm] = React.useState(false)
  const [editingUser, setEditingUser] = React.useState<User | null>(null)
  const requestGenerationRef = React.useRef(0)
  const canManageUsers = getSessionUser()?.permissions?.can_manage_users ?? false
  const showDirectoryMetrics = !userListUnavailable
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
      case 'user_not_found':
        return { translationKey: 'errors.userNotFound' }
      case 'invalid_username':
        return { translationKey: 'errors.invalidUsername' }
      case 'invalid_role':
        return { translationKey: 'errors.invalidRole' }
      case 'duplicate_user':
        return { translationKey: 'errors.duplicateUser' }
      case 'duplicate_route_access':
        return { translationKey: 'errors.duplicateRouteAccess' }
      case 'route_not_found':
        return { translationKey: 'errors.routeNotFound' }
      case 'route_store_failure':
        return { translationKey: 'errors.routeStoreFailure' }
      case 'missing_password':
        return { translationKey: 'errors.missingPassword' }
      case 'password_hash_failed':
      case 'user_store_failure':
        return { translationKey: 'errors.userStoreFailure' }
      default:
        return { message: err.message }
    }
  }, [])

  const getListErrorState = React.useCallback((err: unknown): Exclude<LocalizedTextState, null> => {
    if (err instanceof ApiError && err.code === 'user_store_failure') {
      return { translationKey: 'errors.userDirectoryUnavailable' }
    }

    return getErrorState(err)
  }, [getErrorState])

  const fetchData = React.useCallback(async () => {
    const requestGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = requestGeneration

    try {
      setError(null)
      setUserListUnavailable(false)
      setUserDirectoryUnavailable(false)
      setRouteListUnavailable(false)
      const [usersResult, routesResult, groupsResult] = await Promise.allSettled([
        usersApi.list(),
        routesApi.list(),
        permissionGroupsApi.list(),
      ])

      if (requestGenerationRef.current !== requestGeneration) {
        return
      }

      if (usersResult.status === 'fulfilled') {
        setUsers(usersResult.value)
      } else {
        setUsers([])
        setUserListUnavailable(true)
        setUserDirectoryUnavailable(
          usersResult.reason instanceof ApiError && usersResult.reason.code === 'user_store_failure'
        )
      }

      if (routesResult.status === 'fulfilled') {
        setRoutes(routesResult.value)
      } else {
        setRoutes([])
        setRouteListUnavailable(true)
      }

      if (groupsResult.status === 'fulfilled') {
        setGroups(groupsResult.value)
      } else {
        setGroups([])
      }

      if (usersResult.status === 'rejected') {
        setError(getListErrorState(usersResult.reason))
        return
      }

      if (routesResult.status === 'rejected') {
        setError(getErrorState(routesResult.reason))
      }
    } finally {
      if (requestGenerationRef.current === requestGeneration) {
        setLoading(false)
      }
    }
  }, [getErrorState, getListErrorState])

  React.useEffect(() => {
    if (!canManageUsers) {
      setLoading(false)
      return
    }

    fetchData()
  }, [canManageUsers, fetchData])

  const handleCreate = async (data: UserInput) => {
    try {
      await usersApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleUpdate = async (data: UserInput) => {
    if (!editingUser) return
    try {
      await usersApi.update(editingUser.id, data)
      setShowForm(false)
      setEditingUser(null)
      await fetchData()
    } catch (err) {
      throw new LocalizedError(getErrorState(err))
    }
  }

  const handleDelete = async (user: User) => {
    if (!confirm(t('page.deleteConfirm', { username: user.username }))) return
    try {
      await usersApi.delete(user.id)
      await fetchData()
    } catch (err) {
      setError(getErrorState(err))
    }
  }

  const routeAccessUserCount = users.filter((user) => (user.route_ids?.length ?? 0) > 0).length
  const enabledCount = users.filter((user) => user.enabled !== false).length
  const operatorCount = users.filter((user) => user.role !== 'member').length

  const roleLabel = (value: string) => {
    switch (value) {
      case 'member':
        return t('roles.member')
      case 'viewer':
        return t('roles.viewer')
      case 'editor':
        return t('roles.editor')
      case 'admin':
        return t('roles.admin')
      default:
        return value
    }
  }

  const columns = [
    {
      key: 'username',
      header: t('table.user'),
      render: (value: string, row: User) => (
        <div>
          <div className="font-semibold text-[var(--text-primary)]">{value}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">
            {t('page.assignedRoutes', { count: row.route_ids?.length || 0 })}
          </div>
        </div>
      ),
    },
    {
      key: 'role',
      header: t('table.role'),
      className: 'w-32',
      render: (value: string) => <Badge variant="primary" badgeSize="sm">{roleLabel(value)}</Badge>,
    },
    {
      key: 'enabled',
      header: t('table.status'),
      className: 'w-32',
      render: (value: boolean) => (
        <Badge variant={value !== false ? 'success' : 'default'} badgeSize="sm">
          {value !== false ? t('page.enabled') : t('page.disabled')}
        </Badge>
      ),
    },
    {
      key: 'route_ids',
      header: t('table.accessScope'),
      render: (value: string[], row: User) => {
        if (!value || value.length === 0) return t('page.noAssignedRoutes')
        return t('routeCount', { count: value.length })
      },
    },
  ]

  if (!canManageUsers) {
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
            icon={<UserRound className="h-8 w-8" />}
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
              <span className="text-sm text-[var(--text-muted)]">{t('page.managedUsers', { count: users.length })}</span>
            ) : null}
          </>
        }
        action={
          canManageUsers && !userListUnavailable ? (
            <Button
              icon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingUser(null)
                setShowForm(true)
              }}
            >
              {t('page.addUser')}
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
            label={t('page.enabledUsers')}
            value={enabledCount}
            hint={t('page.enabledUsersHint')}
            icon={<ShieldCheck className="h-5 w-5" />}
            tone="primary"
          />
          <MetricCard
            label={t('page.routeAccessUsers')}
            value={routeAccessUserCount}
            hint={t('page.routeAccessUsersHint')}
            icon={<LockKeyhole className="h-5 w-5" />}
            tone="accent"
          />
          <MetricCard
            label={t('page.operators')}
            value={operatorCount}
            hint={t('page.operatorsHint')}
            icon={<UserRound className="h-5 w-5" />}
          />
        </div>
      ) : null}

      <Card padding="lg" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('page.directoryEyebrow')}
          </div>
          <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
            {t('page.directoryTitle')}
          </h2>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('page.directoryDescription')}
          </p>
        </div>

        {users.length === 0 ? (
          <EmptyState
            icon={<UserRound className="h-8 w-8" />}
            title={
              userDirectoryUnavailable
                ? t('page.directoryUnavailableTitle')
                : userListUnavailable
                ? t('page.listUnavailableTitle')
                : t('page.emptyTitle')
            }
            description={
              userDirectoryUnavailable
                ? t('page.directoryUnavailableDescription')
                : userListUnavailable
                ? t('page.listUnavailableDescription')
                : t('page.emptyDescription')
            }
            action={
              canManageUsers && !userListUnavailable
                ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button>
                : undefined
            }
          />
        ) : (
          <DataTable
            columns={columns}
            data={users}
            onEdit={canManageUsers ? (user) => { setEditingUser(user); setShowForm(true) } : undefined}
            onDelete={canManageUsers ? handleDelete : undefined}
          />
        )}
      </Card>

      <Modal
        open={canManageUsers && showForm}
        onClose={() => {
          setShowForm(false)
          setEditingUser(null)
        }}
        title={editingUser ? t('page.editModalTitle') : t('page.addModalTitle')}
      >
        <UserForm
          user={editingUser}
          routes={routes}
          groups={groups}
          routeListUnavailable={routeListUnavailable}
          onSubmit={editingUser ? handleUpdate : handleCreate}
          onCancel={() => {
            setShowForm(false)
            setEditingUser(null)
          }}
        />
      </Modal>
    </div>
  )
}
