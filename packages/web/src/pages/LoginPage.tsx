import React from 'react'
import { Lock, Shield, Sparkles, User } from 'lucide-react'
import type { LoginResponse } from '../lib/api/types'
import { ApiError } from '../lib/api/client'
import { Button, Card, Input } from '../components/ui'

interface LoginPageProps {
  onLogin: (username: string, password: string) => Promise<LoginResponse>
}

export function LoginPage({ onLogin }: LoginPageProps) {
  const [username, setUsername] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [error, setError] = React.useState('')
  const [loading, setLoading] = React.useState(false)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    try {
      await onLogin(username, password)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Network error')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-10">
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
              Secure Routing Workspace
            </div>
            <h1 className="mt-5 max-w-lg text-5xl font-semibold tracking-[-0.05em] text-white">
              Manage gateway traffic with a calmer, sharper control surface.
            </h1>
            <p className="mt-5 max-w-xl text-base leading-7 text-white/78">
              Configure routes, apply authentication policies, and perform runtime operations from one focused console.
            </p>

            <div className="mt-10 grid gap-4 md:grid-cols-3">
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">Routing</div>
                <div className="mt-2 text-lg font-semibold text-white">Precise matching</div>
              </div>
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">Auth</div>
                <div className="mt-2 text-lg font-semibold text-white">Policy per route</div>
              </div>
              <div className="rounded-[24px] border border-white/12 bg-white/6 p-4 backdrop-blur-md">
                <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-white/58">Runtime</div>
                <div className="mt-2 text-lg font-semibold text-white">Safe reload operations</div>
              </div>
            </div>
          </div>
        </Card>

        <Card className="mx-auto w-full max-w-lg rounded-[32px] p-6 md:p-8">
          <div className="flex flex-col items-center text-center">
            <div className="animate-pulse-glow flex h-16 w-16 items-center justify-center rounded-[24px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-lg)]">
              <Shield className="h-8 w-8" />
            </div>
            <div className="eyebrow mt-5">Auth Gate</div>
            <h1 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-[var(--text-primary)]">
              Sign in to the control plane
            </h1>
            <p className="mt-2 max-w-sm text-sm leading-6 text-[var(--text-muted)]">
              Use an administrator account to manage routes, authentication rules, and runtime operations.
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
              label="Username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="Enter username"
              leftIcon={<User className="h-4 w-4" />}
              required
            />

            <Input
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter password"
              leftIcon={<Lock className="h-4 w-4" />}
              required
            />

            <Button type="submit" className="w-full" size="lg" loading={loading}>
              Sign In
            </Button>
          </form>

          <div className="mt-6 rounded-[22px] border border-[var(--border-default)] bg-[rgba(255,255,255,0.52)] px-4 py-3 text-center text-xs text-[var(--text-muted)]">
            Session access is scoped by your server-side permissions.
          </div>
        </Card>
      </div>
    </div>
  )
}
