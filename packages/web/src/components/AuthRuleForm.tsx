import React from 'react'
import { AuthRule, Route } from '../lib/api'
import { Button, Input, Select } from './ui'

interface AuthRuleFormProps {
  rule: AuthRule | null
  routes: Route[]
  onSubmit: (data: Partial<AuthRule>) => Promise<void> | void
  onCancel: () => void
}

const ruleTypeOptions = [
  { value: 'none', label: 'None' },
  { value: 'apikey', label: 'API Key' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
]

export function AuthRuleForm({ rule, routes, onSubmit, onCancel }: AuthRuleFormProps) {
  const [form, setForm] = React.useState<Partial<AuthRule>>({
    route_id: rule?.route_id || (routes[0]?.id ?? ''),
    type: rule?.type || 'none',
    config: {
      header_name: rule?.config?.header_name || 'X-API-Key',
      secret: rule?.config?.secret || '',
      username: rule?.config?.username || '',
      password: rule?.config?.password || '',
    },
    whitelist: rule?.whitelist || [],
    rate_limit: rule?.rate_limit || 0,
  })

  const type = form.type || 'none'
  const config = form.config || {}

  const updateConfig = (next: Partial<NonNullable<AuthRule['config']>>) => {
    setForm((current) => ({
      ...current,
      config: {
        ...(current.config || {}),
        ...next,
      },
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onSubmit(form)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 md:space-y-6">
      <Select
        label="Route"
        value={form.route_id}
        onChange={(e) => setForm({ ...form, route_id: e.target.value })}
        options={routes.map((route) => ({
          value: route.id,
          label: route.name || route.path_prefix,
        }))}
        required
      />

      <Select
        label="Type"
        value={type}
        onChange={(e) => setForm({ ...form, type: e.target.value as AuthRule['type'] })}
        options={ruleTypeOptions}
        required
      />

      {type === 'apikey' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Header Name"
            value={config.header_name || ''}
            onChange={(e) => updateConfig({ header_name: e.target.value })}
            placeholder="X-API-Key"
            required
          />
          <Input
            label="Secret"
            value={config.secret || ''}
            onChange={(e) => updateConfig({ secret: e.target.value })}
            placeholder="secret-value"
            required
          />
        </div>
      )}

      {type === 'bearer' && (
        <Input
          label="JWT Secret"
          value={config.secret || ''}
          onChange={(e) => updateConfig({ secret: e.target.value })}
          placeholder="jwt-signing-secret"
          hint="Current implementation validates HMAC-signed JWTs with this shared secret."
          required
        />
      )}

      {type === 'basic' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Username"
            value={config.username || ''}
            onChange={(e) => updateConfig({ username: e.target.value })}
            placeholder="service-user"
            required
          />
          <Input
            label="Password"
            type="password"
            value={config.password || ''}
            onChange={(e) => updateConfig({ password: e.target.value })}
            placeholder="service-password"
            required
          />
        </div>
      )}

      <div className="flex flex-col-reverse md:flex-row justify-end gap-2 pt-4 border-t border-[var(--border-default)]">
        <Button variant="secondary" onClick={onCancel} className="w-full md:w-auto">
          Cancel
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {rule ? 'Update' : 'Create'}
        </Button>
      </div>
    </form>
  )
}
