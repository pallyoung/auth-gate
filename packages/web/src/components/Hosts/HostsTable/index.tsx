import { useTranslation } from 'react-i18next'
import type { HostEntry } from '../../../lib/api/types'
import { Switch } from '../../ui/Switch'
import { Button } from '../../ui/Button'

interface HostsTableProps {
  entries: HostEntry[]
  canManage: boolean
  onEdit: (entry: HostEntry) => void
  onDelete: (entry: HostEntry) => void
  onToggleEnabled: (entry: HostEntry, enabled: boolean) => void
}

export const HostsTable = ({ entries, canManage, onEdit, onDelete, onToggleEnabled }: HostsTableProps) => {
  const { t } = useTranslation('hosts')
  if (entries.length === 0) {
    return <p className="text-sm text-[var(--text-muted)]">{t('entriesEmpty')}</p>
  }
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr>
            <th className="px-2 py-2 text-left">IP</th>
            <th className="px-2 py-2 text-left">Hostnames</th>
            <th className="px-2 py-2 text-left">Comment</th>
            <th className="px-2 py-2 text-left">Enabled</th>
            {canManage && <th className="px-2 py-2 text-left">Actions</th>}
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr key={e.id} className="border-t border-[var(--border-soft)]">
              <td className="px-2 py-2 font-mono">{e.ip}</td>
              <td className="px-2 py-2 font-mono">{e.hostnames}</td>
              <td className="px-2 py-2 text-[var(--text-muted)]">{e.comment}</td>
              <td className="px-2 py-2">
                {canManage ? (
                  <Switch
                    checked={e.enabled}
                    onChange={(event) => onToggleEnabled(e, event.target.checked)}
                    aria-label="Enabled"
                  />
                ) : (
                  e.enabled ? '✓' : '—'
                )}
              </td>
              {canManage && (
                <td className="px-2 py-2">
                  <div className="flex gap-2">
                    <Button size="sm" variant="secondary" onClick={() => onEdit(e)}>
                      Edit
                    </Button>
                    <Button size="sm" variant="danger" onClick={() => onDelete(e)}>
                      Delete
                    </Button>
                  </div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
