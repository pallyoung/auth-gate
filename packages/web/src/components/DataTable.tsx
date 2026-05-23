import React from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  EmptyRow,
  MobileCardList,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from './ui'

interface Column<T> {
  key: keyof T | string
  header: string
  render?: (value: any, row: T) => React.ReactNode
  className?: string
  hideOnMobile?: boolean
}

interface DataTableProps<T> {
  columns: Column<T>[]
  data: T[]
  onEdit?: (row: T) => void
  onDelete?: (row: T) => void
  extraActions?: (row: T) => React.ReactNode
  renderMobileCard?: (row: T) => React.ReactNode
  emptyMessage?: string
}

function ActionButton({
  label,
  danger = false,
  onClick,
  children,
}: {
  label: string
  danger?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={[
        'flex h-9 w-9 items-center justify-center rounded-full border transition-all duration-[var(--duration-fast)]',
        danger
          ? 'border-[rgba(208,71,75,0.14)] text-[var(--error)] hover:bg-[var(--error-light)]'
          : 'border-[var(--border-default)] text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text-primary)]',
      ].join(' ')}
      aria-label={label}
    >
      {children}
    </button>
  )
}

export function DataTable<T extends { id: string }>({
  columns,
  data,
  onEdit,
  onDelete,
  extraActions,
  renderMobileCard,
  emptyMessage,
}: DataTableProps<T>) {
  const { t } = useTranslation('common')
  const resolvedEmptyMessage = emptyMessage ?? t('table.noData')
  const mobileCardRenderer =
    renderMobileCard ||
    ((row: T) => (
      <div className="space-y-4">
        <div className="space-y-3">
          {columns
            .filter((col) => !col.hideOnMobile)
            .map((col) => {
              const value = (row as any)[col.key]
              return (
                <div key={col.key as string} className="flex items-start justify-between gap-4">
                  <span className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                    {col.header}
                  </span>
                  <span className="min-w-0 text-right text-sm text-[var(--text-primary)]">
                    {col.render ? col.render(value, row) : String(value ?? '-')}
                  </span>
                </div>
              )
            })}
        </div>
        {(onEdit || onDelete || extraActions) && (
          <div className="flex justify-end gap-2 border-t border-[var(--border-default)] pt-3">
            {extraActions && <div className="flex items-center mr-2">{extraActions(row)}</div>}
            {onEdit && (
              <ActionButton label={t('actions.edit')} onClick={() => onEdit(row)}>
                <Pencil className="h-4 w-4" />
              </ActionButton>
            )}
            {onDelete && (
              <ActionButton label={t('actions.delete')} danger onClick={() => onDelete(row)}>
                <Trash2 className="h-4 w-4" />
              </ActionButton>
            )}
          </div>
        )}
      </div>
    ))

  return (
    <>
      <div className="hidden md:block">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((col) => (
                <TableHead key={col.key as string} className={col.className}>
                  {col.header}
                </TableHead>
              ))}
              {(onEdit || onDelete || extraActions) && (
                <TableHead className="w-40 text-right">{t('table.actions')}</TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.length === 0 ? (
              <EmptyRow
                colSpan={columns.length + ((onEdit || onDelete || extraActions) ? 1 : 0)}
                message={resolvedEmptyMessage}
              />
            ) : (
              data.map((row) => (
                <TableRow key={row.id}>
                  {columns.map((col) => {
                    const value = col.key.toString().includes('.')
                      ? (row as any)[col.key]
                      : row[col.key as keyof T]
                    return (
                      <TableCell key={col.key as string} className={col.className}>
                        {col.render ? col.render(value, row) : String(value ?? '-')}
                      </TableCell>
                    )
                  })}
                  {(onEdit || onDelete || extraActions) && (
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        {extraActions && <div className="flex items-center">{extraActions(row)}</div>}
                        {onEdit && (
                          <ActionButton label={t('actions.edit')} onClick={() => onEdit(row)}>
                            <Pencil className="h-4 w-4" />
                          </ActionButton>
                        )}
                        {onDelete && (
                          <ActionButton label={t('actions.delete')} danger onClick={() => onDelete(row)}>
                            <Trash2 className="h-4 w-4" />
                          </ActionButton>
                        )}
                      </div>
                    </TableCell>
                  )}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <div className="md:hidden">
        <MobileCardList data={data} renderCard={mobileCardRenderer} emptyMessage={resolvedEmptyMessage} />
      </div>
    </>
  )
}
