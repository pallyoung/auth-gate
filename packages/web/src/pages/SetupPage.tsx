import React from 'react'
import { ArrowRight, CheckCircle, Lock, Shield, User } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import type { LoginResponse } from '../lib/api/types'
import { ApiError } from '../lib/api/client'
import { Button } from '../components/ui'

interface SetupPageProps {
  onSetup: (username: string, password: string) => Promise<LoginResponse>
}

export function SetupPage({ onSetup }: SetupPageProps) {
  const { t } = useTranslation('setup')
  const [username, setUsername] = React.useState('admin')
  const [password, setPassword] = React.useState('')
  const [confirmPassword, setConfirmPassword] = React.useState('')
  const [error, setError] = React.useState<string | null>(null)
  const [loading, setLoading] = React.useState(false)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError(null)

    if (password.length < 8) {
      setError(t('errors.passwordTooShort'))
      return
    }
    if (password !== confirmPassword) {
      setError(t('errors.passwordMismatch'))
      return
    }

    setLoading(true)
    try {
      await onSetup(username.trim(), password)
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === 'setup_already_completed') {
          setError(t('errors.setupAlreadyCompleted'))
        } else {
          setError(err.message)
        }
      } else {
        setError(t('errors.network'))
      }
    } finally {
      setLoading(false)
    }
  }

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
        {/* Setup Card */}
        <div className="glass-panel rounded-[20px] p-8">
          <div className="flex flex-col items-center text-center">
            {/* Shield Logo */}
            <div className="animate-pulse-glow flex h-16 w-16 items-center justify-center rounded-[18px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] text-white shadow-[0_0_30px_rgba(15,143,139,0.3)]">
              <Shield className="h-8 w-8" />
            </div>

            <h1 className="mt-5 text-2xl font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
              Auth Gate
            </h1>
            <p className="mt-1 text-xs font-medium uppercase tracking-[0.2em] text-[var(--text-muted)]">
              Initial Setup
            </p>
          </div>

          <form onSubmit={handleSubmit} className="mt-8 space-y-4">
            {error && (
              <div
                className="rounded-[12px] border border-[rgba(248,113,113,0.2)] bg-[var(--error-light)] px-4 py-3 text-sm font-medium text-[var(--error)]"
                role="alert"
              >
                {error}
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
                  autoComplete="new-password"
                  placeholder={t('fields.passwordPlaceholder')}
                  required
                  className="w-full rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)] py-3 pl-10 pr-4 text-sm text-[var(--text-primary)] placeholder-[var(--text-subtle)] transition-colors focus:border-[var(--primary-500)] focus:outline-none focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]"
                />
              </div>
            </div>

            <div>
              <label className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                {t('fields.confirmPassword')}
              </label>
              <div className="relative">
                <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-subtle)]">
                  <CheckCircle className="h-4 w-4" />
                </div>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  placeholder={t('fields.confirmPasswordPlaceholder')}
                  required
                  className="w-full rounded-[12px] border border-[var(--border-default)] bg-[var(--bg-input)] py-3 pl-10 pr-4 text-sm text-[var(--text-primary)] placeholder-[var(--text-subtle)] transition-colors focus:border-[var(--primary-500)] focus:outline-none focus:ring-2 focus:ring-[rgba(15,143,139,0.2)]"
                />
              </div>
            </div>

            {/* Setup Button */}
            <Button
              type="submit"
              className="w-full rounded-[12px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] py-3 text-sm font-semibold text-white shadow-[0_4px_14px_rgba(15,143,139,0.3)] hover:shadow-[0_6px_20px_rgba(15,143,139,0.4)]"
              size="lg"
              loading={loading}
            >
              <span className="flex items-center justify-center gap-2">
                {t('card.submit')}
                <ArrowRight className="h-4 w-4" />
              </span>
            </Button>
          </form>
        </div>

        {/* Version Pill */}
        <div className="mt-6 flex justify-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border-default)] bg-[var(--bg-card)] px-4 py-2 text-[10px] font-medium uppercase tracking-[0.1em] text-[var(--text-muted)]">
            <Shield className="h-3 w-3" />
            <span>Auth Gate v2.4.0</span>
            <span className="text-[var(--text-subtle)]">•</span>
            <span>Enterprise Security</span>
          </div>
        </div>
      </div>
    </div>
  )
}
