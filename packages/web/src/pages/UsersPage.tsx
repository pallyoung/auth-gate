import React from 'react'
import { LockKeyhole, Plus, ShieldCheck, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { UserForm } from '../components/UserForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { routesApi } from '../lib/api/routes'
import { usersApi } from '../lib/api/users'
import { getSessionUser } from '../lib/session-store'
import type { Route, User, UserInput } from '../lib/api/types'

export function UsersPage() {
  const { t } = useTranslation('users')
  const [users, setUsers] = React.useState<User[]>([])
  const [routes, setRoutes] = React.useState<Route[]>([])
  const [loading, setLoading] = React.useState(true)
  const [error, setError] = React.useState('')
  const [showForm, setShowForm] = React.useState(false)
  const [editingUser, setEditingUser] = React.useState<User | null>(null)
  const canManageUsers = getSessionUser()?.permissions?.can_manage_users ?? false

  const fetchData = React.useCallback(async () => {
    try {
      setError('')
      const [usersRes, routesRes] = await Promise.all([usersApi.list(), routesApi.list()])
      setUsers(usersRes)
      setRoutes(routesRes)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  React.useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleCreate = async (data: UserInput) => {
    try {
      await usersApi.create(data)
      setShowForm(false)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleUpdate = async (data: UserInput) => {
    if (!editingUser) return
    try {
      await usersApi.update(editingUser.id, data)
      setShowForm(false)
      setEditingUser(null)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const handleDelete = async (user: User) => {
    if (!confirm(t('page.deleteConfirm', { username: user.username }))) return
    try {
      await usersApi.delete(user.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const memberCount = users.filter((user) => user.role === 'member').length
  const enabledCount = users.filter((user) => user.enabled !== false).length

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
        if (row.role === 'admin' || row.role === 'editor') return t('page.allRoutes')
        if (!value || value.length === 0) return t('page.noAssignedRoutes')
        return t('routeCount', { count: value.length })
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
            <Badge variant="primary">{t('page.badge')}</Badge>
            <span className="text-sm text-[var(--text-muted)]">{t('page.managedUsers', { count: users.length })}</span>
          </>
        }
        action={
          canManageUsers ? (
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

      {error && (
        <Alert variant="error" title={t('page.errorTitle')} className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard
          label={t('page.enabledUsers')}
          value={enabledCount}
          hint={t('page.enabledUsersHint')}
          icon={<ShieldCheck className="h-5 w-5" />}
          tone="primary"
        />
        <MetricCard
          label={t('page.routeMembers')}
          value={memberCount}
          hint={t('page.routeMembersHint')}
          icon={<LockKeyhole className="h-5 w-5" />}
          tone="accent"
        />
        <MetricCard
          label={t('page.operators')}
          value={users.filter((user) => user.role !== 'member').length}
          hint={t('page.operatorsHint')}
          icon={<UserRound className="h-5 w-5" />}
        />
      </div>

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
            title={t('page.emptyTitle')}
            description={t('page.emptyDescription')}
            action={canManageUsers ? <Button onClick={() => setShowForm(true)}>{t('page.createFirst')}</Button> : undefined}
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
