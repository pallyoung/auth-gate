import { useTranslation } from 'react-i18next'
import type { AccessLogAggregateGroup } from '../lib/api/types'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from './ui/Table'

export function AccessLogAggregateTable({ groups, groupBy, onSelect }: { groups: AccessLogAggregateGroup[]; groupBy: string; onSelect: (group: AccessLogAggregateGroup) => void }) {
  const { t, i18n } = useTranslation('accessLogs')
  const number = (value: number, decimals = 0) => value.toLocaleString(i18n.resolvedLanguage, { maximumFractionDigits: decimals })
  return <Table>
    <TableHeader><TableRow>
      {['colKey', 'colCount', 'colErrors', 'colAuthFails', 'colAvgLatency', 'colP95Latency'].map((key, index) => <TableHead key={key} className={index ? 'text-right' : undefined}>{t(`aggregate.${key}`)}</TableHead>)}
    </TableRow></TableHeader>
    <TableBody>{groups.map((group) => {
      const label = group.route_name || (groupBy === 'auth_result' && group.key ? t(`filters.${group.key}`) : group.key) || t('aggregate.emptyKey')
      return <TableRow key={group.key}>
        <TableCell className="max-w-80 break-all text-sm">
          {group.key ? <button className="cursor-pointer rounded text-left font-medium text-[var(--primary-500)] underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--primary-500)]" aria-label={t('aggregate.drillDown', { key: label })} onClick={() => onSelect(group)}>{label}</button> : label}
        </TableCell>
        <TableCell className="text-right font-mono">{number(group.count)}</TableCell>
        <TableCell className="whitespace-nowrap text-right font-mono">{number(group.errors)} <span className="text-xs text-[var(--text-muted)]">({number(group.error_rate, 1)}%)</span></TableCell>
        <TableCell className="text-right font-mono">{number(group.auth_failures)}</TableCell>
        <TableCell className="whitespace-nowrap text-right font-mono">{number(group.avg_latency_ms)} ms</TableCell>
        <TableCell className="whitespace-nowrap text-right font-mono">{number(group.p95_latency_ms)} ms</TableCell>
      </TableRow>
    })}</TableBody>
  </Table>
}
