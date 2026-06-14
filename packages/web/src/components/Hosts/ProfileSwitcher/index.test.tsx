import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ProfileSwitcher } from './index'
import { renderWithI18n } from '../../../test/render'

const profiles = [
  { id: 'a', name: 'Dev', description: '', is_active: true, created_at: '', updated_at: '' },
  { id: 'b', name: 'Prod', description: '', is_active: false, created_at: '', updated_at: '' },
]

describe('ProfileSwitcher', () => {
  it('marks the active profile and notifies the parent on click', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <ProfileSwitcher profiles={profiles} activeId="a" canManage onChange={onChange} />,
      { locale: 'en' }
    )

    expect(screen.getByRole('button', { name: 'Dev' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Prod' })).toHaveAttribute('aria-pressed', 'false')

    await user.click(screen.getByRole('button', { name: 'Prod' }))
    expect(onChange).toHaveBeenCalledWith('b')
  })

  it('hides admin actions when canManage is false', async () => {
    await renderWithI18n(
      <ProfileSwitcher profiles={profiles} activeId="a" canManage={false} onChange={vi.fn()} />,
      { locale: 'en' }
    )
    expect(screen.queryByRole('button', { name: /activate/i })).not.toBeInTheDocument()
  })
})
