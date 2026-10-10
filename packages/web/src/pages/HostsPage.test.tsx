import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { HostsPage } from './HostsPage'
import { hostsApi } from '../lib/api/hosts'
import { ApiError } from '../lib/api/client'
import { renderWithI18n } from '../test/render'

const profiles = [
  { id: 'p1', name: 'Dev', description: 'local dev', is_active: true, created_at: '', updated_at: '' },
  { id: 'p2', name: 'Staging', description: '', is_active: false, created_at: '', updated_at: '' },
]

const entries = [
  { id: 'e1', profile_id: 'p1', position: 0, ip: '127.0.0.1', hostnames: 'api.local', comment: '', enabled: true, created_at: '', updated_at: '' },
]

vi.mock('../lib/api/hosts', () => ({
  hostsApi: {
    list: vi.fn(),
    get: vi.fn(),
    activate: vi.fn(),
    deactivate: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listEntries: vi.fn(),
    createEntry: vi.fn(),
    updateEntry: vi.fn(),
    deleteEntry: vi.fn(),
    reorderEntries: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => ({
    username: 'admin',
    role: 'admin',
    permissions: { can_manage_hosts: true, can_manage_routes: true, can_manage_auth: true, can_manage_users: true, can_view_logs: true },
  }),
}))

describe('HostsPage', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.mocked(hostsApi.list).mockResolvedValue({ profiles, active_id: 'p1' })
    vi.mocked(hostsApi.get).mockImplementation((id) => Promise.resolve(profiles.find((p) => p.id === id)!))
    vi.mocked(hostsApi.listEntries).mockResolvedValue(entries)
    vi.mocked(hostsApi.activate).mockReset()
    vi.mocked(hostsApi.deactivate).mockReset()
    vi.mocked(hostsApi.create).mockReset()
    vi.mocked(hostsApi.createEntry).mockReset()
    vi.mocked(hostsApi.delete).mockReset()
  })

  it('keeps the active status visible when browsing a different profile', async () => {
    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })
    expect(await screen.findByText('Active profile: Dev')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Staging' }))
    expect(screen.getByText('Active profile: Dev')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Activate this profile' })).toBeInTheDocument()
  })

  it('disables repeated activation until the write finishes and then switches the active marker', async () => {
    let finish!: (value: typeof profiles[number]) => void
    vi.mocked(hostsApi.activate).mockImplementation(() => new Promise((resolve) => { finish = resolve }))
    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })
    await user.click(await screen.findByRole('button', { name: 'Staging' }))
    await user.click(screen.getByRole('button', { name: 'Activate this profile' }))
    expect(screen.getByRole('button', { name: 'Activating…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Dev' })).toBeDisabled()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    await act(async () => finish({ ...profiles[1], is_active: true }))
    expect(screen.getByRole('status')).toHaveTextContent('Profile "Staging" activated')
    expect(screen.getByText('Active profile: Staging')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Staging' })).toHaveAccessibleDescription('Active')
    expect(screen.getByRole('button', { name: 'Dev' })).not.toHaveAttribute('aria-describedby')
    expect(screen.getByRole('button', { name: 'Deactivate' })).toBeEnabled()
  })

  it('shows no active profile even when a saved profile is selected', async () => {
    vi.mocked(hostsApi.list).mockResolvedValue({ profiles: profiles.map((p) => ({ ...p, is_active: false })), active_id: '' })
    await renderWithI18n(<HostsPage />, { locale: 'zh-CN' })
    expect(await screen.findByText('暂无激活方案')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '激活此配置方案' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: '取消激活' })).not.toBeInTheDocument()
  })

  it('shows success and clears the active status after deactivation', async () => {
    vi.mocked(hostsApi.deactivate).mockResolvedValue({ ...profiles[0], is_active: false })
    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })
    await user.click(await screen.findByRole('button', { name: 'Deactivate' }))
    expect(await screen.findByText('Profile "Dev" deactivated')).toBeInTheDocument()
    expect(screen.getByText('No active profile')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Activate this profile' })).toBeInTheDocument()
    expect(screen.getByText('api.local')).toBeInTheDocument()
  })

  it('keeps the active status and reports a failed deactivation', async () => {
    vi.mocked(hostsApi.deactivate).mockRejectedValue(new ApiError('Access denied', 500, 'host_permission_denied'))
    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })
    await user.click(await screen.findByRole('button', { name: 'Deactivate' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/Administrator/)
    expect(screen.getByText('Active profile: Dev')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Deactivate' })).toBeEnabled()
  })

  it('renders profiles in ProfileSwitcher and entries in table', async () => {
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    expect(await screen.findByRole('button', { name: 'Dev' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Staging' })).toBeInTheDocument()
    expect(screen.getByText('api.local')).toBeInTheDocument()
    expect(screen.getByText('127.0.0.1')).toBeInTheDocument()
  })

  it('opens Add Entry modal and submits to API', async () => {
    vi.mocked(hostsApi.createEntry).mockResolvedValue({
      id: 'e2', profile_id: 'p1', position: 1, ip: '10.0.0.1', hostnames: 'new.local', comment: '', enabled: true, created_at: '', updated_at: '',
    })
    vi.mocked(hostsApi.listEntries).mockResolvedValueOnce(entries)

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    await user.click(screen.getByRole('button', { name: 'Add Entry' }))

    const ipInput = screen.getByLabelText('IP address')
    await user.type(ipInput, '10.0.0.1')
    await user.type(screen.getByLabelText('Hostnames (space-separated)'), 'new.local')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    await waitFor(() => {
      expect(hostsApi.createEntry).toHaveBeenCalledWith('p1', {
        ip: '10.0.0.1',
        hostnames: ['new.local'],
        comment: '',
        enabled: true,
      })
    })
  })

  it('shows marker-missing banner when activate fails with host_marker_missing', async () => {
    vi.mocked(hostsApi.activate).mockRejectedValue(
      new ApiError('marker missing', 400, 'host_marker_missing')
    )

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    await user.click(screen.getByRole('button', { name: 'Reapply' }))

    expect(await screen.findByText(/AUTH-GATE MANAGED BLOCK/)).toBeInTheDocument()
  })

  it('shows success after reapplying the active profile', async () => {
    vi.mocked(hostsApi.activate).mockResolvedValue({ ...profiles[0], is_active: true })
    const updatedProfiles = profiles.map((p) => ({ ...p, is_active: p.id === 'p1' }))
    vi.mocked(hostsApi.list).mockResolvedValueOnce({ profiles, active_id: 'p1' })
    vi.mocked(hostsApi.list).mockResolvedValueOnce({ profiles: updatedProfiles, active_id: 'p1' })

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    await user.click(screen.getByRole('button', { name: 'Reapply' }))

    await waitFor(() => {
      expect(hostsApi.activate).toHaveBeenCalledWith('p1')
      expect(screen.getByRole('status')).toHaveTextContent('Profile "Dev" activated')
    })
  })

  it('opens Add Profile modal and submits to API', async () => {
    vi.mocked(hostsApi.create).mockResolvedValue({
      id: 'p3', name: 'New Profile', description: '', is_active: false, created_at: '', updated_at: '',
    })
    vi.mocked(hostsApi.listEntries).mockResolvedValueOnce(entries)

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    await user.click(screen.getByRole('button', { name: 'Add Profile' }))

    await user.type(screen.getByLabelText('Profile name'), 'New Profile')
    await user.click(screen.getByRole('button', { name: 'Save Profile' }))

    await waitFor(() => {
      expect(hostsApi.create).toHaveBeenCalledWith({ name: 'New Profile', description: '' })
    })
  })

  it('toggles entry enabled state via Switch', async () => {
    vi.mocked(hostsApi.updateEntry).mockResolvedValue({
      ...entries[0], enabled: false,
    })

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    const toggle = screen.getAllByRole('checkbox', { name: 'Enabled' })[0]
    await user.click(toggle)

    await waitFor(() => {
      expect(hostsApi.updateEntry).toHaveBeenCalledWith('p1', 'e1', {
        ip: '127.0.0.1',
        hostnames: ['api.local'],
        comment: '',
        enabled: false,
      })
    })
  })

  it('deletes an entry after confirm', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.mocked(hostsApi.deleteEntry).mockResolvedValue({ status: 'ok' })

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByRole('button', { name: 'Dev' })
    const deleteBtn = screen.getAllByRole('button', { name: 'Delete' })[0]
    await user.click(deleteBtn)

    await waitFor(() => {
      expect(hostsApi.deleteEntry).toHaveBeenCalledWith('p1', 'e1')
    })

    vi.restoreAllMocks()
  })

  it('switches profile when ProfileSwitcher button is clicked', async () => {
    const stagingEntries = [
      { id: 'e3', profile_id: 'p2', position: 0, ip: '192.168.1.1', hostnames: 'staging.local', comment: '', enabled: true, created_at: '', updated_at: '' },
    ]
    vi.mocked(hostsApi.listEntries).mockResolvedValue(entries)
    vi.mocked(hostsApi.listEntries).mockResolvedValueOnce(entries)

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByText('api.local')

    vi.mocked(hostsApi.listEntries).mockResolvedValue(stagingEntries)
    await user.click(screen.getByRole('button', { name: 'Staging' }))

    await waitFor(() => {
      expect(screen.getByText('staging.local')).toBeInTheDocument()
    })
  })

  it('opens edit entry modal and submits update', async () => {
    vi.mocked(hostsApi.updateEntry).mockResolvedValue({
      ...entries[0], ip: '10.0.0.1',
    })

    const user = userEvent.setup()
    await renderWithI18n(<HostsPage />, { locale: 'en' })

    await screen.findByText('api.local')
    const editButtons = screen.getAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])

    const ipInput = screen.getByLabelText('IP address')
    await user.clear(ipInput)
    await user.type(ipInput, '10.0.0.1')
    await user.click(screen.getByRole('button', { name: 'Save Entry' }))

    await waitFor(() => {
      expect(hostsApi.updateEntry).toHaveBeenCalled()
    })
  })
})
