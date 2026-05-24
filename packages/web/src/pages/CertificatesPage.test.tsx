import { screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { CertificatesPage } from './CertificatesPage'
import { renderWithI18n } from '../test/render'

const RealDate = Date

vi.mock('../lib/api/certificates', () => ({
  certificatesApi: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    renew: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => ({
    username: 'admin',
    role: 'admin',
    permissions: {
      can_manage_routes: true,
      can_manage_auth: true,
      can_manage_users: true,
      can_view_logs: true,
    },
  }),
}))

describe('CertificatesPage i18n', () => {
  beforeEach(async () => {
    const fixedNow = new RealDate('2026-06-01T00:00:00Z')

    class MockDate extends RealDate {
      constructor(value?: string | number | Date) {
        super(value ?? fixedNow)
      }

      static now() {
        return fixedNow.getTime()
      }

      static parse = RealDate.parse
      static UTC = RealDate.UTC
    }

    globalThis.Date = MockDate as DateConstructor

    const { certificatesApi } = await import('../lib/api/certificates')
    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'manual',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
  })

  afterEach(() => {
    globalThis.Date = RealDate
  })

  it('renders translated status and locale-aware dates in zh-CN', async () => {
    await renderWithI18n(<CertificatesPage />, { locale: 'zh-CN' })

    const table = await screen.findByRole('table')
    const certificateName = within(table).getByText('Wildcard')
    const certificateRow = certificateName.closest('tr')

    expect(certificateRow).not.toBeNull()
    expect(within(certificateRow as HTMLTableRowElement).getByText('有效')).toBeInTheDocument()
    expect(within(certificateRow as HTMLTableRowElement).getByText('手动（DIY）')).toBeInTheDocument()

    expect(screen.getByRole('heading', { name: '证书' })).toBeInTheDocument()
    expect(within(certificateRow as HTMLTableRowElement).getByText('2026年6月20日')).toBeInTheDocument()
    expect(within(certificateRow as HTMLTableRowElement).getByText('剩余 19 天')).toBeInTheDocument()
  })
})
