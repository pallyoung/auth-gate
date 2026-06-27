import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import type { PermissionGroup, PermissionGroupInput, Route } from '../lib/api/types'
import { Alert, Button, Card, Input } from './ui'

interface GroupFormProps {
  group: PermissionGroup | null
  routes: Route[]
  onSubmit: (data: PermissionGroupInput) => Promise<void> | void
  onCancel: () => void
}

function getInitialForm(group: PermissionGroup | null): PermissionGroupInput {
  return {
    name: group?.name || '',
    route_ids: group?.route_ids || [],
    route_paths: group?.route_paths || {},
  }
}

export function GroupForm({ group, routes, onSubmit, onCancel }: GroupFormProps) {
  const { t } = useTranslation('groups')
  const [form, setForm] = React.useState<PermissionGroupInput>(() => getInitialForm(group))
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const initialFormSeed = JSON.stringify(getInitialForm(group))
  const errorMessage = resolveLocalizedText(t, error)

  React.useEffect(() => {
    setForm(getInitialForm(group))
    setError(null)
  }, [initialFormSeed])

  const toggleRoute = (routeID: string) => {
    setForm((current) => {
      const ids = current.route_ids || []
      const isSelected = ids.includes(routeID)
      const newRouteIDs = isSelected
        ? ids.filter((id) => id !== routeID)
        : [...ids, routeID]
      const newRoutePaths = { ...(current.route_paths || {}) }
      if (isSelected) {
        delete newRoutePaths[routeID]
      }
      return { ...current, route_ids: newRouteIDs, route_paths: newRoutePaths }
    })
  }

  const setRoutePaths = (routeID: string, pathsStr: string) => {
    setForm((current) => {
      const newRoutePaths = { ...(current.route_paths || {}) }
      if (pathsStr.trim() === '') {
        delete newRoutePaths[routeID]
      } else {
        newRoutePaths[routeID] = pathsStr.split(',').map((p) => p.trim()).filter(Boolean)
      }
      return { ...current, route_paths: newRoutePaths }
    })
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) return

    const submitToken = Symbol('group-form-submit')
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

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {errorMessage && <Alert variant="error">{errorMessage}</Alert>}

      <Card tone="soft" className="space-y-5">
        <Input
          label={t('form.name')}
          value={form.name}
          onChange={(event) => setForm({ ...form, name: event.target.value })}
          placeholder={t('form.namePlaceholder')}
          required
        />
      </Card>

      <Card tone="soft" className="space-y-4">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.routesEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.routesDescription')}
          </p>
        </div>

        {routes.length === 0 ? (
          <div className="rounded-[18px] border border-[var(--border-default)] bg-[var(--bg-card-soft)] px-4 py-5 text-sm text-[var(--text-muted)]">
            {t('form.noRoutes')}
          </div>
        ) : (
          <div className="space-y-3">
            {routes.map((route) => {
              const checked = (form.route_ids || []).includes(route.id)
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

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto" loading={submitting}>
          {group ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
