import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { DataTable } from './DataTable'
import { renderWithI18n } from '../test/render'

type TestRow = {
  id: string
  name: string
}

type NestedRow = {
  id: string
  profile: {
    email: string
  }
}

describe('DataTable i18n', () => {
  it('renders shared table labels in zh-CN', async () => {
    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[]}
        onEdit={vi.fn()}
      />,
      { locale: 'zh-CN' }
    )

    expect(screen.getByText('操作')).toBeInTheDocument()
    expect(screen.getAllByText('暂无数据')).toHaveLength(2)
  })

  it('uses 44px touch targets for row action buttons', async () => {
    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[{ id: 'user-1', name: 'admin' }]}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />,
      { locale: 'en' }
    )

    for (const button of screen.getAllByLabelText('Edit')) {
      expect(button).toHaveClass('h-9')
      expect(button).toHaveClass('w-9')
    }

    for (const button of screen.getAllByLabelText('Delete')) {
      expect(button).toHaveClass('h-9')
      expect(button).toHaveClass('w-9')
    }
  })

  it('prevents duplicate delete actions while a row deletion is pending', async () => {
    let resolveDelete: (() => void) | undefined
    const onDelete = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[{ id: 'user-1', name: 'admin' }]}
        onDelete={onDelete}
      />,
      { locale: 'en' }
    )

    const [deleteButton] = screen.getAllByLabelText('Delete')
    await user.click(deleteButton)
    await user.click(deleteButton)

    expect(onDelete).toHaveBeenCalledTimes(1)

    resolveDelete?.()
  })

  it('prevents back-to-back native delete clicks before the pending state re-renders', async () => {
    let resolveDelete: (() => void) | undefined
    const onDelete = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve
        })
    )

    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[{ id: 'user-1', name: 'admin' }]}
        onDelete={onDelete}
      />,
      { locale: 'en' }
    )

    const [deleteButton] = screen.getAllByLabelText('Delete')

    await act(async () => {
      deleteButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      deleteButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(onDelete).toHaveBeenCalledTimes(1)

    resolveDelete?.()
  })

  it('disables delete actions for the pending row while deletion is in progress', async () => {
    let resolveDelete: (() => void) | undefined
    const onDelete = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveDelete = resolve
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <DataTable
        columns={[{ key: 'name', header: 'Name' }]}
        data={[{ id: 'user-1', name: 'admin' }]}
        onDelete={onDelete}
      />,
      { locale: 'en' }
    )

    const [deleteButton] = screen.getAllByLabelText('Delete')
    await user.click(deleteButton)

    for (const actionButton of screen.getAllByLabelText('Delete')) {
      expect(actionButton).toBeDisabled()
    }

    resolveDelete?.()
  })

  it('keeps each row delete action disabled until its own deletion settles', async () => {
    const deleteResolvers = new Map<string, () => void>()
    const rows: TestRow[] = [
      { id: 'user-1', name: 'admin' },
      { id: 'user-2', name: 'editor' },
    ]
    const onDelete = vi.fn(
      (row: TestRow) =>
        new Promise<void>((resolve) => {
          deleteResolvers.set(row.id, resolve)
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <DataTable<TestRow>
        columns={[{ key: 'name', header: 'Name' }]}
        data={rows}
        onDelete={onDelete}
      />,
      { locale: 'en' }
    )

    const table = screen.getByRole('table')
    const adminRow = within(table).getByText('admin').closest('tr')
    const editorRow = within(table).getByText('editor').closest('tr')

    expect(adminRow).not.toBeNull()
    expect(editorRow).not.toBeNull()

    await user.click(within(adminRow as HTMLTableRowElement).getByLabelText('Delete'))
    await user.click(within(editorRow as HTMLTableRowElement).getByLabelText('Delete'))

    expect(onDelete).toHaveBeenCalledTimes(2)
    expect(within(adminRow as HTMLTableRowElement).getByLabelText('Delete')).toBeDisabled()
    expect(within(editorRow as HTMLTableRowElement).getByLabelText('Delete')).toBeDisabled()

    deleteResolvers.get('user-2')?.()

    await waitFor(() => {
      expect(within(editorRow as HTMLTableRowElement).getByLabelText('Delete')).not.toBeDisabled()
    })
    expect(within(adminRow as HTMLTableRowElement).getByLabelText('Delete')).toBeDisabled()

    deleteResolvers.get('user-1')?.()
  })

  it('renders nested field values when a column key uses dot notation', async () => {
    const rows: NestedRow[] = [
      {
        id: 'user-1',
        profile: {
          email: 'admin@example.com',
        },
      },
    ]

    await renderWithI18n(
      <DataTable<NestedRow>
        columns={[{ key: 'profile.email', header: 'Email' }]}
        data={rows}
      />,
      { locale: 'en' }
    )

    const table = screen.getByRole('table')

    expect(within(table).getByText('admin@example.com')).toBeInTheDocument()
    expect(screen.getByRole('list')).toHaveTextContent('admin@example.com')
  })

  it('does not submit a parent form when a row action button is clicked', async () => {
    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const onEdit = vi.fn()
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <DataTable
          columns={[{ key: 'name', header: 'Name' }]}
          data={[{ id: 'user-1', name: 'admin' }]}
          onEdit={onEdit}
        />
      </form>,
      { locale: 'en' }
    )

    const [editButton] = screen.getAllByLabelText('Edit')
    await user.click(editButton)

    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(onSubmit).not.toHaveBeenCalled()
  })
})
