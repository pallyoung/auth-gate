import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'
import { RouteForm } from './RouteForm'
import { renderWithI18n } from '../test/render'
import type { Route } from '../lib/api/types'

const billingRoute: Route = {
  id: 'route-1',
  name: 'Billing Route',
  host: 'billing.example.com',
  path_prefix: '/billing',
  backend: 'http://127.0.0.1:9000',
  strip_prefix: true,
  enabled: true,
  priority: 100,
  path_match_mode: '',
  rewrite_target: '',
  redirect_code: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const reportsRoute: Route = {
  id: 'route-2',
  name: 'Reports Route',
  host: 'reports.example.com',
  path_prefix: '/reports',
  backend: 'https://reports.internal',
  strip_prefix: false,
  enabled: false,
  priority: 25,
  path_match_mode: 'exact',
  rewrite_target: '/dashboard',
  redirect_code: 301,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const loadBalancedRoute = {
  id: 'route-3',
  name: 'Load Balanced Route',
  host: 'api.example.com',
  path_prefix: '/api',
  backend: '',
  backends: [
    {
      url: 'http://backend-a.example.com',
      weight: 3,
      dial_timeout_ms: 1500,
      read_timeout_ms: 2500,
      write_timeout_ms: 3500,
      max_idle_conns: 10,
    },
    { url: 'https://backend-b.example.com', weight: 1 },
  ],
  strip_prefix: true,
  enabled: true,
  priority: 40,
  tls_cert: '/etc/ssl/certs/api.pem',
  tls_key: '/etc/ssl/private/api.key',
  tls_enabled: true,
  timeout_ms: 4500,
  retry_attempts: 3,
  path_match_mode: '',
  rewrite_target: '',
  redirect_code: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
} as Route

describe('RouteForm', () => {
  it('exposes select fields with accessible labels', async () => {
    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    expect(screen.getByRole('combobox', { name: 'Path Match Mode' })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Redirect Code' })).toBeInTheDocument()
  })

  it('updates path prefix guidance for regex path match modes', async () => {
    const user = userEvent.setup()

    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    const pathPrefixInput = screen.getByLabelText('Path Prefix')
    const pathMatchModeSelect = screen.getByRole('combobox', { name: 'Path Match Mode' })

    expect(pathPrefixInput).toHaveAttribute('placeholder', '/api/v1 or empty to match all')
    expect(screen.getByText('Leave empty to match all paths.')).toBeInTheDocument()

    await user.selectOptions(pathMatchModeSelect, 'regex')

    expect(pathPrefixInput).toHaveAttribute('placeholder', '^/api/v\\d+')
    expect(
      screen.getByText('Use a regular expression such as ^/api/v\\d+ to match paths.')
    ).toBeInTheDocument()
    expect(screen.queryByText('Leave empty to match all paths.')).not.toBeInTheDocument()
  })

  it('prevents duplicate submissions while a route save is pending', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()

    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')

    const submitButton = screen.getByRole('button', { name: 'Create Route' })
    await user.click(submitButton)
    await user.click(submitButton)

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('prevents back-to-back native submissions before the submitting state re-renders', async () => {
    let resolveSubmit: (() => void) | undefined
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve
        })
    )
    const user = userEvent.setup()
    const { container } = await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')

    const form = container.querySelector('form')
    expect(form).not.toBeNull()

    await act(async () => {
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      form?.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
      await Promise.resolve()
    })

    expect(onSubmit).toHaveBeenCalledTimes(1)

    resolveSubmit?.()
  })

  it('submits a trimmed rewrite target when the input has surrounding whitespace', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.type(screen.getByLabelText('Rewrite Target'), '  /dashboard  ')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          rewrite_target: '/dashboard',
        })
      )
    })
  })

  it('submits a lowercased trimmed host when the input has surrounding whitespace and uppercase letters', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Host'), '  API.EXAMPLE.COM  ')
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          host: 'api.example.com',
        })
      )
    })
  })

  it('submits a bare ipv6 host when the input uses brackets', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.click(screen.getByLabelText('Host'))
    await user.paste('  [2001:DB8::1]  ')
    await user.type(screen.getByLabelText('Path Prefix'), '/billing')
    await user.type(screen.getByLabelText('Backend'), 'http://example.com')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          host: '2001:db8::1',
        })
      )
    })
  })

  it('submits pooled backends and tls certificate selection', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    const certs = [
      {
        id: 'cert-1',
        name: 'API Cert',
        domain: 'api.example.com',
        cert_path: '/data/certs/api.example.com/cert.pem',
        key_path: '/data/certs/api.example.com/key.pem',
        source: 'local_ca',
        status: 'active' as const,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    ]

    await renderWithI18n(
      <RouteForm route={null} certificates={certs} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.type(screen.getByLabelText('Path Prefix'), '/api')
    await user.click(screen.getByRole('button', { name: 'Add Backend' }))
    await user.type(screen.getByLabelText('Backend URL 1'), '  http://backend-a.example.com  ')
    await user.clear(screen.getByLabelText('Weight 1'))
    await user.type(screen.getByLabelText('Weight 1'), '2')
    await user.clear(screen.getByLabelText('Dial Timeout (ms) 1'))
    await user.type(screen.getByLabelText('Dial Timeout (ms) 1'), '1200')
    await user.clear(screen.getByLabelText('Read Timeout (ms) 1'))
    await user.type(screen.getByLabelText('Read Timeout (ms) 1'), '2200')
    await user.clear(screen.getByLabelText('Write Timeout (ms) 1'))
    await user.type(screen.getByLabelText('Write Timeout (ms) 1'), '3200')
    await user.clear(screen.getByLabelText('Timeout (ms)'))
    await user.type(screen.getByLabelText('Timeout (ms)'), '4500')
    await user.clear(screen.getByLabelText('Retry Attempts'))
    await user.type(screen.getByLabelText('Retry Attempts'), '3')
    await user.click(screen.getByLabelText('Enable HTTPS'))
    await user.selectOptions(screen.getByRole('combobox', { name: 'System Certificate' }), 'cert-1')
    await user.click(screen.getByRole('button', { name: 'Create Route' }))

    await waitFor(() => {
        expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          backend: '',
          backends: [{
            url: 'http://backend-a.example.com',
            weight: 2,
            dial_timeout_ms: 1200,
            read_timeout_ms: 2200,
            write_timeout_ms: 3200,
          }],
          tls_enabled: true,
          certificate_id: 'cert-1',
          timeout_ms: 4500,
          retry_attempts: 3,
        })
      )
    })
  })

  it('hydrates backend pool and tls fields from an existing load-balanced route', async () => {
    await renderWithI18n(
      <RouteForm route={loadBalancedRoute} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    expect(screen.getByLabelText('Backend')).toHaveValue('')
    expect(screen.getByLabelText('Backend URL 1')).toHaveValue('http://backend-a.example.com')
    expect(screen.getByLabelText('Weight 1')).toHaveValue(3)
    expect(screen.getByLabelText('Dial Timeout (ms) 1')).toHaveValue(1500)
    expect(screen.getByLabelText('Read Timeout (ms) 1')).toHaveValue(2500)
    expect(screen.getByLabelText('Write Timeout (ms) 1')).toHaveValue(3500)
    expect(screen.getByLabelText('Max Idle Conns 1')).toHaveValue(10)
    expect(screen.getByLabelText('Backend URL 2')).toHaveValue('https://backend-b.example.com')
    expect(screen.getByLabelText('Weight 2')).toHaveValue(1)
    expect(screen.getByLabelText('Enable HTTPS')).toBeChecked()
    expect(screen.getByLabelText('Timeout (ms)')).toHaveValue(4500)
    expect(screen.getByLabelText('Retry Attempts')).toHaveValue(3)
    expect(screen.getByText(/This route uses a custom certificate path/)).toBeInTheDocument()
  })

  it('refreshes form fields when the edited route changes while the form stays mounted', async () => {
    const view = await renderWithI18n(
      <RouteForm route={billingRoute} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    expect(screen.getByLabelText('Name')).toHaveValue('Billing Route')
    expect(screen.getByLabelText('Host')).toHaveValue('billing.example.com')
    expect(screen.getByLabelText('Path Prefix')).toHaveValue('/billing')
    expect(screen.getByLabelText('Backend')).toHaveValue('http://127.0.0.1:9000')

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <RouteForm route={reportsRoute} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />
      </I18nextProvider>
    )

    expect(screen.getByLabelText('Name')).toHaveValue('Reports Route')
    expect(screen.getByLabelText('Host')).toHaveValue('reports.example.com')
    expect(screen.getByLabelText('Path Prefix')).toHaveValue('/reports')
    expect(screen.getByLabelText('Backend')).toHaveValue('https://reports.internal')
    expect(screen.getByRole('combobox', { name: 'Path Match Mode' })).toHaveValue('exact')
    expect(screen.getByRole('combobox', { name: 'Redirect Code' })).toHaveValue('301')
  })

  it('refreshes form fields when the same route receives updated values while the form stays mounted', async () => {
    const view = await renderWithI18n(
      <RouteForm route={billingRoute} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    view.rerender(
      <I18nextProvider i18n={view.i18n}>
        <RouteForm
          route={{
            ...billingRoute,
            name: 'Billing Route v2',
            host: 'billing-v2.example.com',
            path_prefix: '/billing-v2',
            backend: 'https://billing-v2.internal',
            priority: 250,
            strip_prefix: false,
            enabled: false,
          }}
          certificates={[]}
          onSubmit={vi.fn()}
          onCancel={vi.fn()}
        />
      </I18nextProvider>
    )

    expect(screen.getByLabelText('Name')).toHaveValue('Billing Route v2')
    expect(screen.getByLabelText('Host')).toHaveValue('billing-v2.example.com')
    expect(screen.getByLabelText('Path Prefix')).toHaveValue('/billing-v2')
    expect(screen.getByLabelText('Backend')).toHaveValue('https://billing-v2.internal')
    expect(screen.getByLabelText('Priority')).toHaveValue(250)
    expect(screen.getByRole('checkbox', { name: /strip prefix/i, hidden: true })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: /enabled/i, hidden: true })).not.toBeChecked()
  })

  it('renders the Header Rules section with empty state text', async () => {
    await renderWithI18n(
      <RouteForm route={null} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    // The header rules section should be visible
    expect(screen.getByText('Header Rules')).toBeInTheDocument()
    expect(screen.getByText('Set Request Headers')).toBeInTheDocument()
    expect(screen.getByText('Remove Request Headers')).toBeInTheDocument()
    expect(screen.getByText('Add Response Headers')).toBeInTheDocument()
    expect(screen.getByText('Remove Response Headers')).toBeInTheDocument()

    // All should show empty state
    const emptyStates = screen.getAllByText('No headers configured.')
    expect(emptyStates.length).toBe(4)
  })

  it('hydrates header manipulation fields from an existing route', async () => {
    const routeWithHeaders: Route = {
      ...billingRoute,
      set_request_headers: { 'X-Custom-Token': 'secret123' },
      remove_request_headers: ['Cookie'],
      add_response_headers: { 'X-Request-Id': 'req-42' },
      remove_response_headers: ['X-Powered-By'],
    }

    await renderWithI18n(
      <RouteForm route={routeWithHeaders} certificates={[]} onSubmit={vi.fn()} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    // The header values should be rendered in inputs
    expect(screen.getByDisplayValue('X-Custom-Token')).toBeInTheDocument()
    expect(screen.getByDisplayValue('secret123')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Cookie')).toBeInTheDocument()
    expect(screen.getByDisplayValue('X-Request-Id')).toBeInTheDocument()
    expect(screen.getByDisplayValue('req-42')).toBeInTheDocument()
    expect(screen.getByDisplayValue('X-Powered-By')).toBeInTheDocument()
  })

  it('submits cleaned header manipulation fields on save', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    const routeWithHeaders: Route = {
      ...billingRoute,
      set_request_headers: { 'X-Token': 'my-secret' },
    }

    await renderWithI18n(
      <RouteForm route={routeWithHeaders} certificates={[]} onSubmit={onSubmit} onCancel={vi.fn()} />,
      { locale: 'en' }
    )

    await user.click(screen.getByRole('button', { name: 'Update Route' }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          set_request_headers: { 'X-Token': 'my-secret' },
          remove_request_headers: undefined,
          add_response_headers: undefined,
          remove_response_headers: undefined,
        })
      )
    })
  })
})
