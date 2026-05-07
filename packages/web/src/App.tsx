import React from 'react'
import { LoginPage } from './pages/LoginPage'
import { AccessLoginPage } from './pages/AccessLoginPage'
import { Layout } from './components/Layout'
import { RoutesPage } from './pages/RoutesPage'
import { AuthRulesPage } from './pages/AuthRulesPage'
import { SettingsPage } from './pages/SettingsPage'
import { UsersPage } from './pages/UsersPage'
import { useSession } from './lib/session'

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
  const { user, token, loading, login, logout } = useSession()
  const path = useRoute()
  const [pathname, search = ''] = path.split('?')
  const searchParams = React.useMemo(() => new URLSearchParams(search), [search])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--bg-page)]">
        <div className="animate-spin w-8 h-8 border-2 border-[var(--primary-500)] border-t-transparent rounded-full" />
      </div>
    )
  }

  if (pathname === '/access-login') {
    return <AccessLoginPage searchParams={searchParams} />
  }

  if (!token || !user) {
    return <LoginPage onLogin={login} />
  }

  const renderPage = () => {
    switch (pathname) {
      case '/auth': return <AuthRulesPage />
      case '/users': return <UsersPage />
      case '/settings': return <SettingsPage />
      default: return <RoutesPage />
    }
  }

  return (
    <Layout currentPath={pathname} user={user} onLogout={logout}>
      {renderPage()}
    </Layout>
  )
}
