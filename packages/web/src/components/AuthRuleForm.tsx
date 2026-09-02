import React from 'react'
import { useTranslation } from 'react-i18next'
import { getLocalizedTextState, resolveLocalizedText, type LocalizedTextState } from '../lib/error-state'
import { normalizeCORSMethods } from '../lib/cors'
import type { AuthRule, AuthRuleInput, Route } from '../lib/api/types'
import { Alert, Button, Card, Input, Select, Switch } from './ui'

interface AuthRuleFormProps {
  rule: AuthRule | null
  routes: Route[]
  defaultRouteId?: string
  onSubmit: (data: AuthRuleInput) => Promise<void> | void
  onCancel: () => void
}

function normalizeListValue(value: string): string[] {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function normalizeCommaSeparatedValue(value: string): string {
  return normalizeListValue(value).join(',')
}

function normalizeOptionalNumber(value: number | undefined): number | undefined {
  return value && value > 0 ? value : undefined
}

function parseOptionalNonNegativeInteger(value: string): number | undefined | null {
  if (value.trim() === '') return undefined
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : null
}

type NumericDrafts = {
  rateLimit: string
  burst: string
  corsMaxAge: string
}

type PolicyFieldErrors = Partial<Record<'rateLimit' | 'burst' | 'corsMaxAge' | 'corsAllowedMethods', string>>

function createNumericDrafts(form: AuthRuleInput): NumericDrafts {
  return {
    rateLimit: form.rate_limit == null ? '' : String(form.rate_limit),
    burst: form.burst == null ? '' : String(form.burst),
    corsMaxAge: form.cors_max_age == null ? '' : String(form.cors_max_age),
  }
}

function getInitialAuthRuleForm(rule: AuthRule | null, routes: Route[], defaultRouteId?: string): AuthRuleInput {
  return {
    route_id: rule?.route_id || defaultRouteId || routes[0]?.id || '',
    type: rule?.type || 'none',
    config: {
      header_name: rule?.config?.header_name || 'X-API-Key',
      secret: '',
      username: rule?.config?.username || '',
      password: '',
      login_mode: rule?.config?.login_mode || '',
    },
    whitelist: rule?.whitelist || [],
    rate_limit: normalizeOptionalNumber(rule?.rate_limit),
    burst: normalizeOptionalNumber(rule?.burst),
    cors_allowed_origins: rule?.cors_allowed_origins || '',
    cors_allowed_methods: rule?.cors_allowed_methods || '',
    cors_allowed_headers: rule?.cors_allowed_headers || '',
    cors_allow_credentials: rule?.cors_allow_credentials || false,
    cors_max_age: normalizeOptionalNumber(rule?.cors_max_age),
  }
}

export function AuthRuleForm({ rule, routes, defaultRouteId, onSubmit, onCancel }: AuthRuleFormProps) {
  const { t } = useTranslation('authRules')
  const initialForm = getInitialAuthRuleForm(rule, routes, defaultRouteId)
  const [form, setForm] = React.useState<AuthRuleInput>(() => initialForm)
  const [whitelistText, setWhitelistText] = React.useState(() => (initialForm.whitelist || []).join(', '))
  const [numericDrafts, setNumericDrafts] = React.useState<NumericDrafts>(() => createNumericDrafts(initialForm))
  const [fieldErrors, setFieldErrors] = React.useState<PolicyFieldErrors>({})
  const [error, setError] = React.useState<LocalizedTextState>(null)
  const [submitting, setSubmitting] = React.useState(false)
  const activeSubmitRef = React.useRef<symbol | null>(null)
  const initialFormSeed = JSON.stringify(initialForm)
  const errorMessage = resolveLocalizedText(t, error)

  React.useEffect(() => {
    setForm(initialForm)
    setWhitelistText((initialForm.whitelist || []).join(', '))
    setNumericDrafts(createNumericDrafts(initialForm))
    setFieldErrors({})
    setError(null)
  }, [initialFormSeed])

  const type = form.type || 'none'
  const config = form.config || {}
  const isEditingSameType = rule?.type === type

  const updateConfig = (next: Partial<AuthRuleInput['config']>) => {
    setForm((current) => ({
      ...current,
      config: {
        ...(current.config || {}),
        ...next,
      },
    }))
  }

  const updatePolicy = (next: Partial<Omit<AuthRuleInput, 'route_id' | 'type' | 'config'>>) => {
    setForm((current) => ({
      ...current,
      ...next,
    }))
  }

  const buildSubmitPayload = (
    rateLimit: number | undefined,
    burst: number | undefined,
    corsMaxAge: number | undefined,
    corsAllowedMethods: string
  ): AuthRuleInput => {
    const routeID = form.route_id.trim()
    const headerName = (config.header_name || '').trim()
    const username = (config.username || '').trim()
    const password = config.password || ''
    const hasPassword = password.trim() !== ''
    const loginMode = (config.login_mode || '').trim()
    const whitelist = form.whitelist || []
    const initialWhitelist = initialForm.whitelist || []
    const initialRateLimit = initialForm.rate_limit
    const initialBurst = initialForm.burst
    const corsAllowedOrigins = normalizeCommaSeparatedValue(form.cors_allowed_origins || '')
    const initialCORSAllowedOrigins = normalizeCommaSeparatedValue(initialForm.cors_allowed_origins || '')
    const initialCORSAllowedMethods = normalizeCORSMethods(initialForm.cors_allowed_methods || '') || ''
    const corsAllowedHeaders = normalizeCommaSeparatedValue(form.cors_allowed_headers || '')
    const initialCORSAllowedHeaders = normalizeCommaSeparatedValue(initialForm.cors_allowed_headers || '')
    const corsAllowCredentials = !!form.cors_allow_credentials
    const initialCORSAllowCredentials = !!initialForm.cors_allow_credentials
    const initialCORSMaxAge = initialForm.cors_max_age
    const runtimePolicy: Omit<AuthRuleInput, 'route_id' | 'type' | 'config'> = {
      ...(whitelist.length > 0 || initialWhitelist.length > 0 ? { whitelist } : {}),
      ...(rateLimit !== undefined || initialRateLimit !== undefined ? { rate_limit: rateLimit ?? 0 } : {}),
      ...(burst !== undefined || initialBurst !== undefined ? { burst: burst ?? 0 } : {}),
      ...(corsAllowedOrigins || initialCORSAllowedOrigins ? { cors_allowed_origins: corsAllowedOrigins } : {}),
      ...(corsAllowedMethods || initialCORSAllowedMethods ? { cors_allowed_methods: corsAllowedMethods } : {}),
      ...(corsAllowedHeaders || initialCORSAllowedHeaders ? { cors_allowed_headers: corsAllowedHeaders } : {}),
      ...(corsAllowCredentials || initialCORSAllowCredentials ? { cors_allow_credentials: corsAllowCredentials } : {}),
      ...(corsMaxAge !== undefined || initialCORSMaxAge !== undefined ? { cors_max_age: corsMaxAge ?? 0 } : {}),
    }

    switch (type) {
      case 'apikey':
        return {
          route_id: routeID,
          type,
          config: {
            header_name: headerName || 'X-API-Key',
          },
          ...runtimePolicy,
        }
      case 'gateway':
        return {
          route_id: routeID,
          type,
          config: {
            login_mode: loginMode || 'form',
          },
          ...runtimePolicy,
        }
      default:
        return {
          route_id: routeID,
          type,
          config: {},
          ...runtimePolicy,
        }
    }
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) {
      return
    }

    const rateLimit = parseOptionalNonNegativeInteger(numericDrafts.rateLimit)
    const burst = parseOptionalNonNegativeInteger(numericDrafts.burst)
    const corsMaxAge = parseOptionalNonNegativeInteger(numericDrafts.corsMaxAge)
    const corsAllowedMethods = normalizeCORSMethods(form.cors_allowed_methods || '')
    const nextFieldErrors: PolicyFieldErrors = {}
    const numberError = t('form.nonNegativeIntegerError')
    if (rateLimit === null) nextFieldErrors.rateLimit = numberError
    if (burst === null) nextFieldErrors.burst = numberError
    if (corsMaxAge === null) nextFieldErrors.corsMaxAge = numberError
    if (corsAllowedMethods === null) {
      nextFieldErrors.corsAllowedMethods = t('form.allowedMethodsError')
    }
    if (Object.keys(nextFieldErrors).length > 0) {
      setFieldErrors(nextFieldErrors)
      return
    }

    const submitToken = Symbol('auth-rule-form-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setFieldErrors({})
    setSubmitting(true)
    try {
      await onSubmit(buildSubmitPayload(rateLimit, burst, corsMaxAge, corsAllowedMethods))
    } catch (err) {
      setError(getLocalizedTextState(err))
    } finally {
      if (activeSubmitRef.current === submitToken) {
        activeSubmitRef.current = null
        setSubmitting(false)
      }
    }
  }

  const ruleTypeOptions = [
    { value: 'none', label: t('types.none') },
    { value: 'apikey', label: t('types.apikey') },
    { value: 'gateway', label: t('types.gateway') },
  ]

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-5">
      {errorMessage && <Alert variant="error">{errorMessage}</Alert>}

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
          disabled={!!defaultRouteId && !rule}
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
        <Card tone="soft" className="space-y-4">
          <Input
            label={t('form.headerName')}
            value={config.header_name || ''}
            onChange={(event) => updateConfig({ header_name: event.target.value })}
            placeholder="X-API-Key"
            required
          />
          {rule?.config?.secret && (
            <div className="space-y-1">
              <div className="text-xs font-medium text-[var(--text-muted)]">{t('form.apiKey')}</div>
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded bg-[var(--surface-inset)] px-3 py-2 text-xs break-all text-[var(--text-primary)]">
                  {rule.config.secret}
                </code>
                <Button
                  variant="ghost"
                  type="button"
                  onClick={() => navigator.clipboard.writeText(rule.config.secret!)}
                  className="shrink-0"
                >
                  {t('form.copyKey')}
                </Button>
              </div>
              <p className="text-xs text-[var(--text-muted)]">{t('form.apiKeyHint')}</p>
            </div>
          )}
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

      <Card tone="soft" className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Input
          label={t('form.whitelist')}
          value={whitelistText}
          onChange={(event) => {
            setWhitelistText(event.target.value)
            updatePolicy({ whitelist: normalizeListValue(event.target.value) })
          }}
          placeholder={t('form.whitelistPlaceholder')}
          hint={t('form.whitelistHint')}
        />
        <Input
          label={t('form.rateLimit')}
          type="number"
          min={0}
          step={1}
          value={numericDrafts.rateLimit}
          onChange={(event) => {
            setNumericDrafts((current) => ({ ...current, rateLimit: event.target.value }))
            setFieldErrors((current) => ({ ...current, rateLimit: undefined }))
          }}
          error={fieldErrors.rateLimit}
          placeholder="15"
          hint={t('form.rateLimitHint')}
        />
        <Input
          label={t('form.burst')}
          type="number"
          min={0}
          step={1}
          value={numericDrafts.burst}
          onChange={(event) => {
            setNumericDrafts((current) => ({ ...current, burst: event.target.value }))
            setFieldErrors((current) => ({ ...current, burst: undefined }))
          }}
          error={fieldErrors.burst}
          placeholder="30"
          hint={t('form.burstHint')}
        />
      </Card>

      <Card tone="soft" className="space-y-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Input
            label={t('form.allowedOrigins')}
            value={form.cors_allowed_origins || ''}
            onChange={(event) => updatePolicy({ cors_allowed_origins: event.target.value })}
            placeholder="https://app.example.com, .example.com"
            hint={t('form.allowedOriginsHint')}
          />
          <Input
            label={t('form.allowedMethods')}
            value={form.cors_allowed_methods || ''}
            onChange={(event) => {
              updatePolicy({ cors_allowed_methods: event.target.value })
              setFieldErrors((current) => ({ ...current, corsAllowedMethods: undefined }))
            }}
            error={fieldErrors.corsAllowedMethods}
            placeholder="GET,POST,OPTIONS"
            hint={t('form.allowedMethodsHint')}
          />
          <Input
            label={t('form.allowedHeaders')}
            value={form.cors_allowed_headers || ''}
            onChange={(event) => updatePolicy({ cors_allowed_headers: event.target.value })}
            placeholder="Authorization,Content-Type"
            hint={t('form.allowedHeadersHint')}
          />
          <Input
            label={t('form.maxAge')}
            type="number"
            min={0}
            step={1}
            value={numericDrafts.corsMaxAge}
            onChange={(event) => {
              setNumericDrafts((current) => ({ ...current, corsMaxAge: event.target.value }))
              setFieldErrors((current) => ({ ...current, corsMaxAge: undefined }))
            }}
            error={fieldErrors.corsMaxAge}
            placeholder="7200"
            hint={t('form.maxAgeHint')}
          />
        </div>

        <Switch
          label={t('form.allowCredentials')}
          description={t('form.allowCredentialsHint')}
          checked={!!form.cors_allow_credentials}
          onChange={(event) => updatePolicy({ cors_allow_credentials: event.target.checked })}
        />
      </Card>

      <div className="flex flex-col-reverse justify-end gap-2 border-t border-[var(--border-default)] pt-4 md:flex-row">
        <Button variant="ghost" onClick={onCancel} className="w-full md:w-auto">
          {t('common:actions.cancel')}
        </Button>
        <Button type="submit" className="w-full md:w-auto" loading={submitting}>
          {rule ? t('form.update') : t('form.create')}
        </Button>
      </div>
    </form>
  )
}
