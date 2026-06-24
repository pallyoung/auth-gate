import React from 'react'
import { Lock, Shield, Sparkles, User, CheckCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import type { LoginResponse } from '../lib/api/types'
import { ApiError } from '../lib/api/client'
import { Button, Card, Input } from '../components/ui'

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
      <div className="absolute left-[8%] top-[12%] hidden h-40 w-40 rounded-full bg-[rgba(15,143,139,0.16)] blur-3xl md:block" />
      <div className="absolute bottom-[10%] right-[10%] hidden h-48 w-48 rounded-full bg-[rgba(189,122,24,0.16)] blur-3xl md:block" />

      <div className="grid w-full max-w-6xl items-center gap-6 lg:grid-cols-[1.15fr_0.85fr]">
        <Card
          tone="inverse"
          className="relative hidden overflow-hidden rounded-[36px] px-8 py-10 lg:block"
        >
          <div className="absolute inset-0 opacity-70">
            <div className="absolute left-10 top-10 h-40 w-40 rounded-full bg-white/10 blur-3xl" />
            <div className="absolute bottom-10 right-10 h-48 w-48 rounded-full bg-[rgba(56,199,186,0.18)] blur-3xl" />
          </div>
          <div className="relative">
            <div className="eyebrow text-white/72">
              <Sparkles className="h-3.5 w-3.5" />
              {t('hero.eyebrow')}
            </div>
            <h1 className="mt-5 max-w-lg text-5xl font-semibold tracking-[-0.05em] text-white">
              {t('hero.title')}
            </h1>
            <p className="mt-5 max-w-xl text-base leading-7 text-white/78">
              {t('hero.description')}
            </p>

            <div className="mt-10 grid gap-4 md:grid-cols-3">
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">{t('hero.step1Label')}</div>
                <div className="mt-2 text-lg font-semibold text-white">{t('hero.step1Value')}</div>
              </div>
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">{t('hero.step2Label')}</div>
                <div className="mt-2 text-lg font-semibold text-white">{t('hero.step2Value')}</div>
              </div>
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">{t('hero.step3Label')}</div>
                <div className="mt-2 text-lg font-semibold text-white">{t('hero.step3Value')}</div>
              </div>
            </div>
          </div>
        </Card>

        <Card className="mx-auto w-full max-w-lg rounded-[32px] p-6 md:p-8">
          <div className="flex flex-col items-center text-center">
            <div className="animate-pulse-glow flex h-16 w-16 items-center justify-center rounded-[24px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-lg)]">
              <Shield className="h-8 w-8" />
            </div>
            <div className="eyebrow mt-5">{t('card.eyebrow')}</div>
            <h1 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[var(--text-primary)]">
              {t('card.title')}
            </h1>
            <p className="mt-2 max-w-sm text-sm leading-6 text-[var(--text-muted)]">
              {t('card.description')}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="mt-8 space-y-5">
            {error && (
              <div
                className="rounded-[20px] border border-[rgba(208,71,75,0.16)] bg-[var(--error-light)] px-4 py-3 text-sm font-medium text-[var(--error)]"
                role="alert"
              >
                {error}
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
              autoComplete="new-password"
              placeholder={t('fields.passwordPlaceholder')}
              leftIcon={<Lock className="h-4 w-4" />}
              required
            />

            <Input
              label={t('fields.confirmPassword')}
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              autoComplete="new-password"
              placeholder={t('fields.confirmPasswordPlaceholder')}
              leftIcon={<CheckCircle className="h-4 w-4" />}
              required
            />

            <Button type="submit" className="w-full" size="lg" loading={loading}>
              {t('card.submit')}
            </Button>
          </form>
        </Card>
      </div>
    </div>
  )
}
