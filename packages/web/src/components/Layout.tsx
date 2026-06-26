import React from 'react'
import {
  FileKey,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  Monitor,
  Moon,
  Network,
  Route as RouteIcon,
  ScrollText,
  Settings,
  Shield,
  Sun,
  User as UserIcon,
  Users,
  X,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from './LanguageSwitcher'
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
      can_manage_hosts: boolean
    }
    features?: {
      certificates: boolean
    }
  }
  onLogout?: () => void
}

function getSystemTheme(): 'dark' | 'light' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function Layout({ children, currentPath, user, onLogout }: LayoutProps) {
  const { t, i18n } = useTranslation(['layout', 'users'])
  const isZh = i18n.resolvedLanguage === 'zh-CN'
  const [sidebarOpen, setSidebarOpen] = React.useState(false)
  const [themeMode, setThemeMode] = React.useState<'dark' | 'light' | 'system'>(() => {
    const stored = localStorage.getItem('theme')
    if (stored === 'light' || stored === 'dark') return stored
    return 'system'
  })
  const [systemTheme, setSystemTheme] = React.useState<'dark' | 'light'>(getSystemTheme)
  const theme = themeMode === 'system' ? systemTheme : themeMode
  const previousActiveElement = React.useRef<HTMLElement | null>(null)
  const sidebarRef = React.useRef<HTMLElement | null>(null)
  const closeSidebarButtonRef = React.useRef<HTMLButtonElement | null>(null)

  const allNavItems = React.useMemo(() => {
    return [
      {
        path: '/',
        icon: LayoutDashboard,
        label: t('sections.dashboard.label'),
        description: t('sections.dashboard.description'),
        visible: true,
      },
      {
        path: '/routes',
        icon: RouteIcon,
        label: t('sections.routes.label'),
        description: t('sections.routes.description'),
        visible: true,
      },
      {
        path: '/logs',
        icon: ScrollText,
        label: t('sections.logs.label'),
        description: t('sections.logs.description'),
        visible: user?.permissions?.can_view_logs === true,
      },
      {
        path: '/certificates',
        icon: FileKey,
        label: t('sections.certificates.label'),
        description: t('sections.certificates.description'),
        visible: user?.features?.certificates === true,
      },
      {
        path: '/hosts',
        icon: Network,
        label: t('sections.hosts.label'),
        description: t('sections.hosts.description'),
        visible: user?.permissions?.can_manage_hosts === true,
      },
      {
        path: '/auth',
        icon: Shield,
        label: '鉴权配置',
        description: '配置路由级认证方式和 API Key',
        visible: true,
      },
      {
        path: '/users',
        icon: Users,
        label: t('sections.users.label'),
        description: t('sections.users.description'),
        visible: user?.permissions?.can_manage_users === true,
      },
      {
        path: '/settings',
        icon: Settings,
        label: t('sections.settings.label'),
        description: t('sections.settings.description'),
        visible: true,
      },
    ]
  }, [t, user])
  const navItems = React.useMemo(
    () => allNavItems.filter((item) => item.visible),
    [allNavItems]
  )

  const activeItem =
    allNavItems.find((item) => item.path === currentPath) ??
    navItems[0] ??
    allNavItems[0]
  const closeSidebar = () => setSidebarOpen(false)
  const getSidebarFocusableElements = React.useCallback(() => {
    if (!sidebarRef.current) {
      return [] as HTMLElement[]
    }

    return Array.from(
      sidebarRef.current.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      )
    )
  }, [])

  React.useEffect(() => {
    if (sidebarOpen) {
      previousActiveElement.current = document.activeElement as HTMLElement
      document.body.style.overflow = 'hidden'
      closeSidebarButtonRef.current?.focus()
      return
    }

    document.body.style.overflow = ''
    previousActiveElement.current?.focus()
  }, [sidebarOpen])

  React.useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (!sidebarOpen) {
        return
      }

      if (event.key === 'Escape') {
        closeSidebar()
        return
      }

      if (event.key !== 'Tab') {
        return
      }

      const focusableElements = getSidebarFocusableElements()
      if (focusableElements.length === 0) {
        return
      }

      const [firstFocusableElement] = focusableElements
      const lastFocusableElement = focusableElements[focusableElements.length - 1]
      const activeElement = document.activeElement as HTMLElement | null

      if (!activeElement || !sidebarRef.current?.contains(activeElement)) {
        event.preventDefault()
        firstFocusableElement.focus()
        return
      }

      if (event.shiftKey && activeElement === firstFocusableElement) {
        event.preventDefault()
        lastFocusableElement.focus()
        return
      }

      if (!event.shiftKey && activeElement === lastFocusableElement) {
        event.preventDefault()
        firstFocusableElement.focus()
      }
    }

    document.addEventListener('keydown', handleEscape)
    return () => {
      document.body.style.overflow = ''
      document.removeEventListener('keydown', handleEscape)
    }
  }, [getSidebarFocusableElements, sidebarOpen])

  React.useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', themeMode)
  }, [theme, themeMode])

  // Listen for system theme changes when in 'system' mode
  React.useEffect(() => {
    if (themeMode !== 'system') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = (e: MediaQueryListEvent) => setSystemTheme(e.matches ? 'dark' : 'light')
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [themeMode])

  const roleLabel = (role: string) => {
    switch (role) {
      case 'member':
        return t('users:roles.member')
      case 'viewer':
        return t('users:roles.viewer')
      case 'editor':
        return t('users:roles.editor')
      case 'admin':
        return t('users:roles.admin')
      default:
        return role
    }
  }

  const sidebarContent = (
    <>
      {/* Sidebar Header */}
      <div className="border-b border-[var(--border-default)] px-5 py-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-[12px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] text-white">
            <Shield className="h-5 w-5" aria-hidden="true" />
          </div>
          <div>
            <div className="text-sm font-semibold text-[var(--text-primary)]">
              Auth Gate
            </div>
            <div className="mt-0.5 text-[10px] font-medium uppercase tracking-wider text-[var(--text-muted)]">
              API GATEWAY
            </div>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 px-3 py-4" role="navigation" aria-label={t('navigation.main')}>
        {navItems.map((item) => {
          const isActive = currentPath === item.path
          return (
            <a
              key={item.path}
              href={`#${item.path}`}
              onClick={closeSidebar}
              aria-current={isActive ? 'page' : undefined}
              className={cn(
                'flex items-center gap-3 rounded-[10px] px-3 py-2.5 transition-all duration-[var(--duration-normal)]',
                isActive
                  ? 'bg-[var(--bg-soft-primary)] text-[var(--primary-600)]'
                  : 'text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]'
              )}
            >
              <item.icon className={cn('h-5 w-5', isActive ? 'text-[var(--primary-600)]' : '')} aria-hidden="true" />
              <span className="text-sm font-medium">{item.label}</span>
            </a>
          )
        })}
      </nav>

      {/* Language Switcher */}
      <div className="px-4 pb-2">
        <LanguageSwitcher className="w-full" />
      </div>

      {/* Theme Toggle */}
      <div className="px-4 pb-3">
        <button
          type="button"
          onClick={() => setThemeMode(m => m === 'system' ? 'dark' : m === 'dark' ? 'light' : 'system')}
          className="flex w-full items-center gap-3 rounded-[10px] px-3 py-2.5 text-sm font-medium text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
        >
          {themeMode === 'system' ? (
            <>
              <Monitor className="h-4 w-4" />
              {isZh ? '跟随系统' : 'System'}
            </>
          ) : themeMode === 'dark' ? (
            <>
              <Moon className="h-4 w-4" />
              {isZh ? '深色模式' : 'Dark Mode'}
            </>
          ) : (
            <>
              <Sun className="h-4 w-4" />
              {isZh ? '浅色模式' : 'Light Mode'}
            </>
          )}
        </button>
      </div>

      {/* User Profile */}
      {user && (
        <div className="border-t border-[var(--border-default)] p-4 space-y-3">
          <a
            href="#/settings"
            onClick={closeSidebar}
            className="flex items-center gap-2 rounded-[10px] bg-[var(--bg-soft-primary)] px-3 py-2.5 text-sm font-medium text-[var(--primary-600)] transition-colors hover:bg-[rgba(15,143,139,0.15)]"
          >
            <Settings className="h-4 w-4" />
            Admin Panel
          </a>
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-[10px] bg-[var(--bg-soft-primary)] text-[var(--primary-600)]">
              <UserIcon className="h-4 w-4" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-[var(--text-primary)]">{user.username}</p>
              <p className="text-[11px] uppercase tracking-wider text-[var(--text-muted)]">Admin Profile</p>
            </div>
            <button
              type="button"
              onClick={onLogout}
              className="flex h-8 w-8 items-center justify-center rounded-[8px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
              aria-label={t('user.logout')}
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </>
  )

  return (
    <div className="min-h-screen bg-transparent">
      {sidebarOpen && (
        <div className="fixed inset-0 z-40 bg-[rgba(0,0,0,0.5)] backdrop-blur-sm md:hidden" onClick={closeSidebar} aria-hidden="true" />
      )}

      {/* Desktop Sidebar */}
      <aside
        className={cn(
          'fixed left-0 top-0 hidden h-screen w-[var(--sidebar-width)] flex-col overflow-y-auto border-r border-[var(--border-default)] bg-[var(--bg-page-strong)] md:flex',
          'z-50'
        )}
      >
        {sidebarContent}
      </aside>

      {/* Mobile Sidebar */}
      {sidebarOpen && (
        <aside
          ref={sidebarRef}
          className="fixed left-0 top-0 z-50 flex h-screen w-[var(--sidebar-width)] flex-col overflow-y-auto border-r border-[var(--border-default)] bg-[var(--bg-page-strong)] transition-transform duration-[var(--duration-slow)] md:hidden"
          role="dialog"
          aria-modal="true"
          aria-label={t('navigation.main')}
        >
          <div className="absolute right-3 top-3 z-10">
            <button
              type="button"
              ref={closeSidebarButtonRef}
              onClick={closeSidebar}
              className="flex h-9 w-9 items-center justify-center rounded-[8px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]"
              aria-label={t('navigation.closeMenu')}
            >
              <X className="h-5 w-5" />
            </button>
          </div>
          {sidebarContent}
        </aside>
      )}

      <main className="min-h-screen md:ml-[var(--sidebar-width)]">
        {/* Mobile Header */}
        <header className="sticky top-0 z-30 border-b border-[var(--border-default)] bg-[var(--bg-page)] backdrop-blur-xl md:hidden">
          <div className="flex h-[var(--header-height)] items-center justify-between px-4">
            <button
              type="button"
              onClick={() => setSidebarOpen(true)}
              className="flex h-9 w-9 items-center justify-center rounded-[8px] bg-[var(--bg-card)] text-[var(--text-primary)] shadow-[var(--shadow-sm)]"
              aria-label={t('navigation.openMenu')}
            >
              <Menu className="h-5 w-5" />
            </button>
            <div className="text-center">
              <div className="text-sm font-semibold text-[var(--text-primary)]">{activeItem?.label}</div>
            </div>
            <div className="flex h-9 w-9 items-center justify-center rounded-[8px] bg-[linear-gradient(135deg,var(--primary-500),var(--primary-600))] text-white">
              <Shield className="h-4 w-4" />
            </div>
          </div>
        </header>

        <div className="flex min-h-screen w-full flex-col px-4 pb-[calc(var(--bottom-nav-height)+1rem)] pt-5 md:px-8 md:pb-8 md:pt-6">
          {children}
        </div>

        {/* Mobile Bottom Nav */}
        <nav
          className="fixed bottom-3 left-3 right-3 z-30 flex rounded-[16px] border border-[var(--border-default)] bg-[var(--bg-card-strong)] px-2 py-2 backdrop-blur-xl md:hidden"
          role="navigation"
          aria-label={t('navigation.mobile')}
        >
          {navItems.map((item) => {
            const isActive = currentPath === item.path
            return (
              <a
                key={item.path}
                href={`#${item.path}`}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex flex-1 flex-col items-center justify-center gap-1 rounded-[10px] px-2 py-2 text-[10px] font-medium transition-all',
                  isActive
                    ? 'bg-[var(--bg-soft-primary)] text-[var(--primary-600)]'
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
