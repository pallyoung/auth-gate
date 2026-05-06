import React from 'react'
import { cn } from '../../lib/utils'

interface TableProps extends React.HTMLAttributes<HTMLTableElement> {}

export function Table({ className, children, ...props }: TableProps) {
  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--border-default)]">
      <table
        className={cn('w-full bg-[var(--bg-card)]', className)}
        {...props}
      >
        {children}
      </table>
    </div>
  )
}

interface TableHeaderProps extends React.HTMLAttributes<HTMLTableSectionElement> {}

export function TableHeader({ className, children, ...props }: TableHeaderProps) {
  return (
    <thead className={cn('bg-[var(--neutral-50)]', className)} {...props}>
      {children}
    </thead>
  )
}

interface TableBodyProps extends React.HTMLAttributes<HTMLTableSectionElement> {}

export function TableBody({ className, children, ...props }: TableBodyProps) {
  return <tbody className={className}>{children}</tbody>
}

interface TableRowProps extends React.HTMLAttributes<HTMLTableRowElement> {
  selected?: boolean
}

export function TableRow({ selected, className, children, ...props }: TableRowProps) {
  return (
    <tr
      className={cn(
        'border-t border-[var(--border-default)] transition-colors',
        'hover:bg-[var(--bg-hover)]',
        selected && 'bg-[var(--primary-50)]',
        className
      )}
      aria-selected={selected}
      {...props}
    >
      {children}
    </tr>
  )
}

interface TableHeadProps extends React.HTMLAttributes<HTMLTableCellElement> {
  sortable?: boolean
}

export function TableHead({ sortable, className, children, ...props }: TableHeadProps) {
  return (
    <th
      className={cn(
        'h-12 px-4 text-left text-[var(--text-sm)] font-medium text-[var(--text-secondary)]',
        sortable && 'cursor-pointer select-none hover:text-[var(--text-primary)]',
        className
      )}
      scope="col"
      aria-sort={sortable ? 'none' : undefined}
      {...props}
    >
      {children}
    </th>
  )
}

interface TableCellProps extends React.HTMLAttributes<HTMLTableCellElement> {}

export function TableCell({ className, children, ...props }: TableCellProps) {
  return (
    <td
      className={cn(
        'h-14 px-4 text-[var(--text-sm)] text-[var(--text-primary)]',
        className
      )}
      {...props}
    >
      {children}
    </td>
  )
}

interface EmptyRowProps {
  colSpan: number
  message?: string
}

export function EmptyRow({ colSpan, message = 'No data' }: EmptyRowProps) {
  return (
    <tr>
      <td
        colSpan={colSpan}
        className="h-32 text-center text-[var(--text-muted)]"
        role="status"
      >
        {message}
      </td>
    </tr>
  )
}

interface MobileCardListProps<T> {
  data: T[]
  renderCard: (item: T) => React.ReactNode
  emptyMessage?: string
}

export function MobileCardList<T extends { id: string }>({
  data,
  renderCard,
  emptyMessage = 'No data',
}: MobileCardListProps<T>) {
  if (data.length === 0) {
    return (
      <div className="py-12 text-center text-[var(--text-muted)]" role="status">
        {emptyMessage}
      </div>
    )
  }

  return (
    <div className="space-y-3" role="list">
      {data.map((item) => (
        <div
          key={item.id}
          className="bg-[var(--bg-card)] rounded-[var(--radius-lg)] border border-[var(--border-default)] p-4"
          role="listitem"
        >
          {renderCard(item)}
        </div>
      ))}
    </div>
  )
}
