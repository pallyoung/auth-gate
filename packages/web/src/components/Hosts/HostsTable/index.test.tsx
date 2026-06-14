import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { HostsTable } from './index'
import { renderWithI18n } from '../../../test/render'

const entries = [
  {
    id: 'e1',
    profile_id: 'p1',
    position: 0,
    ip: '127.0.0.1',
    hostnames: 'api.local db.local',
    comment: 'dev box',
    enabled: true,
    created_at: '',
    updated_at: '',
  },
  {
    id: 'e2',
    profile_id: 'p1',
    position: 1,
    ip: '::1',
    hostnames: 'v6.local',
    comment: '',
    enabled: false,
    created_at: '',
    updated_at: '',
  },
]

describe('HostsTable', () => {
  it('renders entries and dispatches edit / delete / toggle callbacks', async () => {
    const onEdit = vi.fn()
    const onDelete = vi.fn()
    const onToggle = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <HostsTable
        entries={entries}
        canManage
        onEdit={onEdit}
        onDelete={onDelete}
        onToggleEnabled={onToggle}
      />,
      { locale: 'en' }
    )

    expect(screen.getByText('api.local db.local')).toBeInTheDocument()
    expect(screen.getByText('127.0.0.1')).toBeInTheDocument()

    const editButtons = screen.getAllByRole('button', { name: 'Edit' })
    await user.click(editButtons[0])
    expect(onEdit).toHaveBeenCalledWith(entries[0])

    const deleteButtons = screen.getAllByRole('button', { name: 'Delete' })
    await user.click(deleteButtons[0])
    expect(onDelete).toHaveBeenCalledWith(entries[0])

    const toggles = screen.getAllByRole('checkbox', { name: 'Enabled' })
    await user.click(toggles[0])
    expect(onToggle).toHaveBeenCalledWith(entries[0], false)
  })

  it('renders the empty-state when there are no entries', async () => {
    await renderWithI18n(
      <HostsTable
        entries={[]}
        canManage
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onToggleEnabled={vi.fn()}
      />,
      { locale: 'en' }
    )
    expect(screen.getByText('No entries in this profile yet.')).toBeInTheDocument()
  })
})
