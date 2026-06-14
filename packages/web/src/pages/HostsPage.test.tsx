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
    vi.mocked(hostsApi.list).mockResolvedValue({ profiles, active_id: 'p1' })
    vi.mocked(hostsApi.get).mockImplementation((id) => Promise.resolve(profiles.find((p) => p.id === id)!))
    vi.mocked(hostsApi.listEntries).mockResolvedValue(entries)
    vi.mocked(hostsApi.activate).mockReset()
    vi.mocked(hostsApi.create).mockReset()
    vi.mocked(hostsApi.createEntry).mockReset()
    vi.mocked(hostsApi.delete).mockReset()
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
    await user.click(screen.getByRole('button', { name: 'Activate this profile' }))

    expect(await screen.findByText(/AUTH-GATE MANAGED BLOCK/)).toBeInTheDocument()
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
})
