import { act, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { CertificatesPage } from './CertificatesPage'
import { certificatesApi } from '../lib/api/certificates'
import { ApiError } from '../lib/api/client'
import { renderWithI18n } from '../test/render'

const RealDate = Date
const sessionUser = {
  username: 'admin',
  role: 'admin',
  permissions: {
    can_manage_routes: true,
    can_manage_auth: true,
    can_manage_users: true,
    can_view_logs: true,
  },
  features: {
    certificates: true,
  },
}

vi.mock('../lib/api/certificates', () => ({
  certificatesApi: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    renew: vi.fn(),
  },
}))

vi.mock('../lib/session-store', () => ({
  getSessionUser: () => sessionUser,
}))

describe('CertificatesPage i18n', () => {
  beforeEach(async () => {
    sessionUser.features = {
      certificates: true,
    }
    sessionUser.permissions = {
      can_manage_routes: true,
      can_manage_auth: true,
      can_manage_users: true,
      can_view_logs: true,
    }

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
    vi.mocked(certificatesApi.create).mockReset()
    vi.mocked(certificatesApi.delete).mockReset()
    vi.mocked(certificatesApi.renew).mockReset()
  })

  afterEach(() => {
    globalThis.Date = RealDate
    vi.unstubAllGlobals()
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

  it('does not request certificates when certificate automation is unavailable', async () => {
    sessionUser.features.certificates = false
    vi.mocked(certificatesApi.list).mockClear()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(screen.getByText('Certificate automation is unavailable on this server.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision Certificate' })).not.toBeInTheDocument()
    expect(vi.mocked(certificatesApi.list)).not.toHaveBeenCalled()
  })

  it('shows read-only empty state guidance when certificate management is unavailable to the account', async () => {
    sessionUser.features.certificates = true
    sessionUser.permissions.can_manage_routes = false

    vi.mocked(certificatesApi.list).mockResolvedValue([])

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(await screen.findByText('No certificates provisioned')).toBeInTheDocument()
    expect(
      screen.getByText('Your account can review certificates here, but only editors or administrators can provision or renew them.')
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'Provision your first certificate to enable HTTPS for your routes. Wildcard certificates are supported via DNS-01 challenge.'
      )
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision First Certificate' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision Certificate' })).not.toBeInTheDocument()
  })

  it('treats accounts with missing certificate-management permissions as read-only', async () => {
    sessionUser.features.certificates = true
    sessionUser.permissions = undefined as any

    vi.mocked(certificatesApi.list).mockResolvedValue([])

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(await screen.findByText('No certificates provisioned')).toBeInTheDocument()
    expect(
      screen.getByText('Your account can review certificates here, but only editors or administrators can provision or renew them.')
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision First Certificate' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision Certificate' })).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw backend error when loading certificates fails', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing certificates.')
    ).toBeInTheDocument()
    expect(screen.queryByText('unauthorized')).not.toBeInTheDocument()
  })

  it('does not fall back to the empty certificate state when the certificate directory fails to load', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(
      await screen.findByText(
        'The current certificate list could not be loaded. Resolve the current error before reviewing or managing certificates.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('No certificates provisioned')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision Certificate' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Provision First Certificate' })).not.toBeInTheDocument()
  })

  it('does not show normal directory counts or metrics when the certificate directory is unavailable', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(new ApiError('unauthorized', 401, 'unauthorized'))

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    await screen.findByText(
      'The current certificate list could not be loaded. Resolve the current error before reviewing or managing certificates.'
    )

    expect(screen.queryByText('0 certificates')).not.toBeInTheDocument()
    expect(screen.queryByText('Active Certificates')).not.toBeInTheDocument()
    expect(screen.queryByText('In Progress')).not.toBeInTheDocument()
    expect(screen.queryByText('Failed')).not.toBeInTheDocument()
  })

  it('shows session expiry guidance instead of the raw invalid token error when loading certificates fails', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing certificates.')
    ).toBeInTheDocument()
    expect(screen.queryByText('invalid token')).not.toBeInTheDocument()
  })

  it('shows a directory loading message instead of mixed load/save copy when loading certificates fails with a storage error', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(
      new ApiError('failed to list certificates', 500, 'database_error')
    )

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(
      await screen.findByText('The certificate directory could not be loaded. Try again in a moment.')
    ).toBeInTheDocument()
    expect(screen.queryByText('Certificate data could not be loaded or saved. Try again in a moment.')).not.toBeInTheDocument()
    expect(screen.queryByText('failed to list certificates')).not.toBeInTheDocument()
  })

  it('shows a duplicate domain message in the form and re-enables submit after a failed create', async () => {
    vi.mocked(certificatesApi.list).mockResolvedValue([])
    vi.mocked(certificatesApi.create).mockRejectedValue(
      new ApiError('certificate already exists for domain: *.example.com', 400, 'domain_exists')
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Provision Certificate' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    const dialog = await screen.findByRole('dialog')

    await user.type(within(dialog).getByLabelText('Certificate Name'), 'Wildcard')
    await user.type(within(dialog).getByLabelText('Domain'), '*.example.com')
    await user.type(within(dialog).getByLabelText('CloudFlare API Token'), 'cf_test_token')
    await user.click(within(dialog).getByRole('button', { name: 'Provision Certificate' }))

    expect(
      await within(dialog).findByText(
        'A certificate for this domain already exists. Use the existing certificate or choose a different domain.'
      )
    ).toBeInTheDocument()

    await waitFor(() => {
      expect(within(dialog).getByRole('button', { name: 'Provision Certificate' })).not.toBeDisabled()
    })

    expect(
      screen.queryByText('certificate already exists for domain: *.example.com')
    ).not.toBeInTheDocument()
  })

  it('shows a certificate name message in the form instead of the raw backend error', async () => {
    vi.mocked(certificatesApi.list).mockResolvedValue([])
    vi.mocked(certificatesApi.create).mockRejectedValue(
      new ApiError('certificate name required', 400, 'invalid_name')
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Provision Certificate' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    const dialog = await screen.findByRole('dialog')

    await user.type(within(dialog).getByLabelText('Certificate Name'), '   ')
    await user.type(within(dialog).getByLabelText('Domain'), '*.example.com')
    await user.type(within(dialog).getByLabelText('CloudFlare API Token'), 'cf_test_token')
    await user.click(within(dialog).getByRole('button', { name: 'Provision Certificate' }))

    expect(
      await within(dialog).findByText('Enter a certificate name before provisioning.')
    ).toBeInTheDocument()

    expect(screen.queryByText('certificate name required')).not.toBeInTheDocument()
  })

  it('shows a refresh guidance message instead of the raw API error when renewal fails', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockRejectedValue(
      new ApiError('certificate not found: cert-1', 404, 'cert_not_found')
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    const [renewButton] = await screen.findAllByRole('button', { name: 'Renew' })
    await user.click(renewButton)

    expect(
      await screen.findByText('This certificate no longer exists. Refresh the page and try again.')
    ).toBeInTheDocument()
    expect(screen.queryByText('certificate not found: cert-1')).not.toBeInTheDocument()
  })

  it('does not submit a parent form when the renew action is clicked', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockResolvedValue(undefined)

    const onSubmit = vi.fn((event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
    })
    const user = userEvent.setup()

    await renderWithI18n(
      <form onSubmit={onSubmit}>
        <CertificatesPage />
      </form>,
      { locale: 'en' }
    )

    const [renewButton] = await screen.findAllByRole('button', { name: 'Renew' })
    await user.click(renewButton)

    expect(certificatesApi.renew).toHaveBeenCalledTimes(1)
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('prevents duplicate renewals while a certificate renewal is pending', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    let resolveRenew: (() => void) | undefined

    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveRenew = resolve
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    const [renewButton] = await screen.findAllByRole('button', { name: 'Renew' })
    await user.click(renewButton)
    await user.click(renewButton)

    expect(certificatesApi.renew).toHaveBeenCalledTimes(1)

    resolveRenew?.()
  })

  it('prevents back-to-back native renew clicks before the renewing state re-renders', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    let resolveRenew: (() => void) | undefined

    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveRenew = resolve
        })
    )

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    const [renewButton] = await screen.findAllByRole('button', { name: 'Renew' })

    await act(async () => {
      renewButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      renewButton.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(certificatesApi.renew).toHaveBeenCalledTimes(1)

    resolveRenew?.()
  })

  it('keeps each certificate renewal action disabled until its own request settles', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    const renewResolvers = new Map<string, () => void>()

    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
      {
        id: 'cert-2',
        name: 'API',
        domain: 'api.example.com',
        cert_path: '/tmp/cert-2.pem',
        key_path: '/tmp/key-2.pem',
        dns_provider: 'route53',
        status: 'active',
        not_after: '2026-06-21T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockImplementation(
      (id: string) =>
        new Promise<void>((resolve) => {
          renewResolvers.set(id, resolve)
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    const table = await screen.findByRole('table')
    const wildcardRow = within(table).getByText('Wildcard').closest('tr')
    const apiRow = within(table).getByText('API').closest('tr')

    expect(wildcardRow).not.toBeNull()
    expect(apiRow).not.toBeNull()
    expect(within(table).getByText('Wildcard')).toBeInTheDocument()
    expect(within(table).getByText('API')).toBeInTheDocument()

    await user.click(within(wildcardRow as HTMLTableRowElement).getByRole('button', { name: 'Renew' }))
    await user.click(within(apiRow as HTMLTableRowElement).getByRole('button', { name: 'Renew' }))

    expect(certificatesApi.renew).toHaveBeenCalledTimes(2)
    expect(within(wildcardRow as HTMLTableRowElement).getByRole('button', { name: 'Renewing...' })).toBeDisabled()
    expect(within(apiRow as HTMLTableRowElement).getByRole('button', { name: 'Renewing...' })).toBeDisabled()

    renewResolvers.get('cert-1')?.()
    renewResolvers.get('cert-2')?.()
  })

  it('keeps the newest certificate refresh results when an older renewal refresh resolves later', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))

    const renewResolvers = new Map<string, () => void>()
    const listResolvers: Array<(value: Awaited<ReturnType<typeof certificatesApi.list>>) => void> = []

    vi.mocked(certificatesApi.list).mockImplementation(
      () =>
        new Promise((resolve) => {
          listResolvers.push(resolve)
        })
    )
    vi.mocked(certificatesApi.renew).mockImplementation(
      (id: string) =>
        new Promise<void>((resolve) => {
          renewResolvers.set(id, resolve)
        })
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    await act(async () => {
      listResolvers[0]?.([
        {
          id: 'cert-1',
          name: 'Wildcard',
          domain: '*.example.com',
          cert_path: '/tmp/cert.pem',
          key_path: '/tmp/key.pem',
          dns_provider: 'cloudflare',
          status: 'active',
          not_after: '2026-06-20T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:00:00Z',
        },
        {
          id: 'cert-2',
          name: 'API',
          domain: 'api.example.com',
          cert_path: '/tmp/cert-2.pem',
          key_path: '/tmp/key-2.pem',
          dns_provider: 'route53',
          status: 'active',
          not_after: '2026-06-21T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:00:00Z',
        },
      ])
      await Promise.resolve()
    })

    const table = await screen.findByRole('table')
    const wildcardRow = within(table).getByText('Wildcard').closest('tr')
    const apiRow = within(table).getByText('API').closest('tr')

    expect(wildcardRow).not.toBeNull()
    expect(apiRow).not.toBeNull()

    await user.click(within(wildcardRow as HTMLTableRowElement).getByRole('button', { name: 'Renew' }))
    await user.click(within(apiRow as HTMLTableRowElement).getByRole('button', { name: 'Renew' }))

    await act(async () => {
      renewResolvers.get('cert-1')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(listResolvers).toHaveLength(2)
    })

    await act(async () => {
      renewResolvers.get('cert-2')?.()
      await Promise.resolve()
    })

    await waitFor(() => {
      expect(listResolvers).toHaveLength(3)
    })

    await act(async () => {
      listResolvers[2]?.([
        {
          id: 'cert-1',
          name: 'Wildcard Rotated',
          domain: '*.example.com',
          cert_path: '/tmp/cert.pem',
          key_path: '/tmp/key.pem',
          dns_provider: 'cloudflare',
          status: 'active',
          not_after: '2026-07-20T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-06-02T00:00:00Z',
        },
        {
          id: 'cert-2',
          name: 'API Rotated',
          domain: 'api.example.com',
          cert_path: '/tmp/cert-2.pem',
          key_path: '/tmp/key-2.pem',
          dns_provider: 'route53',
          status: 'active',
          not_after: '2026-07-21T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-06-02T00:00:00Z',
        },
      ])
      await Promise.resolve()
    })

    expect(await within(table).findByText('Wildcard Rotated')).toBeInTheDocument()
    expect(within(table).getByText('API Rotated')).toBeInTheDocument()

    await act(async () => {
      listResolvers[1]?.([
        {
          id: 'cert-1',
          name: 'Wildcard',
          domain: '*.example.com',
          cert_path: '/tmp/cert.pem',
          key_path: '/tmp/key.pem',
          dns_provider: 'cloudflare',
          status: 'active',
          not_after: '2026-06-20T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:00:00Z',
        },
        {
          id: 'cert-2',
          name: 'API',
          domain: 'api.example.com',
          cert_path: '/tmp/cert-2.pem',
          key_path: '/tmp/key-2.pem',
          dns_provider: 'route53',
          status: 'active',
          not_after: '2026-06-21T00:00:00Z',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:00:00Z',
        },
      ])
      await Promise.resolve()
    })

    expect(within(table).getByText('Wildcard Rotated')).toBeInTheDocument()
    expect(within(table).getByText('API Rotated')).toBeInTheDocument()
    expect(within(table).queryByText('Wildcard')).not.toBeInTheDocument()
    expect(within(table).queryByText('API')).not.toBeInTheDocument()
  })

  it('shows permission guidance instead of the raw backend error when certificate renewal is rejected', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-1',
        name: 'Wildcard',
        domain: '*.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])
    vi.mocked(certificatesApi.renew).mockRejectedValue(
      new ApiError('insufficient permissions', 403, 'insufficient_permissions')
    )

    const user = userEvent.setup()

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    const [renewButton] = await screen.findAllByRole('button', { name: 'Renew' })
    await user.click(renewButton)

    expect(
      await screen.findByText(
        'Your account cannot manage certificates. Ask an editor or administrator to apply the change.'
      )
    ).toBeInTheDocument()
    expect(screen.queryByText('insufficient permissions')).not.toBeInTheDocument()
  })

  it('does not offer renew actions for active certificates backed by unsupported legacy DNS providers', async () => {
    vi.mocked(certificatesApi.list).mockResolvedValue([
      {
        id: 'cert-manual',
        name: 'Legacy Manual',
        domain: '*.legacy.example.com',
        cert_path: '/tmp/cert.pem',
        key_path: '/tmp/key.pem',
        dns_provider: 'manual',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
      {
        id: 'cert-pdns',
        name: 'Legacy PowerDNS',
        domain: '*.pdns.example.com',
        cert_path: '/tmp/cert-2.pem',
        key_path: '/tmp/key-2.pem',
        dns_provider: 'pdns',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
      {
        id: 'cert-cloudflare',
        name: 'Supported',
        domain: '*.supported.example.com',
        cert_path: '/tmp/cert-3.pem',
        key_path: '/tmp/key-3.pem',
        dns_provider: 'cloudflare',
        status: 'active',
        not_after: '2026-06-20T00:00:00Z',
        created_at: '2026-05-01T00:00:00Z',
        updated_at: '2026-05-01T00:00:00Z',
      },
    ])

    await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(await screen.findAllByRole('button', { name: 'Renew' })).toHaveLength(2)
  })

  it('retranslates the current certificate list error when the language changes', async () => {
    vi.mocked(certificatesApi.list).mockRejectedValue(new ApiError('invalid token', 401, 'invalid_token'))

    const view = await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    expect(
      await screen.findByText('Your session has expired. Sign in again before managing certificates.')
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(screen.getByText('你的会话已失效。请重新登录后再管理证书。')).toBeInTheDocument()
    expect(
      screen.queryByText('Your session has expired. Sign in again before managing certificates.')
    ).not.toBeInTheDocument()
  })

  it('retranslates the current certificate form error when the language changes', async () => {
    vi.mocked(certificatesApi.list).mockResolvedValue([])
    vi.mocked(certificatesApi.create).mockRejectedValue(
      new ApiError('certificate already exists for domain: *.example.com', 400, 'domain_exists')
    )

    const user = userEvent.setup()
    const view = await renderWithI18n(<CertificatesPage />, { locale: 'en' })

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Provision Certificate' })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: 'Provision Certificate' }))

    const dialog = await screen.findByRole('dialog')

    await user.type(within(dialog).getByLabelText('Certificate Name'), 'Wildcard')
    await user.type(within(dialog).getByLabelText('Domain'), '*.example.com')
    await user.type(within(dialog).getByLabelText('CloudFlare API Token'), 'cf_test_token')
    await user.click(within(dialog).getByRole('button', { name: 'Provision Certificate' }))

    expect(
      await within(dialog).findByText(
        'A certificate for this domain already exists. Use the existing certificate or choose a different domain.'
      )
    ).toBeInTheDocument()

    await act(async () => {
      await view.i18n.changeLanguage('zh-CN')
    })

    expect(
      within(dialog).getByText('该域名的证书已存在。请使用现有证书，或更换其他域名。')
    ).toBeInTheDocument()
    expect(
      screen.queryByText(
        'A certificate for this domain already exists. Use the existing certificate or choose a different domain.'
      )
    ).not.toBeInTheDocument()
  })
})
