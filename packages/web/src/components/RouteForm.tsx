import React from 'react'
import type { Route, RouteInput } from '../lib/api/types'
import { Button, Card, Input, Switch } from './ui'

// Path match modes with nginx syntax hints.
export const PATH_MATCH_MODES = [
  { value: '',          label: 'Plain Prefix',       hint: '/prefix — longest-prefix match' },
  { value: 'exact',     label: 'Exact (=)',          hint: '= /path — exact path only' },
  { value: 'stop',      label: 'Prefix Stop (^~)',    hint: '^~ /prefix — stop regex scan' },
  { value: 'regex',     label: 'Regex (~)',           hint: '~ ^/api/v\\d+ — case-sensitive' },
  { value: 'regex_i',   label: 'Regex Insensitive (~*)', hint: '~* \\.(png|jpg)$ — case-insensitive' },
]

interface RouteFormProps {
  route: Route | null
  onSubmit: (data: RouteInput) => void
  onCancel: () => void
}

export function RouteForm({ route, onSubmit, onCancel }: RouteFormProps) {
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

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Route Identity
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Define the matching scope and backend target for this route.
          </p>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Name"
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            placeholder="Billing API"
          />
          <Input
            label="Host"
            value={form.host}
            onChange={(event) => setForm({ ...form, host: event.target.value })}
            placeholder="api.example.com"
            hint="Leave empty to match every host."
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Path Prefix"
            value={form.path_prefix}
            onChange={(event) => setForm({ ...form, path_prefix: event.target.value })}
            placeholder="/billing or empty to match all"
            hint="Leave empty to match all paths."
          />
          <Input
            label="Backend"
            value={form.backend}
            onChange={(event) => setForm({ ...form, backend: event.target.value })}
            placeholder="http://127.0.0.1:3000"
            required
          />
        </div>

        <Input
          label="Priority"
          type="number"
          value={form.priority}
          onChange={(event) => setForm({ ...form, priority: parseInt(event.target.value, 10) || 0 })}
          hint="Higher values win when multiple routes can match the same request."
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-[var(--text-primary)]">
              Path Match Mode
            </label>
            <select
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
              value={form.path_match_mode || ''}
              onChange={(e) => setForm({ ...form, path_match_mode: e.target.value })}
            >
              {PATH_MATCH_MODES.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
            <p className="mt-1 text-xs text-[var(--text-muted)]">
              {PATH_MATCH_MODES.find((m) => m.value === (form.path_match_mode || ''))?.hint
                || PATH_MATCH_MODES[0].hint}
            </p>
          </div>
          <Input
            label="Rewrite Target"
            value={form.rewrite_target || ''}
            onChange={(event) => setForm({ ...form, rewrite_target: event.target.value })}
            placeholder="/new/$1"
            hint="Use $1, $2 for regex capture groups. $& for full match."
          />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-[var(--text-primary)]">
              Redirect Code
            </label>
            <select
              className="w-full rounded-lg border border-[var(--border-default)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)] focus:outline-none focus:ring-1 focus:ring-[var(--accent)]"
              value={form.redirect_code || 0}
              onChange={(e) => setForm({ ...form, redirect_code: parseInt(e.target.value, 10) || 0 })}
            >
              <option value={0}>No redirect</option>
              <option value={301}>301 Moved Permanently</option>
              <option value={302}>302 Found (Temporary)</option>
            </select>
            <p className="mt-1 text-xs text-[var(--text-muted)]">
              Use with Rewrite Target for external redirects (e.g. HTTP → HTTPS).
            </p>
          </div>
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Switch
          label="Strip Prefix"
          description="Remove the incoming prefix before forwarding traffic to the backend."
          checked={form.strip_prefix}
          onChange={(event) => setForm({ ...form, strip_prefix: event.target.checked })}
        />
        <Switch
          label="Enabled"
          description="Keep the route active and available to the runtime router."
          checked={form.enabled}
          onChange={(event) => setForm({ ...form, enabled: event.target.checked })}
        />
      </div>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {route ? 'Update Route' : 'Create Route'}
        </Button>
      </div>
    </form>
  )
}
