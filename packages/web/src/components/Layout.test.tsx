import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Layout } from './Layout'
import { renderWithI18n } from '../test/render'

const adminUser = {
  username: 'admin',
  role: 'admin',
  permissions: {
    can_manage_routes: true,
    can_manage_auth: true,
    can_manage_users: true,
    can_view_logs: true, can_manage_hosts: true,
  },
  features: {
    certificates: true,
  },
}

const viewerUser = {
  username: 'viewer-debug',
  role: 'viewer',
  permissions: {
    can_manage_routes: false,
    can_manage_auth: false,
    can_manage_users: false,
    can_view_logs: false, can_manage_hosts: false,
  },
  features: {
    certificates: false,
  },
}

describe('Layout language switching', () => {
  it('switches navigation labels when the language toggle changes', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <Layout
        currentPath="/"
        user={adminUser}
        onLogout={vi.fn()}
      >
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    expect(screen.getAllByText('Routes')[0]).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Language switcher' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: '中文' }))

    expect(screen.getAllByText('路由')[0]).toBeInTheDocument()
    expect(screen.getByText('管理员')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: '语言切换' })).toBeInTheDocument()
    expect(localStorage.getItem('auth-gate.locale')).toBe('zh-CN')
  })

  it('closes the mobile navigation drawer when Escape is pressed', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Open menu' }))

    expect(screen.getByRole('dialog', { name: 'Main navigation' })).toBeInTheDocument()

    await user.keyboard('{Escape}')

    expect(screen.queryByRole('dialog', { name: 'Main navigation' })).not.toBeInTheDocument()
  })

  it('moves focus into the mobile navigation drawer and locks body scroll while it is open', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    const openMenuButton = screen.getByRole('button', { name: 'Open menu' })
    await user.click(openMenuButton)

    const closeMenuButton = screen.getByRole('button', { name: 'Close menu' })
    expect(closeMenuButton).toHaveFocus()
    expect(document.body.style.overflow).toBe('hidden')

    await user.click(closeMenuButton)

    expect(openMenuButton).toHaveFocus()
    expect(document.body.style.overflow).toBe('')
  })

  it('keeps focus trapped inside the mobile navigation drawer when tabbing past the last control', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Open menu' }))

    const dialog = screen.getByRole('dialog', { name: 'Main navigation' })
    const closeMenuButton = within(dialog).getByRole('button', { name: 'Close menu' })
    const logoutButton = within(dialog).getByRole('button', { name: 'Logout' })

    logoutButton.focus()
    await user.tab()

    expect(closeMenuButton).toHaveFocus()
  })

  it('uses 44px touch targets for mobile icon buttons', async () => {
    await renderWithI18n(
      <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    const openMenuButton = screen.getByRole('button', { name: 'Open menu' })

    expect(openMenuButton).toHaveClass('h-11')
    expect(openMenuButton).toHaveClass('w-11')

    await userEvent.setup().click(openMenuButton)

    const dialog = screen.getByRole('dialog', { name: 'Main navigation' })
    const closeMenuButton = within(dialog).getByRole('button', { name: 'Close menu' })
    const logoutButton = within(dialog).getByRole('button', { name: 'Logout' })

    expect(closeMenuButton).toHaveClass('h-11')
    expect(closeMenuButton).toHaveClass('w-11')
    expect(logoutButton).toHaveClass('h-11')
    expect(logoutButton).toHaveClass('w-11')
  })

  it('does not submit a parent form when the open menu button is clicked', async () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
          <div>content</div>
        </Layout>
      </form>,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Open menu' }))

    expect(screen.getByRole('dialog', { name: 'Main navigation' })).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('does not submit a parent form when the logout button is clicked', async () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const onLogout = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <Layout currentPath="/" user={adminUser} onLogout={onLogout}>
          <div>content</div>
        </Layout>
      </form>,
      { locale: 'en' }
    )

    const logoutButtons = screen.getAllByRole('button', { name: 'Logout' })
    await user.click(logoutButtons[0])

    expect(onLogout).toHaveBeenCalledTimes(1)
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('does not submit a parent form when the close menu button is clicked', async () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <Layout currentPath="/" user={adminUser} onLogout={vi.fn()}>
          <div>content</div>
        </Layout>
      </form>,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Open menu' }))
    await user.click(screen.getByRole('button', { name: 'Close menu' }))

    expect(screen.queryByRole('dialog', { name: 'Main navigation' })).not.toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('hides the certificates navigation item when certificate automation is unavailable', async () => {
    await renderWithI18n(
      <Layout
        currentPath="/"
        user={{
          ...adminUser,
          features: {
            certificates: false,
          },
        }}
        onLogout={vi.fn()}
      >
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    expect(screen.queryByRole('link', { name: /certificates/i })).not.toBeInTheDocument()
  })

  it('still labels the current section as certificates when the certificates nav item is hidden', async () => {
    await renderWithI18n(
      <Layout
        currentPath="/certificates"
        user={{
          ...adminUser,
          features: {
            certificates: false,
          },
        }}
        onLogout={vi.fn()}
      >
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    expect(screen.getAllByText('Certificates')).toHaveLength(2)
    expect(screen.queryByRole('link', { name: /certificates/i })).not.toBeInTheDocument()
  })

  it('shows readable routes and auth navigation items for viewer accounts', async () => {
    await renderWithI18n(
      <Layout currentPath="/" user={viewerUser} onLogout={vi.fn()}>
        <div>content</div>
      </Layout>,
      { locale: 'en' }
    )

    const desktopNav = screen.getByRole('navigation', { name: 'Main navigation' })
    const mobileNav = screen.getByRole('navigation', { name: 'Mobile navigation' })

    expect(within(desktopNav).getByRole('link', { name: /routes/i })).toBeInTheDocument()
    expect(within(desktopNav).getByRole('link', { name: /auth rules/i })).toBeInTheDocument()
    expect(within(desktopNav).queryByRole('link', { name: /users/i })).not.toBeInTheDocument()

    expect(within(mobileNav).getByRole('link', { name: /routes/i })).toBeInTheDocument()
    expect(within(mobileNav).getByRole('link', { name: /auth rules/i })).toBeInTheDocument()
    expect(within(mobileNav).queryByRole('link', { name: /users/i })).not.toBeInTheDocument()
  })
})
