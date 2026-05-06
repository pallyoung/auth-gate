import React from 'react'
import { Input, Switch } from './ui'
import { Button } from './ui'
import type { Route, RouteInput } from '../lib/api/types'

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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(form)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 md:space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="Name"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="My Service"
        />
        <Input
          label="Host"
          value={form.host}
          onChange={(e) => setForm({ ...form, host: e.target.value })}
          placeholder="example.com"
          hint="Leave empty to match all hosts"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="Path Prefix"
          value={form.path_prefix}
          onChange={(e) => setForm({ ...form, path_prefix: e.target.value })}
          placeholder="/api"
          required
        />
        <Input
          label="Backend"
          value={form.backend}
          onChange={(e) => setForm({ ...form, backend: e.target.value })}
          placeholder="http://127.0.0.1:3000"
          required
        />
      </div>

      <Input
        label="Priority"
        type="number"
        value={form.priority}
        onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 0 })}
        hint="Higher priority routes are matched first"
      />

      <div className="flex gap-6 pt-2">
        <Switch
          label="Strip Prefix"
          checked={form.strip_prefix}
          onChange={(e) => setForm({ ...form, strip_prefix: e.target.checked })}
        />
        <Switch
          label="Enabled"
          checked={form.enabled}
          onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
        />
      </div>

      <div className="flex flex-col-reverse md:flex-row justify-end gap-2 pt-4 border-t border-[var(--border-default)]">
        <Button variant="secondary" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {route ? 'Update' : 'Create'}
        </Button>
      </div>
    </form>
  )
}
