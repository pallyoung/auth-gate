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
  onDelete?: (row: T) => void | Promise<void>
  extraActions?: (row: T) => React.ReactNode
  renderMobileCard?: (row: T) => React.ReactNode
  emptyMessage?: string
}

function getColumnValue<T>(row: T, key: keyof T | string) {
  const path = String(key)

  if (!path.includes('.')) {
    return row[key as keyof T]
  }

  return path.split('.').reduce<unknown>((value, segment) => {
    if (value === null || value === undefined || typeof value !== 'object') {
      return undefined
    }

    return (value as Record<string, unknown>)[segment]
  }, row)
}

function ActionButton({
  label,
  danger = false,
  onClick,
  disabled = false,
  children,
}: {
  label: string
  danger?: boolean
  onClick: () => void
  disabled?: boolean
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={[
        'flex h-11 w-11 items-center justify-center rounded-full border transition-all duration-[var(--duration-fast)] md:h-9 md:w-9',
        'disabled:cursor-not-allowed disabled:opacity-50',
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
  const [pendingDeleteIds, setPendingDeleteIds] = React.useState<string[]>([])
  const activeDeleteIdsRef = React.useRef(new Set<string>())
  const resolvedEmptyMessage = emptyMessage ?? t('table.noData')
  const handleDelete = async (row: T) => {
    if (!onDelete || activeDeleteIdsRef.current.has(row.id)) {
      return
    }

    activeDeleteIdsRef.current.add(row.id)
    setPendingDeleteIds((current) => (
      current.includes(row.id) ? current : [...current, row.id]
    ))

    try {
      await onDelete(row)
    } finally {
      activeDeleteIdsRef.current.delete(row.id)
      setPendingDeleteIds((current) => current.filter((id) => id !== row.id))
    }
  }
  const mobileCardRenderer =
    renderMobileCard ||
    ((row: T) => (
      <div className="space-y-4">
        <div className="space-y-3">
          {columns
            .filter((col) => !col.hideOnMobile)
            .map((col) => {
              const value = getColumnValue(row, col.key)
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
              <ActionButton
                label={t('actions.delete')}
                danger
                disabled={pendingDeleteIds.includes(row.id)}
                onClick={() => {
                  void handleDelete(row)
                }}
              >
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
                    const value = getColumnValue(row, col.key)
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
                          <ActionButton
                            label={t('actions.delete')}
                            danger
                            disabled={pendingDeleteIds.includes(row.id)}
                            onClick={() => {
                              void handleDelete(row)
                            }}
                          >
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
