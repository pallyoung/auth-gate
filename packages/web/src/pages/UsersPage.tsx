import React from 'react'
import { LockKeyhole, Plus, ShieldCheck, UserRound } from 'lucide-react'
import { DataTable } from '../components/DataTable'
import { PageHeader } from '../components/PageHeader'
import { UserForm } from '../components/UserForm'
import { Alert, Badge, Button, Card, EmptyState, MetricCard, Modal } from '../components/ui'
import { routesApi } from '../lib/api/routes'
import { usersApi } from '../lib/api/users'
import { getSessionUser } from '../lib/session-store'
import type { Route, User, UserInput } from '../lib/api/types'

export function UsersPage() {
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
    if (!confirm(`Delete user "${user.username}"?`)) return
    try {
      await usersApi.delete(user.id)
      await fetchData()
    } catch (e) {
      setError((e as Error).message)
    }
  }

  const memberCount = users.filter((user) => user.role === 'member').length
  const enabledCount = users.filter((user) => user.enabled !== false).length

  const columns = [
    {
      key: 'username',
      header: 'User',
      render: (value: string, row: User) => (
        <div>
          <div className="font-semibold text-[var(--text-primary)]">{value}</div>
          <div className="mt-1 text-xs text-[var(--text-muted)]">{row.route_ids?.length || 0} assigned routes</div>
        </div>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      className: 'w-32',
      render: (value: string) => <Badge variant="primary" badgeSize="sm">{value}</Badge>,
    },
    {
      key: 'enabled',
      header: 'Status',
      className: 'w-32',
      render: (value: boolean) => <Badge variant={value !== false ? 'success' : 'default'} badgeSize="sm">{value !== false ? 'Enabled' : 'Disabled'}</Badge>,
    },
    {
      key: 'route_ids',
      header: 'Access Scope',
      render: (value: string[], row: User) => {
        if (row.role === 'admin' || row.role === 'editor') return 'All routes'
        if (!value || value.length === 0) return 'No assigned routes'
        return `${value.length} route${value.length > 1 ? 's' : ''}`
      },
    },
  ]

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex items-center gap-3 text-[var(--text-muted)]">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--primary-500)] border-t-transparent" />
          Loading users...
        </div>
      </div>
    )
  }

  return (
    <div className="animate-rise-in">
      <PageHeader
        eyebrow="Identity Directory"
        title="Users"
        description="Manage control-plane operators and route-access members from a shared account directory."
        meta={
          <>
            <Badge variant="primary">Access Model</Badge>
            <span className="text-sm text-[var(--text-muted)]">{users.length} managed users</span>
          </>
        }
        action={
          canManageUsers ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => { setEditingUser(null); setShowForm(true) }}>
              Add User
            </Button>
          ) : null
        }
      />

      {error && (
        <Alert variant="error" title="User operation failed" className="mb-5">
          {error}
        </Alert>
      )}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <MetricCard label="Enabled Users" value={enabledCount} hint="Accounts that can currently authenticate." icon={<ShieldCheck className="h-5 w-5" />} tone="primary" />
        <MetricCard label="Route Members" value={memberCount} hint="Users intended for gateway-managed route access." icon={<LockKeyhole className="h-5 w-5" />} tone="accent" />
        <MetricCard label="Operators" value={users.filter((user) => user.role !== 'member').length} hint="Control-plane capable accounts." icon={<UserRound className="h-5 w-5" />} />
      </div>

      <Card padding="lg" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Directory
          </div>
          <h2 className="mt-2 text-2xl font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
            User access registry
          </h2>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Assign control-plane roles separately from route access permissions so gateway users do not automatically become admins.
          </p>
        </div>

        {users.length === 0 ? (
          <EmptyState
            icon={<UserRound className="h-8 w-8" />}
            title="No users configured"
            description="Create your first user to start assigning control-plane or route access."
            action={canManageUsers ? <Button onClick={() => setShowForm(true)}>Create First User</Button> : undefined}
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
        onClose={() => { setShowForm(false); setEditingUser(null) }}
        title={editingUser ? 'Edit User' : 'Add User'}
      >
        <UserForm
          user={editingUser}
          routes={routes}
          onSubmit={editingUser ? handleUpdate : handleCreate}
          onCancel={() => { setShowForm(false); setEditingUser(null) }}
        />
      </Modal>
    </div>
  )
}
