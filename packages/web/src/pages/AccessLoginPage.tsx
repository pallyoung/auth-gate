import React from 'react'
import { KeyRound, Lock, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import { authApi } from '../lib/api/auth'
import { ApiError } from '../lib/api/client'
import { navigateTo } from '../lib/navigation'
import { Button, Card, Input } from '../components/ui'

interface AccessLoginPageProps {
  searchParams: URLSearchParams
}

type AccessLoginErrorState = {
  translationKey?: string
  message?: string
} | null

export function AccessLoginPage({ searchParams }: AccessLoginPageProps) {
  const { t } = useTranslation('accessLogin')
  const routeId = searchParams.get('route_id') || ''
  const rawRouteName = searchParams.get('route_name') || ''
  const routeName = rawRouteName || t('protectedRouteFallback')
  const pathPrefix = searchParams.get('path_prefix') || '/'
  const next = searchParams.get('next') || pathPrefix
  const routeContextKey = `${routeId}\n${rawRouteName}\n${pathPrefix}\n${next}`
  const routeContextError = !routeId ? t('errors.missingRoute') : ''

  const [username, setUsername] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [error, setError] = React.useState<AccessLoginErrorState>(null)
  const [loading, setLoading] = React.useState(false)
  const [routeUnavailable, setRouteUnavailable] = React.useState(false)
  const requestGenerationRef = React.useRef(0)
  const activeSubmitRef = React.useRef<symbol | null>(null)

  React.useEffect(() => {
    requestGenerationRef.current += 1
    activeSubmitRef.current = null
    setError(null)
    setRouteUnavailable(false)
    setLoading(false)
  }, [routeContextKey])

  const getErrorState = (err: ApiError): Exclude<AccessLoginErrorState, null> => {
    switch (err.code) {
      case 'route_not_found':
        return { translationKey: 'errors.routeUnavailable' }
      case 'invalid_credentials':
        return { translationKey: 'errors.invalidCredentials' }
      case 'route_access_denied':
        return { translationKey: 'errors.routeAccessDenied' }
      case 'user_disabled':
        return { translationKey: 'errors.userDisabled' }
      case 'session_store_failure':
      case 'token_generation_failed':
        return { translationKey: 'errors.sessionUnavailable' }
      default:
        return { message: err.message }
    }
  }

  const errorMessage = error?.translationKey
    ? t(error.translationKey as any)
    : error?.message || ''

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    if (activeSubmitRef.current) {
      return
    }
    if (!routeId) {
      setError({ translationKey: 'errors.missingRoute' })
      return
    }

    const submitToken = Symbol('access-login-submit')
    activeSubmitRef.current = submitToken
    setError(null)
    setRouteUnavailable(false)
    setLoading(true)
    const requestGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = requestGeneration

    try {
      const result = await authApi.accessLogin({
        route_id: routeId,
        username: username.trim(),
        password,
        next,
      })
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      navigateTo(result.next)
    } catch (err) {
      if (requestGenerationRef.current !== requestGeneration) {
        return
      }
      if (err instanceof ApiError) {
        if (err.code === 'route_not_found') {
          setRouteUnavailable(true)
        }
        setError(getErrorState(err))
      } else {
        setError({ translationKey: 'errors.network' })
      }
    } finally {
      if (requestGenerationRef.current === requestGeneration) {
        activeSubmitRef.current = null
        setLoading(false)
      } else if (activeSubmitRef.current === submitToken) {
        activeSubmitRef.current = null
      }
    }
  }

  const displayError = errorMessage || routeContextError

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-10">
      <div className="absolute right-4 top-4 z-10 sm:right-6 sm:top-6">
        <LanguageSwitcher />
      </div>
      <div className="absolute left-[8%] top-[12%] hidden h-40 w-40 rounded-full bg-[rgba(189,122,24,0.16)] blur-3xl md:block" />
      <div className="absolute bottom-[10%] right-[10%] hidden h-48 w-48 rounded-full bg-[rgba(15,143,139,0.16)] blur-3xl md:block" />

      <Card className="mx-auto w-full max-w-lg rounded-[32px] p-6 md:p-8">
        <div className="flex flex-col items-center text-center">
            <div className="animate-pulse-glow flex h-16 w-16 items-center justify-center rounded-[24px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-lg)]">
              <KeyRound className="h-8 w-8" />
            </div>
          <div className="eyebrow mt-5">{t('eyebrow')}</div>
          <h1 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[var(--text-primary)]">
            {t('title')}
          </h1>
          <p className="mt-2 max-w-sm text-sm leading-6 text-[var(--text-muted)]">
            {t('description', { routeName })}
          </p>
          <div className="mt-4 rounded-full bg-[rgba(255,255,255,0.6)] px-3 py-1 text-xs font-semibold uppercase tracking-[0.12em] text-[var(--text-muted)]">
            {pathPrefix}
          </div>
        </div>

        <form onSubmit={handleSubmit} className="mt-8 space-y-5">
          {displayError && (
            <div
              className="rounded-[20px] border border-[rgba(208,71,75,0.16)] bg-[var(--error-light)] px-4 py-3 text-sm font-medium text-[var(--error)]"
              role="alert"
            >
              {displayError}
            </div>
          )}

          <Input
            label={t('fields.username')}
            type="text"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            autoComplete="username"
            placeholder={t('fields.usernamePlaceholder')}
            leftIcon={<User className="h-4 w-4" />}
            required
          />

          <Input
            label={t('fields.password')}
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            placeholder={t('fields.passwordPlaceholder')}
            leftIcon={<Lock className="h-4 w-4" />}
            required
          />

          <Button type="submit" className="w-full" size="lg" loading={loading} disabled={!routeId || routeUnavailable}>
            {t('submit')}
          </Button>
        </form>
      </Card>
    </div>
  )
}
