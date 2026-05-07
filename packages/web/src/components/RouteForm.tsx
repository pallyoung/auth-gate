import React from 'react'
import type { Route, RouteInput } from '../lib/api/types'
import { Button, Card, Input, Switch } from './ui'

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
            placeholder="/billing"
            required
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
