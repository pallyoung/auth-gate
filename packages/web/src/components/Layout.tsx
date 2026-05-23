import React from 'react'
import {
  FileKey,
  KeyRound,
  LogOut,
  Menu,
  Route as RouteIcon,
  Settings,
  Shield,
  User as UserIcon,
  Users,
  X,
} from 'lucide-react'
import { cn } from '../lib/utils'

interface LayoutProps {
  children: React.ReactNode
  currentPath: string
  user?: {
    username: string
    role: string
    permissions?: {
      can_manage_routes: boolean
      can_manage_auth: boolean
      can_manage_users: boolean
      can_view_logs: boolean
    }
  }
  onLogout?: () => void
}

export function Layout({ children, currentPath, user, onLogout }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = React.useState(false)

  const navItems = React.useMemo(() => {
    const items = [
      {
        path: '/',
        icon: RouteIcon,
        label: 'Routes',
        description: 'Traffic orchestration',
        visible: user?.permissions?.can_manage_routes !== false,
      },
      {
        path: '/certificates',
        icon: FileKey,
        label: 'Certificates',
        description: 'TLS automation',
        visible: user?.permissions?.can_manage_routes !== false,
      },
      {
        path: '/auth',
        icon: KeyRound,
        label: 'Auth Rules',
        description: 'Policy enforcement',
        visible: user?.permissions?.can_manage_auth !== false,
      },
      {
        path: '/users',
        icon: Users,
        label: 'Users',
        description: 'Directory and access',
        visible: user?.permissions?.can_manage_users === true,
      },
      {
        path: '/settings',
        icon: Settings,
        label: 'Settings',
        description: 'Runtime operations',
        visible: true,
      },
    ]

    return items.filter((item) => item.visible)
  }, [user])

  const activeItem = navItems.find((item) => item.path === currentPath) ?? navItems[0]
  const closeSidebar = () => setSidebarOpen(false)

  const sidebarContent = (
    <>
      <div className="relative border-b border-[var(--border-soft)] px-5 py-5">
        <div className="flex items-center gap-4">
          <div className="animate-pulse-glow flex h-12 w-12 items-center justify-center rounded-[18px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-md)]">
            <Shield className="h-6 w-6" aria-hidden="true" />
          </div>
          <div>
            <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--text-muted)]">
              Auth Gate
            </div>
            <div className="mt-1 text-lg font-semibold tracking-[-0.03em] text-[var(--text-primary)]">
              Control Plane
            </div>
          </div>
        </div>
      </div>

      <div className="px-4 pt-5">
        <div className="rounded-[24px] border border-[var(--border-soft)] bg-[linear-gradient(135deg,rgba(15,143,139,0.12),rgba(255,255,255,0.08))] px-4 py-4">
          <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--text-muted)]">
            Current Focus
          </div>
          <div className="mt-2 text-lg font-semibold tracking-[-0.02em] text-[var(--text-primary)]">
            {activeItem?.label}
          </div>
          <p className="mt-1 text-sm text-[var(--text-muted)]">{activeItem?.description}</p>
        </div>
      </div>

      <nav className="flex-1 space-y-2 px-4 py-5" role="navigation" aria-label="Main navigation">
        {navItems.map((item) => {
          const isActive = currentPath === item.path
          return (
            <a
              key={item.path}
              href={`#${item.path}`}
              onClick={closeSidebar}
              aria-current={isActive ? 'page' : undefined}
              className={cn(
                'group flex items-center gap-3 rounded-[22px] px-4 py-3 transition-all duration-[var(--duration-normal)]',
                isActive
                  ? 'bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-md)]'
                  : 'text-[var(--text-secondary)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]'
              )}
            >
              <div
                className={cn(
                  'flex h-11 w-11 items-center justify-center rounded-[16px] transition-colors',
                  isActive ? 'bg-white/18 text-white' : 'bg-[rgba(255,255,255,0.56)] text-[var(--primary-600)]'
                )}
              >
                <item.icon className="h-5 w-5" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <div className="text-sm font-semibold">{item.label}</div>
                <div className={cn('text-xs', isActive ? 'text-white/78' : 'text-[var(--text-muted)]')}>
                  {item.description}
                </div>
              </div>
            </a>
          )
        })}
      </nav>

      {user && (
        <div className="border-t border-[var(--border-soft)] p-4">
          <div className="rounded-[24px] border border-[var(--border-soft)] bg-[rgba(255,255,255,0.18)] p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-[16px] bg-[var(--bg-soft-primary)] text-[var(--primary-600)]">
                <UserIcon className="h-5 w-5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-[var(--text-primary)]">{user.username}</p>
                <p className="text-xs uppercase tracking-[0.12em] text-[var(--text-muted)]">{user.role}</p>
              </div>
              <button
                onClick={onLogout}
                className="flex h-10 w-10 items-center justify-center rounded-full text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
                aria-label="Logout"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )

  return (
    <div className="app-shell-grid min-h-screen bg-transparent">
      {sidebarOpen && (
        <div className="fixed inset-0 z-40 bg-[rgba(15,23,34,0.42)] backdrop-blur-sm md:hidden" onClick={closeSidebar} aria-hidden="true" />
      )}

      <aside
        className={cn(
          'sidebar-sheen glass-panel-strong fixed left-5 top-5 hidden h-[calc(100vh-2.5rem)] flex-col overflow-hidden md:flex',
          'w-[var(--sidebar-width)] z-50'
        )}
      >
        {sidebarContent}
      </aside>

      <aside
        className={cn(
          'glass-panel-strong fixed left-3 top-3 z-50 flex h-[calc(100vh-1.5rem)] w-[var(--sidebar-width)] flex-col overflow-hidden transition-transform duration-[var(--duration-slow)] md:hidden',
          sidebarOpen ? 'translate-x-0' : '-translate-x-[120%]'
        )}
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
      >
        <div className="absolute right-4 top-4 z-10">
          <button
            onClick={closeSidebar}
            className="flex h-10 w-10 items-center justify-center rounded-full bg-[rgba(255,255,255,0.44)] text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
            aria-label="Close menu"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        {sidebarContent}
      </aside>

      <main className="min-h-screen md:ml-[calc(var(--sidebar-width)+2.25rem)]">
        <header className="sticky top-0 z-30 border-b border-[rgba(255,255,255,0.35)] bg-[rgba(246,243,236,0.72)] backdrop-blur-xl md:hidden">
          <div className="flex h-[var(--header-height)] items-center justify-between px-4">
            <button
              onClick={() => setSidebarOpen(true)}
              className="flex h-11 w-11 items-center justify-center rounded-full bg-[rgba(255,255,255,0.72)] text-[var(--text-primary)] shadow-[var(--shadow-sm)]"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
            <div className="text-center">
              <div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--text-muted)]">
                Auth Gate
              </div>
              <div className="text-sm font-semibold text-[var(--text-primary)]">{activeItem?.label}</div>
            </div>
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-sm)]">
              <Shield className="h-5 w-5" />
            </div>
          </div>
        </header>

        <div className="mx-auto flex min-h-screen w-full max-w-[calc(var(--page-max-width)+4rem)] flex-col px-4 pb-[calc(var(--bottom-nav-height)+1rem)] pt-5 md:px-8 md:pb-10 md:pt-5">
          {children}
        </div>

        <nav
          className="glass-panel fixed bottom-3 left-3 right-3 z-30 flex rounded-[28px] border border-[var(--border-soft)] px-2 py-2 md:hidden"
          role="navigation"
          aria-label="Mobile navigation"
        >
          {navItems.map((item) => {
            const isActive = currentPath === item.path
            return (
              <a
                key={item.path}
                href={`#${item.path}`}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex flex-1 flex-col items-center justify-center gap-1 rounded-[22px] px-2 py-2 text-[11px] font-semibold transition-all',
                  isActive
                    ? 'bg-[linear-gradient(135deg,var(--primary-500),var(--primary-700))] text-white shadow-[var(--shadow-sm)]'
                    : 'text-[var(--text-muted)]'
                )}
              >
                <item.icon className="h-4 w-4" />
                <span>{item.label}</span>
              </a>
            )
          })}
        </nav>
      </main>
    </div>
  )
}
