import React from 'react'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, EmptyRow, MobileCardList } from './ui'
import { Pencil, Trash2 } from 'lucide-react'
import { cn } from '../lib/utils'

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
  renderMobileCard?: (row: T) => React.ReactNode
  emptyMessage?: string
}

export function DataTable<T extends { id: string }>({
  columns,
  data,
  onEdit,
  onDelete,
  renderMobileCard,
  emptyMessage = 'No data',
}: DataTableProps<T>) {
  const mobileCardRenderer = renderMobileCard || ((row: T) => (
    <div className="space-y-2">
      {columns
        .filter(col => !col.hideOnMobile)
        .map(col => {
          const value = (row as any)[col.key]
          return (
            <div key={col.key as string} className="flex justify-between items-start">
              <span className="text-[var(--text-muted)] text-xs">{col.header}</span>
              <span className="text-[var(--text-sm)] text-right">
                {col.render ? col.render(value, row) : String(value ?? '-')}
              </span>
            </div>
          )
        })}
      {(onEdit || onDelete) && (
        <div className="flex justify-end gap-2 pt-2 border-t border-[var(--border-default)]">
          {onEdit && (
            <button
              onClick={() => onEdit(row)}
              className="p-2 rounded hover:bg-[var(--bg-hover)] text-[var(--text-muted)]"
            >
              <Pencil className="w-4 h-4" />
            </button>
          )}
          {onDelete && (
            <button
              onClick={() => onDelete(row)}
              className="p-2 rounded hover:bg-[var(--error-light)] text-[var(--text-muted)]"
            >
              <Trash2 className="w-4 h-4" />
            </button>
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
              {(onEdit || onDelete) && (
                <TableHead className="w-24 text-right">Actions</TableHead>
              )}
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.length === 0 ? (
              <EmptyRow colSpan={columns.length + (onEdit || onDelete ? 1 : 0)} message={emptyMessage} />
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
                  {(onEdit || onDelete) && (
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        {onEdit && (
                          <button
                            onClick={() => onEdit(row)}
                            className="p-1.5 rounded hover:bg-[var(--bg-hover)] text-[var(--text-muted)] hover:text-[var(--text-primary)]"
                            aria-label="Edit"
                          >
                            <Pencil className="w-4 h-4" />
                          </button>
                        )}
                        {onDelete && (
                          <button
                            onClick={() => onDelete(row)}
                            className="p-1.5 rounded hover:bg-[var(--error-light)] text-[var(--text-muted)] hover:text-[var(--error)]"
                            aria-label="Delete"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
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
        <MobileCardList
          data={data}
          renderCard={mobileCardRenderer}
          emptyMessage={emptyMessage}
        />
      </div>
    </>
  )
}
