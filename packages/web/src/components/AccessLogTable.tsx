import { useTranslation } from 'react-i18next'
import type { AccessLogEntry } from '../lib/api/types'
import { Badge } from './ui/Badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from './ui/Table'

export function AccessLogTable({ entries, onSelect }: { entries: AccessLogEntry[]; onSelect: (entry: AccessLogEntry) => void }) {
  const { t, i18n } = useTranslation('accessLogs')
  return <Table>
    <TableHeader><TableRow>
      {['timestamp', 'host', 'method', 'path', 'route', 'statusCode', 'clientIP', 'username', 'authResult', 'latency'].map((key) => <TableHead key={key}>{t(`table.${key}`)}</TableHead>)}
    </TableRow></TableHeader>
    <TableBody>{entries.map((entry) => <TableRow key={entry.request_id}>
      <TableCell className="whitespace-nowrap text-xs text-[var(--text-muted)]">{new Date(entry.timestamp).toLocaleString(i18n.resolvedLanguage)}</TableCell>
      <TableCell className="max-w-56 break-all font-mono text-xs">{entry.host || <span className="text-[var(--text-muted)]">{t('table.unknownHost')}</span>}</TableCell>
      <TableCell><Badge>{entry.method}</Badge></TableCell>
      <TableCell className="max-w-80 break-all font-mono text-sm">
        <button className="cursor-pointer rounded text-left text-[var(--primary-500)] underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--primary-500)]" aria-label={t('table.openDetail', { id: entry.request_id })} onClick={() => onSelect(entry)}>{entry.path}</button>
      </TableCell>
      <TableCell className="max-w-40 truncate text-sm" title={entry.route_name || entry.route_id}>{entry.route_name || entry.route_id || '—'}</TableCell>
      <TableCell><Badge variant={entry.status_code >= 400 ? 'error' : entry.status_code >= 300 ? 'warning' : 'success'}>{entry.status_code}</Badge></TableCell>
      <TableCell className="font-mono text-xs">{entry.client_ip}</TableCell>
      <TableCell className="text-sm">{entry.username || '—'}</TableCell>
      <TableCell><Badge variant={entry.auth_result === 'fail' ? 'error' : entry.auth_result === 'pass' ? 'success' : 'default'} className="normal-case tracking-normal">{t(`filters.${entry.auth_result}`)}</Badge></TableCell>
      <TableCell className="whitespace-nowrap text-sm">{entry.backend_latency_ms.toLocaleString(i18n.resolvedLanguage)} ms</TableCell>
    </TableRow>)}</TableBody>
  </Table>
}
