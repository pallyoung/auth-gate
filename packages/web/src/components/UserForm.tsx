import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import type { PermissionGroup, Route, User, UserInput } from '../lib/api/types'
import { Alert, Button, Card, Input, Select, Switch } from './ui'

interface UserFormProps {
  user: User | null
  routes: Route[]
  groups?: PermissionGroup[]
  routeListUnavailable?: boolean
  onSubmit: (data: UserInput) => Promise<void> | void
  onCancel: () => void
}

function getInitialUserForm(user: User | null): UserInput {
  return {
    username: user?.username || '',
    password: '',
    role: user?.role || 'member',
    enabled: user?.enabled ?? true,
    route_ids: user?.route_ids || [],
    group_ids: user?.group_ids || [],
    route_paths: user?.route_paths || {},
  }
}

export function UserForm({ user, routes, groups = [], routeListUnavailable = false, onSubmit, onCancel }: UserFormProps) {
  const { t } = useTranslation('users')
  const [form, setForm] = React.useState<UserInput>(() => getInitialUserForm(user))
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const initialFormSeed = JSON.stringify(getInitialUserForm(user))
  const errorMessage = resolveLocalizedText(t, error)

  React.useEffect(() => {
    setForm(getInitialUserForm(user))
    setError(null)
  }, [initialFormSeed])

  const toggleRoute = (routeID: string) => {
    setForm((current) => {
      const isSelected = current.route_ids.includes(routeID)
      const newRouteIDs = isSelected
        ? current.route_ids.filter((id) => id !== routeID)
        : [...current.route_ids, routeID]
      const newRoutePaths = { ...current.route_paths }
      if (isSelected) {
        delete newRoutePaths[routeID]
      }
      return { ...current, route_ids: newRouteIDs, route_paths: newRoutePaths }
    })
  }

  const setRoutePaths = (routeID: string, pathsStr: string) => {
    setForm((current) => {
      const newRoutePaths = { ...current.route_paths }
      if (pathsStr.trim() === '') {
        delete newRoutePaths[routeID]
      } else {
        newRoutePaths[routeID] = pathsStr.split(',').map((p) => p.trim()).filter(Boolean)
      }
      return { ...current, route_paths: newRoutePaths }
    })
  }

  const toggleGroup = (groupID: string) => {
    setForm((current) => ({
      ...current,
      group_ids: (current.group_ids || []).includes(groupID)
        ? (current.group_ids || []).filter((id) => id !== groupID)
        : [...(current.group_ids || []), groupID],
    }))
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) {
      return
    }

    const submitToken = Symbol('user-form-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setSubmitting(true)
    try {
      await onSubmit(form)
    } catch (err) {
      setError(getLocalizedTextState(err))
    } finally {
      if (activeSubmitRef.current === submitToken) {
        activeSubmitRef.current = null
        setSubmitting(false)
      }
    }
  }

  const needsRouteAssignments = form.role === 'member'
  const roleOptions = [
    { value: 'member', label: t('roles.member') },
    { value: 'viewer', label: t('roles.viewer') },
    { value: 'editor', label: t('roles.editor') },
    { value: 'admin', label: t('roles.admin') },
  ]

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {errorMessage && <Alert variant="error">{errorMessage}</Alert>}

      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.identityEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.identityDescription')}
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.username')}
            value={form.username}
            onChange={(event) => setForm({ ...form, username: event.target.value })}
            placeholder="alice"
            required
          />
          <Select
            label={t('form.role')}
            value={form.role}
            onChange={(event) => setForm({ ...form, role: event.target.value })}
            options={roleOptions}
            hint={needsRouteAssignments ? t('form.memberHint') : t('form.operatorHint')}
            required
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={user ? t('form.newPassword') : t('form.password')}
            type="password"
            value={form.password || ''}
            onChange={(event) => setForm({ ...form, password: event.target.value })}
            placeholder={user ? t('form.newPasswordPlaceholder') : t('form.passwordPlaceholder')}
            required={!user}
            hint={user ? t('form.newPasswordHint') : undefined}
          />
          <Switch
            label={t('form.enabled')}
            description={t('form.enabledDescription')}
            checked={form.enabled}
            onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
          />
        </div>
      </Card>

      <Card tone="soft" className="space-y-4">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.accessEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.accessDescription')}
          </p>
        </div>

        {routes.length === 0 ? (
          routeListUnavailable ? (
            <Alert variant="warning">
              {t('form.routeListUnavailable')}
            </Alert>
          ) : (
            <div className="rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-5 text-sm text-[var(--text-muted)]">
              {t('form.noRoutes')}
            </div>
          )
        ) : (
          <div className="space-y-3">
            {routes.map((route) => {
              const checked = form.route_ids.includes(route.id)
              const pathsValue = form.route_paths?.[route.id]?.join(', ') || ''
              return (
                <div key={route.id}>
                  <label
                    className="flex items-center justify-between gap-4 rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-3 shadow-[var(--shadow-sm)]"
                  >
                    <div className="min-w-0">
                      <div className="text-sm font-semibold text-[var(--text-primary)]">{route.name || route.path_prefix}</div>
                      <div className="mt-1 text-xs text-[var(--text-muted)]">
                        {route.host || t('form.allHosts')} · {route.path_prefix}
                      </div>
                    </div>
                    <input
                      type="checkbox"
                      className="h-4 w-4 rounded border-[var(--border-default)] text-[var(--primary-600)]"
                      checked={checked}
                      onChange={() => toggleRoute(route.id)}
                    />
                  </label>
                  {checked && (
                    <div className="mt-1 ml-4">
                      <Input
                        label={t('form.allowedPaths')}
                        value={pathsValue}
                        onChange={(event) => setRoutePaths(route.id, event.target.value)}
                        placeholder={t('form.allowedPathsPlaceholder')}
                        hint={t('form.allowedPathsHint')}
                      />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </Card>

      {groups.length > 0 && (
        <Card tone="soft" className="space-y-4">
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
              {t('form.groupsEyebrow')}
            </div>
            <p className="mt-2 text-sm text-[var(--text-muted)]">
              {t('form.groupsDescription')}
            </p>
          </div>

          <div className="space-y-3">
            {groups.map((group) => {
              const checked = (form.group_ids || []).includes(group.id)
              const routeCount = group.route_ids?.length ?? Object.keys(group.route_paths || {}).length
              return (
                <label
                  key={group.id}
                  className="flex items-center justify-between gap-4 rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-3 shadow-[var(--shadow-sm)]"
                >
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-[var(--text-primary)]">{group.name}</div>
                    <div className="mt-1 text-xs text-[var(--text-muted)]">
                      {routeCount} {routeCount === 1 ? 'route' : 'routes'}
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-[var(--border-default)] text-[var(--primary-600)]"
                    checked={checked}
                    onChange={() => toggleGroup(group.id)}
                  />
                </label>
              )
            })}
          </div>
        </Card>
      )}

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto" loading={submitting}>
          {user ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
