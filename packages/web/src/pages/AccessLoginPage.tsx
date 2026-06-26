import React from 'react'
import { ArrowRight, KeyRound, Lock, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import { authApi } from '../lib/api/auth'
import { ApiError } from '../lib/api/client'
import { navigateTo } from '../lib/navigation'
import { Button } from '../components/ui'

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

      {/* Background gradient blobs */}
      <div className="absolute left-[15%] top-[20%] h-64 w-64 rounded-full bg-[rgba(15,143,139,0.15)] blur-[100px]" />
      <div className="absolute bottom-[20%] right-[15%] h-72 w-72 rounded-full bg-[rgba(47,114,200,0.12)] blur-[100px]" />
      <div className="absolute left-[60%] top-[10%] h-48 w-48 rounded-full bg-[rgba(189,122,24,0.1)] blur-[80px]" />

      <div className="relative z-10 w-full max-w-md">
        {/* Login Card */}
        <div className="glass-panel rounded-[20px] p-8">
          <div className="flex flex-col items-center text-center">
            {/* Key Icon */}
            <div className="animate-pulse-glow flex h-16 w-16 items-center justify-center rounded-[18px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] text-white shadow-[0_0_30px_rgba(15,143,139,0.3)]">
              <KeyRound className="h-8 w-8" />
            </div>

            <h1 className="mt-5 text-2xl font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
              {t('title')}
            </h1>
            <p className="mt-1 text-xs font-medium uppercase tracking-[0.2em] text-[var(--text-muted)]">
              {t('description', { routeName })}
            </p>
            <div className="mt-3 inline-flex items-center rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-3 py-1 text-xs font-medium text-[var(--text-muted)]">
              {pathPrefix}
            </div>
          </div>

          <form onSubmit={handleSubmit} className="mt-8 space-y-4">
            {displayError && (
              <div
                className="rounded-[12px] border border-[rgba(248,113,113,0.2)] bg-[var(--error-light)] px-4 py-3 text-sm font-medium text-[var(--error)]"
                role="alert"
              >
                {displayError}
              </div>
            )}

            <div>
              <label className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {t('fields.username')}
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-subtle)]">
                  <User className="h-4 w-4" />
                </div>
                <input
                  type="text"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  autoComplete="username"
                  placeholder={t('fields.usernamePlaceholder')}
                  required
                  className="w-full rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)] py-3 pl-10 pr-4 text-sm text-[var(--text-primary)] placeholder-[var(--text-subtle)] transition-colors focus:border-[var(--primary-500)] focus:outline-none focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]"
                />
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {t('fields.password')}
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-subtle)]">
                  <Lock className="h-4 w-4" />
                </div>
                <input
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="current-password"
                  placeholder={t('fields.passwordPlaceholder')}
                  required
                  className="w-full rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)] py-3 pl-10 pr-4 text-sm text-[var(--text-primary)] placeholder-[var(--text-subtle)] transition-colors focus:border-[var(--primary-500)] focus:outline-none focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]"
                />
              </div>
            </div>

            {/* Sign In Button */}
            <Button
              type="submit"
              className="w-full rounded-[12px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] py-3 text-sm font-semibold text-white shadow-[0_4px_14px_rgba(15,143,139,0.3)] hover:shadow-[0_6px_20px_rgba(15,143,139,0.4)]"
              size="lg"
              loading={loading}
              disabled={!routeId || routeUnavailable}
            >
              <span className="flex items-center justify-center gap-2">
                {t('submit')}
                <ArrowRight className="h-4 w-4" />
              </span>
            </Button>
          </form>
        </div>

        {/* Version Pill */}
        <div className="mt-6 flex justify-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-2 text-[10px] font-medium uppercase tracking-[0.1em] text-[var(--text-muted)]">
            <KeyRound className="h-3 w-3" />
            <span>Protected Route</span>
          </div>
        </div>
      </div>
    </div>
  )
}
