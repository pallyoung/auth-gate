import React from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from './ui/Input'
import { Select } from './ui/Select'
import type { AccessLogQueryParams } from '../lib/api/types'

interface AccessLogFiltersProps {
  filters: AccessLogQueryParams
  onChange: (filters: AccessLogQueryParams) => void
}

export const AccessLogFilters: React.FC<AccessLogFiltersProps> = ({ filters, onChange }) => {
  const { t } = useTranslation('accessLogs')

  const handleChange = (key: keyof AccessLogQueryParams, value: string | number | undefined) => {
    onChange({ ...filters, [key]: value })
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.clientIP')}
        </label>
        <Input
          placeholder="192.168.1.1"
          value={filters.client_ip || ''}
          onChange={(e) => handleChange('client_ip', e.target.value || undefined)}
        />
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.path')}
        </label>
        <Input
          placeholder="/api/users"
          value={filters.path || ''}
          onChange={(e) => handleChange('path', e.target.value || undefined)}
        />
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.username')}
        </label>
        <Input
          placeholder="admin"
          value={filters.username || ''}
          onChange={(e) => handleChange('username', e.target.value || undefined)}
        />
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
          {t('filters.authResult')}
        </label>
        <Select
          value={filters.auth_result || ''}
          onChange={(e) => handleChange('auth_result', e.target.value || undefined)}
        >
          <option value="">{t('filters.all')}</option>
          <option value="pass">{t('filters.pass')}</option>
          <option value="fail">{t('filters.fail')}</option>
          <option value="none">{t('filters.none')}</option>
        </Select>
      </div>
    </div>
  )
}
