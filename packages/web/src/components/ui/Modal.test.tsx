import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Modal } from './Modal'
import { renderWithI18n } from '../../test/render'

describe('Modal', () => {
  it('moves focus to the first control when the dialog opens', async () => {
    await renderWithI18n(
      <div>
        <button type="button">Outside</button>
        <Modal open onClose={vi.fn()} title="Route editor">
          <label>
            Name
            <input type="text" />
          </label>
          <button type="button">Save</button>
        </Modal>
      </div>,
      { locale: 'en' }
    )

    const dialog = screen.getByRole('dialog', { name: 'Route editor' })
    const closeButton = within(dialog).getByRole('button', { name: 'Close modal' })

    expect(closeButton).toHaveFocus()
  })

  it('uses a 44px touch target for the close button on mobile', async () => {
    await renderWithI18n(
      <Modal open onClose={vi.fn()} title="Route editor">
        <button type="button">Save</button>
      </Modal>,
      { locale: 'en' }
    )

    const closeButton = screen.getByRole('button', { name: 'Close modal' })

    expect(closeButton).toHaveClass('h-9')
    expect(closeButton).toHaveClass('w-9')
  })

  it('keeps focus trapped inside the dialog when tabbing past the last control', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <div>
        <button type="button">Outside</button>
        <Modal open onClose={vi.fn()} title="Route editor">
          <label>
            Name
            <input type="text" />
          </label>
          <button type="button">Save</button>
        </Modal>
      </div>,
      { locale: 'en' }
    )

    const dialog = screen.getByRole('dialog', { name: 'Route editor' })
    const closeButton = within(dialog).getByRole('button', { name: 'Close modal' })
    const saveButton = within(dialog).getByRole('button', { name: 'Save' })

    saveButton.focus()
    await user.tab()

    expect(closeButton).toHaveFocus()
  })

  it('does not submit a parent form when the close button is clicked', async () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const onClose = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <Modal open onClose={onClose} title="Route editor">
          <button type="button">Save</button>
        </Modal>
      </form>,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Close modal' }))

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
