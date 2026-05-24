import React from 'react'
import { useTranslation } from 'react-i18next'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'
import { Button, Card, Input, Select } from './ui'

interface AuthRuleFormProps {
  rule: AuthRule | null
  routes: Route[]
  onSubmit: (data: AuthRuleInput) => Promise<void> | void
  onCancel: () => void
}

export function AuthRuleForm({ rule, routes, onSubmit, onCancel }: AuthRuleFormProps) {
  const { t } = useTranslation('authRules')
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

  const ruleTypeOptions = [
    { value: 'none', label: t('types.none') },
    { value: 'apikey', label: t('types.apikey') },
    { value: 'bearer', label: t('types.bearer') },
    { value: 'basic', label: t('types.basic') },
    { value: 'gateway', label: t('types.gateway') },
  ]

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <Card tone="soft" className="space-y-5">
        <div>
          <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
            {t('form.scopeEyebrow')}
          </div>
          <p className="mt-2 text-sm text-[var(--text-muted)]">
            {t('form.scopeDescription')}
          </p>
        </div>

        <Select
          label={t('form.route')}
          value={form.route_id}
          onChange={(event) => setForm({ ...form, route_id: event.target.value })}
          options={routes.map((route) => ({
            value: route.id,
            label: route.name || route.path_prefix,
          }))}
          required
        />

        <Select
          label={t('form.type')}
          value={type}
          onChange={(event) => setForm({ ...form, type: event.target.value as AuthRule['type'] })}
          options={ruleTypeOptions}
          required
        />
      </Card>

      {type === 'apikey' && (
        <Card tone="soft" className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.headerName')}
            value={config.header_name || ''}
            onChange={(event) => updateConfig({ header_name: event.target.value })}
            placeholder="X-API-Key"
            required
          />
          <Input
            label={t('form.secret')}
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
            label={t('form.jwtSecret')}
            value={config.secret || ''}
            onChange={(event) => updateConfig({ secret: event.target.value })}
            placeholder="jwt-signing-secret"
            hint={t('form.jwtSecretHint')}
            required
          />
        </Card>
      )}

      {type === 'basic' && (
        <Card tone="soft" className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.username')}
            value={config.username || ''}
            onChange={(event) => updateConfig({ username: event.target.value })}
            placeholder={t('form.usernamePlaceholder')}
            required
          />
          <Input
            label={t('form.password')}
            type="password"
            value={config.password || ''}
            onChange={(event) => updateConfig({ password: event.target.value })}
            placeholder={t('form.passwordPlaceholder')}
            required
          />
        </Card>
      )}

      {type === 'gateway' && (
        <Card tone="soft">
          <Select
            label={t('form.loginMode')}
            value={config.login_mode || 'form'}
            onChange={(event) => updateConfig({ login_mode: event.target.value })}
            options={[{ value: 'form', label: t('types.gatewayForm') }]}
            hint={t('form.loginModeHint')}
            required
          />
        </Card>
      )}

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto">
          {rule ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
