import React from 'react'
import { useTranslation } from 'react-i18next'
import type { Route, RouteInput } from '../lib/api/types'
import { Button, Card, Input, Switch } from './ui'

interface RouteFormProps {
  route: Route | null
  onSubmit: (data: RouteInput) => void
  onCancel: () => void
}

export function RouteForm({ route, onSubmit, onCancel }: RouteFormProps) {
  const { t } = useTranslation('routes')
  const [form, setForm] = React.useState<RouteInput>({
    name: route?.name || '',
    host: route?.host || '',
    path_prefix: route?.path_prefix || '',
    backend: route?.backend || '',
    strip_prefix: route?.strip_prefix ?? true,
    enabled: route?.enabled ?? true,
    priority: route?.priority || 0,
    path_match_mode: route?.path_match_mode || '',
    rewrite_target: route?.rewrite_target || '',
    redirect_code: route?.redirect_code || 0,
  })

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault()
    onSubmit(form)
  }

  const pathMatchModes = [
    {
      value: '',
      label: t('pathMatchModes.plainPrefix.label'),
      hint: t('pathMatchModes.plainPrefix.hint'),
    },
    {
      value: 'exact',
      label: t('pathMatchModes.exact.label'),
      hint: t('pathMatchModes.exact.hint'),
    },
    {
      value: 'stop',
      label: t('pathMatchModes.stop.label'),
      hint: t('pathMatchModes.stop.hint'),
    },
    {
      value: 'regex',
      label: t('pathMatchModes.regex.label'),
      hint: t('pathMatchModes.regex.hint'),
    },
    {
      value: 'regex_i',
      label: t('pathMatchModes.regexInsensitive.label'),
      hint: t('pathMatchModes.regexInsensitive.hint'),
    },
  ]

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
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
            label={t('form.name')}
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder={t('form.namePlaceholder')}
          />
          <Input
            label={t('form.host')}
            value={form.host}
            onChange={(event) => setForm({ ...form, host: event.target.value })}
            placeholder={t('form.hostPlaceholder')}
            hint={t('form.hostHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.pathPrefix')}
            value={form.path_prefix}
            onChange={(event) => setForm({ ...form, path_prefix: event.target.value })}
            placeholder={t('form.pathPrefixPlaceholder')}
            hint={t('form.pathPrefixHint')}
          />
          <Input
            label={t('form.backend')}
            value={form.backend}
            onChange={(event) => setForm({ ...form, backend: event.target.value })}
            placeholder={t('form.backendPlaceholder')}
            required
          />
        </div>

        <Input
          label={t('form.priority')}
          type="number"
          value={form.priority}
          onChange={(event) => setForm({ ...form, priority: parseInt(event.target.value, 10) || 0 })}
          hint={t('form.priorityHint')}
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-[var(--text-primary)]">
              {t('form.pathMatchMode')}
            </label>
            <select
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
              value={form.path_match_mode || ''}
              onChange={(e) => setForm({ ...form, path_match_mode: e.target.value })}
            >
              {pathMatchModes.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
            <p className="mt-1 text-xs text-[var(--text-muted)]">
              {pathMatchModes.find((m) => m.value === (form.path_match_mode || ''))?.hint || pathMatchModes[0].hint}
            </p>
          </div>
          <Input
            label={t('form.rewriteTarget')}
            value={form.rewrite_target || ''}
            onChange={(event) => setForm({ ...form, rewrite_target: event.target.value })}
            placeholder={t('form.rewriteTargetPlaceholder')}
            hint={t('form.rewriteTargetHint')}
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-[var(--text-primary)]">
              {t('form.redirectCode')}
            </label>
            <select
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
              value={form.redirect_code || 0}
              onChange={(e) => setForm({ ...form, redirect_code: parseInt(e.target.value, 10) || 0 })}
            >
              <option value={0}>{t('form.noRedirect')}</option>
              <option value={301}>{t('form.movedPermanently')}</option>
              <option value={302}>{t('form.foundTemporary')}</option>
            </select>
            <p className="mt-1 text-xs text-[var(--text-muted)]">
              {t('form.redirectHint')}
            </p>
          </div>
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Switch
          label={t('form.stripPrefix')}
          description={t('form.stripPrefixDescription')}
          checked={form.strip_prefix}
          onChange={(event) => setForm({ ...form, strip_prefix: event.target.checked })}
        />
        <Switch
          label={t('form.enabled')}
          description={t('form.enabledDescription')}
          checked={form.enabled}
          onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
        />
      </div>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {route ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
