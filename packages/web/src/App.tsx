import React from 'react'
import { LoginPage } from './pages/LoginPage'
import { AccessLoginPage } from './pages/AccessLoginPage'
import { Layout } from './components/Layout'
import { RoutesPage } from './pages/RoutesPage'
import { AuthRulesPage } from './pages/AuthRulesPage'
import { SettingsPage } from './pages/SettingsPage'
import { UsersPage } from './pages/UsersPage'
import { CertificatesPage } from './pages/CertificatesPage'
import { HostsPage } from './pages/HostsPage'
import { useSession } from './lib/session'

const knownControlPlanePaths = new Set([
  '/',
  '/access-login',
  '/auth',
  '/certificates',
  '/settings',
  '/users',
  '/hosts',
])

function useRoute() {
  const [path, setPath] = React.useState(window.location.hash.slice(1) || '/')

  React.useEffect(() => {
    const handler = () => setPath(window.location.hash.slice(1) || '/')
    window.addEventListener('hashchange', handler)
    return () => window.removeEventListener('hashchange', handler)
  }, [])

  return path
}

export default function App() {
  const { user, token, loading, bootstrapping, notice, login, logout, clearNotice } = useSession()
  const path = useRoute()
  const [pathname, search = ''] = path.split('?')
  const searchParams = React.useMemo(() => new URLSearchParams(search), [search])
  const normalizedPathname = knownControlPlanePaths.has(pathname) ? pathname : '/'
  const waitingForUsersPermissionBootstrap =
    normalizedPathname === '/users' && bootstrapping && token && user
  const effectivePathname =
    !waitingForUsersPermissionBootstrap &&
    normalizedPathname === '/users' &&
    token &&
    user &&
    user.permissions?.can_manage_users !== true
      ? '/'
      : normalizedPathname

  React.useEffect(() => {
    if (effectivePathname !== pathname) {
      window.location.hash = effectivePathname
    }
  }, [effectivePathname, pathname])

  if (pathname === '/access-login') {
    return <AccessLoginPage searchParams={searchParams} />
  }

  if (loading || waitingForUsersPermissionBootstrap) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--bg-page)]">
        <div className="animate-spin w-8 h-8 border-2 border-[var(--primary-500)] border-t-transparent rounded-full" />
      </div>
    )
  }

  if (!token || !user) {
    return <LoginPage onLogin={login} sessionNotice={notice} onSessionNoticeShown={clearNotice} />
  }

  const renderPage = () => {
    switch (effectivePathname) {
      case '/auth': return <AuthRulesPage />
      case '/users': return <UsersPage />
      case '/certificates': return <CertificatesPage />
      case '/settings': return <SettingsPage />
      case '/hosts': return <HostsPage />
      default: return <RoutesPage />
    }
  }

  return (
    <Layout currentPath={effectivePathname} user={user} onLogout={logout}>
      {renderPage()}
    </Layout>
  )
}
