import React from 'react'
import { cn } from '../lib/utils'
import { Shield, Route as RouteIcon, Settings, KeyRound, Menu, X, LogOut, User as UserIcon } from 'lucide-react'

interface LayoutProps {
  children: React.ReactNode
  currentPath: string
  user?: { username: string; role: string }
  onLogout?: () => void
}

const navItems = [
  { path: '/', icon: RouteIcon, label: 'Routes' },
  { path: '/auth', icon: KeyRound, label: 'Auth Rules' },
  { path: '/settings', icon: Settings, label: 'Settings' },
]

export function Layout({ children, currentPath, user, onLogout }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = React.useState(false)
  const [userMenuOpen, setUserMenuOpen] = React.useState(false)

  const closeSidebar = () => setSidebarOpen(false)

  return (
    <div className="flex min-h-screen bg-[var(--bg-page)]">
      {sidebarOpen && (
        <div className="fixed inset-0 bg-black/50 z-40 md:hidden" onClick={closeSidebar} aria-hidden="true" />
      )}

      {/* Sidebar - Desktop */}
      <aside className={cn(
        'fixed left-0 top-0 h-screen bg-[var(--bg-card)] border-r border-[var(--border-default)]',
        'hidden md:flex flex-col z-50 w-[var(--sidebar-width)]'
      )}>
        <div className="h-[var(--header-height)] px-4 flex items-center gap-2 border-b border-[var(--border-default)]">
          <Shield className="w-8 h-8 text-[var(--primary-500)]" aria-hidden="true" />
          <span className="font-semibold text-[var(--text-primary)]">Auth Gate</span>
        </div>

        <nav className="flex-1 p-3 space-y-1" role="navigation" aria-label="Main navigation">
          {navItems.map((item) => {
            const isActive = currentPath === item.path
            return (
              <a key={item.path} href={'#' + item.path}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-md)]',
                  'text-[var(--text-sm)] font-medium transition-colors',
                  isActive ? 'bg-[var(--primary-500)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'
                )}>
                <item.icon className="w-5 h-5" aria-hidden="true" />
                <span>{item.label}</span>
              </a>
            )
          })}
        </nav>

        {user && (
          <div className="p-4 border-t border-[var(--border-default)]">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-[var(--primary-100)] flex items-center justify-center">
                <UserIcon className="w-4 h-4 text-[var(--primary-500)]" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{user.username}</p>
                <p className="text-xs text-[var(--text-muted)]">{user.role}</p>
              </div>
              <button onClick={onLogout} className="p-2 rounded hover:bg-[var(--bg-hover)] text-[var(--text-muted)]" aria-label="Logout">
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}
      </aside>

      {/* Sidebar - Mobile */}
      <aside className={cn(
        'fixed left-0 top-0 h-screen w-[280px] bg-[var(--bg-card)] border-r border-[var(--border-default)]',
        'flex flex-col z-50 transition-transform duration-[var(--duration-slow)] md:hidden',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      )} role="dialog" aria-modal="true" aria-label="Navigation menu">
        <div className="h-[var(--header-height)] px-4 flex items-center justify-between border-b border-[var(--border-default)]">
          <div className="flex items-center gap-2">
            <Shield className="w-8 h-8 text-[var(--primary-500)]" />
            <span className="font-semibold">Auth Gate</span>
          </div>
          <button onClick={closeSidebar} className="p-2 rounded hover:bg-[var(--bg-hover)]" aria-label="Close menu">
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className="flex-1 p-3 space-y-1">
          {navItems.map((item) => {
            const isActive = currentPath === item.path
            return (
              <a key={item.path} href={'#' + item.path} onClick={closeSidebar}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex items-center gap-3 px-3 py-3 rounded-[var(--radius-md)] text-[var(--text-sm)] font-medium transition-colors',
                  isActive ? 'bg-[var(--primary-500)] text-white' : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-hover)]'
                )}>
                <item.icon className="w-5 h-5" />
                <span>{item.label}</span>
              </a>
            )
          })}
        </nav>

        {user && (
          <div className="p-4 border-t border-[var(--border-default)]">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-full bg-[var(--primary-100)] flex items-center justify-center">
                <UserIcon className="w-4 h-4 text-[var(--primary-500)]" />
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium">{user.username}</p>
                <p className="text-xs text-[var(--text-muted)]">{user.role}</p>
              </div>
              <button onClick={onLogout} className="p-2 rounded hover:bg-[var(--bg-hover)]" aria-label="Logout">
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}
      </aside>

      {/* Main content */}
      <main className={cn('flex-1 flex flex-col min-h-screen md:ml-[var(--sidebar-width)]')}>
        <header className="h-[var(--header-height)] px-4 flex items-center justify-between border-b border-[var(--border-default)] bg-[var(--bg-card)] md:hidden">
          <button onClick={() => setSidebarOpen(true)} className="p-2 rounded hover:bg-[var(--bg-hover)]" aria-label="Open menu">
            <Menu className="w-6 h-6" />
          </button>
          <Shield className="w-7 h-7 text-[var(--primary-500)]" />
          <div className="w-10" />
        </header>

        <div className="flex-1 p-4 md:p-8 pb-[calc(var(--bottom-nav-height)+1rem)] md:pb-8">
          {children}
        </div>

        <nav className="fixed bottom-0 left-0 right-0 h-[var(--bottom-nav-height)] bg-[var(--bg-card)] border-t border-[var(--border-default)] flex md:hidden z-30" role="navigation" aria-label="Mobile navigation">
          {navItems.map((item) => {
            const isActive = currentPath === item.path
            return (
              <a key={item.path} href={'#' + item.path}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex-1 flex flex-col items-center justify-center gap-1 text-[var(--text-xs)] font-medium transition-colors',
                  isActive ? 'text-[var(--primary-500)]' : 'text-[var(--text-muted)]'
                )}>
                <item.icon className="w-5 h-5" />
                <span>{item.label}</span>
              </a>
            )
          })}
        </nav>
      </main>
    </div>
  )
}
