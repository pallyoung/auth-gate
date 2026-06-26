import React from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from './ui/Modal'
import { Badge } from './ui/Badge'
import type { AccessLogEntry } from '../lib/api/types'

interface AccessLogDetailProps {
  entry: AccessLogEntry | null
  onClose: () => void
}

export const AccessLogDetail: React.FC<AccessLogDetailProps> = ({ entry, onClose }) => {
  const { t } = useTranslation('accessLogs')

  if (!entry) {
    return null
  }

  const getStatusBadge = (statusCode: number) => {
    if (statusCode >= 200 && statusCode < 300) {
      return <Badge variant="success">{statusCode}</Badge>
    }
    if (statusCode >= 300 && statusCode < 400) {
      return <Badge variant="warning">{statusCode}</Badge>
    }
    if (statusCode >= 400) {
      return <Badge variant="error">{statusCode}</Badge>
    }
    return <Badge>{statusCode}</Badge>
  }

  const getAuthBadge = (authResult: string) => {
    switch (authResult) {
      case 'pass':
        return <Badge variant="success">{t('filters.pass')}</Badge>
      case 'fail':
        return <Badge variant="error">{t('filters.fail')}</Badge>
      case 'none':
        return <Badge>{t('filters.none')}</Badge>
      default:
        return <Badge>{authResult}</Badge>
    }
  }

  return (
    <Modal open={!!entry} onClose={onClose} title={t('detail.title')} modalSize="lg">
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.timestamp')}
            </label>
            <p className="mt-1 text-sm text-[var(--text-primary)]">
              {new Date(entry.timestamp).toLocaleString('zh-CN')}
            </p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.method')}
            </label>
            <p className="mt-1">
              <Badge variant="primary">{entry.method}</Badge>
            </p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.path')}
            </label>
            <p className="mt-1 font-mono text-sm text-[var(--text-primary)]">
              {entry.path}
            </p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.statusCode')}
            </label>
            <p className="mt-1">{getStatusBadge(entry.status_code)}</p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.clientIP')}
            </label>
            <p className="mt-1 font-mono text-sm text-[var(--text-primary)]">
              {entry.client_ip}
            </p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.username')}
            </label>
            <p className="mt-1 text-sm text-[var(--text-primary)]">
              {entry.username || '-'}
            </p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.authResult')}
            </label>
            <p className="mt-1">{getAuthBadge(entry.auth_result)}</p>
          </div>

          <div>
            <label className="text-sm font-medium text-[var(--text-secondary)]">
              {t('table.latency')}
            </label>
            <p className="mt-1 text-sm text-[var(--text-primary)]">
              {entry.backend_latency_ms} ms
            </p>
          </div>
        </div>

        <div className="border-t border-[var(--border-default)] pt-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium text-[var(--text-secondary)]">
                {t('detail.requestID')}
              </label>
              <p className="mt-1 font-mono text-xs text-[var(--text-muted)]">
                {entry.request_id}
              </p>
            </div>

            <div>
              <label className="text-sm font-medium text-[var(--text-secondary)]">
                {t('detail.routeID')}
              </label>
              <p className="mt-1 font-mono text-xs text-[var(--text-muted)]">
                {entry.route_id}
              </p>
            </div>

            <div>
              <label className="text-sm font-medium text-[var(--text-secondary)]">
                {t('detail.backendURL')}
              </label>
              <p className="mt-1 font-mono text-xs text-[var(--text-muted)]">
                {entry.backend_url || '-'}
              </p>
            </div>

            <div>
              <label className="text-sm font-medium text-[var(--text-secondary)]">
                {t('detail.userAgent')}
              </label>
              <p className="mt-1 truncate text-xs text-[var(--text-muted)]" title={entry.user_agent}>
                {entry.user_agent}
              </p>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  )
}
