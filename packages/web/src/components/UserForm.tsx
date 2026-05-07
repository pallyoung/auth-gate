import React from 'react'
import type { Route, User, UserInput } from '../lib/api/types'
import { Button, Card, Input, Select, Switch } from './ui'

interface UserFormProps {
  user: User | null
  routes: Route[]
  onSubmit: (data: UserInput) => Promise<void> | void
  onCancel: () => void
}

const roleOptions = [
  { value: 'member', label: 'Member' },
  { value: 'viewer', label: 'Viewer' },
  { value: 'editor', label: 'Editor' },
  { value: 'admin', label: 'Admin' },
]

export function UserForm({ user, routes, onSubmit, onCancel }: UserFormProps) {
  const [form, setForm] = React.useState<UserInput>({
    username: user?.username || '',
    password: '',
    role: user?.role || 'member',
    enabled: user?.enabled ?? true,
    route_ids: user?.route_ids || [],
  })

  const toggleRoute = (routeID: string) => {
    setForm((current) => ({
      ...current,
      route_ids: current.route_ids.includes(routeID)
        ? current.route_ids.filter((id) => id !== routeID)
        : [...current.route_ids, routeID],
    }))
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    await onSubmit(form)
  }

  const needsRouteAssignments = form.role === 'member'

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Identity
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Create a control-plane or route-access account from the shared user directory.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Username"
            value={form.username}
            onChange={(event) => setForm({ ...form, username: event.target.value })}
            placeholder="alice"
            required
          />
          <Select
            label="Role"
            value={form.role}
            onChange={(event) => setForm({ ...form, role: event.target.value })}
            options={roleOptions}
            hint={needsRouteAssignments ? 'Members do not get control-plane access.' : 'Viewer/editor/admin can log into the control plane.'}
            required
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={user ? 'New Password' : 'Password'}
            type="password"
            value={form.password || ''}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
            placeholder={user ? 'Leave blank to keep existing password' : 'Enter password'}
            required={!user}
            hint={user ? 'Only fill this when you want to rotate the password.' : undefined}
          />
          <Switch
            label="Enabled"
            description="Disabled users cannot access the control plane or protected routes."
            checked={form.enabled}
            onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
          />
        </div>
      </Card>

      <Card tone="soft" className="space-y-4">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Route Access
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Assign which routes this account may access when a route uses gateway-managed login.
          </p>
        </div>

        {routes.length === 0 ? (
          <div className="rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-5 text-sm text-[var(--text-muted)]">
            No routes available yet. Create routes before assigning access.
          </div>
        ) : (
          <div className="space-y-3">
            {routes.map((route) => {
              const checked = form.route_ids.includes(route.id)
              return (
                <label
                  key={route.id}
                  className="flex items-center justify-between gap-4 rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-3 shadow-[var(--shadow-sm)]"
                >
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-[var(--text-primary)]">{route.name || route.path_prefix}</div>
                    <div className="mt-1 text-xs text-[var(--text-muted)]">
                      {route.host || 'all hosts'} · {route.path_prefix}
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-[var(--border-default)] text-[var(--primary-600)]"
                    checked={checked}
                    onChange={() => toggleRoute(route.id)}
                  />
                </label>
              )
            })}
          </div>
        )}
      </Card>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {user ? 'Update User' : 'Create User'}
        </Button>
      </div>
    </form>
  )
}
