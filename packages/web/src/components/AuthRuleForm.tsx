import React from 'react'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'
import { Button, Card, Input, Select } from './ui'

interface AuthRuleFormProps {
  rule: AuthRule | null
  routes: Route[]
  onSubmit: (data: AuthRuleInput) => Promise<void> | void
  onCancel: () => void
}

const ruleTypeOptions = [
  { value: 'none', label: 'None' },
  { value: 'apikey', label: 'API Key' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
]

export function AuthRuleForm({ rule, routes, onSubmit, onCancel }: AuthRuleFormProps) {
  const [form, setForm] = React.useState<AuthRuleInput>({
    route_id: rule?.route_id || routes[0]?.id || '',
    type: rule?.type || 'none',
    config: {
      header_name: rule?.config?.header_name || 'X-API-Key',
      secret: '',
      username: rule?.config?.username || '',
      password: '',
    },
  })

  const type = form.type || 'none'
  const config = form.config || {}

  const updateConfig = (next: Partial<AuthRuleInput['config']>) => {
    setForm((current) => ({
      ...current,
      config: {
        ...(current.config || {}),
        ...next,
      },
    }))
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    await onSubmit(form)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            Policy Scope
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            Attach an auth policy to a route and provide the credentials it should validate.
          </p>
        </div>

        <Select
          label="Route"
          value={form.route_id}
          onChange={(event) => setForm({ ...form, route_id: event.target.value })}
          options={routes.map((route) => ({
            value: route.id,
            label: route.name || route.path_prefix,
          }))}
          required
        />

        <Select
          label="Type"
          value={type}
          onChange={(event) => setForm({ ...form, type: event.target.value as AuthRule['type'] })}
          options={ruleTypeOptions}
          required
        />
      </Card>

      {type === 'apikey' && (
        <Card tone="soft" className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Header Name"
            value={config.header_name || ''}
            onChange={(event) => updateConfig({ header_name: event.target.value })}
            placeholder="X-API-Key"
            required
          />
          <Input
            label="Secret"
            value={config.secret || ''}
            onChange={(event) => updateConfig({ secret: event.target.value })}
            placeholder="secret-value"
            required
          />
        </Card>
      )}

      {type === 'bearer' && (
        <Card tone="soft">
          <Input
            label="JWT Secret"
            value={config.secret || ''}
            onChange={(event) => updateConfig({ secret: event.target.value })}
            placeholder="jwt-signing-secret"
            hint="Current implementation validates HMAC-signed JWTs with this shared secret."
            required
          />
        </Card>
      )}

      {type === 'basic' && (
        <Card tone="soft" className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label="Username"
            value={config.username || ''}
            onChange={(event) => updateConfig({ username: event.target.value })}
            placeholder="service-user"
            required
          />
          <Input
            label="Password"
            type="password"
            value={config.password || ''}
            onChange={(event) => updateConfig({ password: event.target.value })}
            placeholder="service-password"
            required
          />
        </Card>
      )}

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {rule ? 'Update Rule' : 'Create Rule'}
        </Button>
      </div>
    </form>
  )
}
